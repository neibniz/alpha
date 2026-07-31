package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	usermodel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/syncx"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestXrayUserNodeColumnsStayLightweight(t *testing.T) {
	assertColumns(t, xrayUserNodeColumns, []string{
		"id",
		"tags",
		"protocol",
		"inbound_alias",
	}, []string{
		"address",
		"server_id",
		"created_at",
		"updated_at",
	})
}

func TestXrayUserProductColumnsStayLightweight(t *testing.T) {
	assertColumns(t, xrayUserProductColumns, []string{
		"id",
		"nodes",
		"node_tags",
		"speed_limit",
		"device_limit",
	}, []string{
		"description",
		"discount",
		"inventory",
		"traffic",
		"unit_price",
		"created_at",
		"updated_at",
	})
}

func TestXrayUserSubscriptionColumnsStayLightweight(t *testing.T) {
	assertColumns(t, xrayUserSubscriptionColumns, []string{
		"id",
		"user_id",
		"subscribe_id",
		"product_snapshot",
		"uuid",
		"expire_time",
		"traffic",
		"upload",
		"download",
	}, []string{
		"token",
		"note",
		"product_name",
		"order_id",
		"status",
		"created_at",
		"updated_at",
	})
}

func TestXrayUserColumnsStayLightweight(t *testing.T) {
	assertColumns(t, xrayUserColumns, []string{
		"id",
		"enable",
		"deleted_at",
	}, []string{
		"password",
		"salt",
		"rules",
		"balance",
		"commission",
		"avatar",
	})
}

func TestResolveInboundTagsUsesRenderedConfigTag(t *testing.T) {
	enabled := true
	tags, err := resolveInboundTagRefs(&node.Server{Id: 1, Name: "sg", Address: "127.0.0.1"}, []*node.ServerXrayTemplate{
		{
			Alias: "VLESS Reality gRPC",
			Template: &node.XrayTemplate{
				Type: node.XrayTemplateInbound,
				Config: `{
					"tag": "vless-reality-grpc",
					"listen": "0.0.0.0",
					"port": 32001,
					"protocol": "vless",
					"settings": {
						"clients": [{"email": "template", "flow": "", "id": "00000000-0000-0000-0000-000000000000"}],
						"decryption": "none"
					},
					"streamSettings": {
						"network": "grpc",
						"security": "reality",
						"grpcSettings": {"serviceName": "shop"},
						"realitySettings": {
							"target": "ebay.com:443",
							"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
							"serverNames": ["ebay.com", "www.ebay.com"],
							"shortIds": ["0123456789abcdef"]
						}
					}
				}`,
				DefaultVariables: "{}",
				Enabled:          &enabled,
			},
			Variables: "{}",
			Enabled:   &enabled,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"VLESS Reality gRPC", "VLESS Reality gRPC Inbound", "vless-reality-grpc"} {
		ref, ok := tags.Find(key)
		if !ok || ref.Tag != "vless-reality-grpc" {
			t.Fatalf("tags[%q] = %#v, want rendered config tag", key, ref)
		}
	}
}

func TestInboundTagRefIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newInboundTagRefIndex()
	index.Set("VLESS", inboundTagRef{Tag: "vless-in", Sort: 2})
	index.Set("Trojan", inboundTagRef{Tag: "trojan-in", Sort: 1})
	index.Set("VLESS", inboundTagRef{Tag: "vless-updated", Sort: 3})

	if index.byAlias != nil {
		t.Fatal("small inbound tag ref index should not allocate map")
	}
	if index.Len() != 2 {
		t.Fatalf("index length = %d, want 2", index.Len())
	}
	if ref, ok := index.Find(" VLESS "); !ok || ref.Tag != "vless-updated" || ref.Sort != 3 {
		t.Fatalf("Find(VLESS) = %#v, %v", ref, ok)
	}
	if _, ok := index.Find("missing"); ok {
		t.Fatal("missing alias should not be found")
	}
}

func TestInboundTagRefIndexUpgradesLargeInputToMap(t *testing.T) {
	index := newInboundTagRefIndex()
	for i := 1; i <= smallXrayUniqueThreshold+1; i++ {
		alias := "alias-" + strconv.Itoa(i)
		index.Set(alias, inboundTagRef{Tag: "tag-" + strconv.Itoa(i), Sort: i})
	}
	if index.byAlias == nil {
		t.Fatal("large inbound tag ref index should allocate map")
	}
	lastAlias := "alias-" + strconv.Itoa(smallXrayUniqueThreshold+1)
	ref, ok := index.Find(lastAlias)
	if !ok || ref.Tag != "tag-"+strconv.Itoa(smallXrayUniqueThreshold+1) {
		t.Fatalf("Find(%s) = %#v, %v", lastAlias, ref, ok)
	}
}

func TestInboundTagRefIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 40
	index := newInboundTagRefIndexWithHint(capHint)
	if index.rows != nil || index.byAlias != nil {
		t.Fatalf("large hinted inbound tag ref index should not allocate before first alias: %#v", index)
	}
	index.Set("VLESS", inboundTagRef{Tag: "vless-in", Sort: 2})
	index.Set("VLESS", inboundTagRef{Tag: "vless-updated", Sort: 3})
	if index.byAlias != nil || index.Len() != 1 {
		t.Fatalf("duplicate hinted inbound tag ref index should stay linear: %#v", index)
	}
	if ref, ok := index.Find("VLESS"); !ok || ref.Tag != "vless-updated" {
		t.Fatalf("Find(VLESS) = %#v, %v", ref, ok)
	}
	for i := 2; i <= smallXrayUniqueThreshold+1; i++ {
		index.Set("alias-"+strconv.Itoa(i), inboundTagRef{Tag: "tag-" + strconv.Itoa(i), Sort: i})
	}
	if index.byAlias == nil {
		t.Fatal("hinted inbound tag ref index should allocate map after actual unique aliases exceed threshold")
	}
	if maxCap := index.Len() * 2; cap(index.rows) > maxCap {
		t.Fatalf("inbound tag ref index cap = %d, want <= %d", cap(index.rows), maxCap)
	}
}

func TestResolveAllowedInboundTagsUsesCurrentProductNodes(t *testing.T) {
	nodes := []node.Node{
		{Id: 1, Protocol: "vless", InboundAlias: "VLESS Reality gRPC", Tags: "premium"},
		{Id: 2, Protocol: "hysteria2", InboundAlias: "Hysteria2", Tags: "premium"},
	}
	refs := map[string]inboundTagRef{
		"VLESS Reality gRPC": {Tag: "vless-reality-grpc", Sort: 2},
		"Hysteria2":          {Tag: "hysteria2", Sort: 1},
	}
	index := newXrayInboundNodeIndex(nodes, refs)

	tags := resolveAllowedInboundTags(&subscribe.Subscribe{Nodes: "1"}, index)
	if len(tags) != 1 || tags[0] != "vless-reality-grpc" {
		t.Fatalf("tags = %#v, want only explicitly selected inbound", tags)
	}

	tags = resolveAllowedInboundTags(&subscribe.Subscribe{NodeTags: "premium"}, index)
	if len(tags) != 2 || tags[0] != "hysteria2" || tags[1] != "vless-reality-grpc" {
		t.Fatalf("tags = %#v, want all current tag-matched inbounds sorted by binding sort", tags)
	}
}

func TestResolveAllowedInboundTagsDeduplicatesIndexedNodes(t *testing.T) {
	nodes := []node.Node{
		{Id: 1, Protocol: "vless", InboundAlias: "VLESS", Tags: "premium, shared"},
		{Id: 2, Protocol: "vless", InboundAlias: "VLESS", Tags: "premium"},
		{Id: 3, Protocol: "trojan", InboundAlias: "Trojan", Tags: "shared"},
	}
	index := newXrayInboundNodeIndex(nodes, map[string]inboundTagRef{
		"VLESS":  {Tag: "vless-in", Sort: 2},
		"Trojan": {Tag: "trojan-in", Sort: 1},
	})

	tags := resolveAllowedInboundTags(&subscribe.Subscribe{Nodes: "1, 1, bad, 0", NodeTags: "premium, premium,shared, "}, index)
	if len(tags) != 2 || tags[0] != "trojan-in" || tags[1] != "vless-in" {
		t.Fatalf("tags = %#v, want unique indexed inbounds sorted by binding sort", tags)
	}
}

func TestXrayInboundNodeIndexDeduplicatesTagRefs(t *testing.T) {
	index := newXrayInboundNodeIndex([]node.Node{
		{Id: 1, Protocol: "vless", InboundAlias: "VLESS", Tags: "premium, premium, shared"},
		{Id: 2, Protocol: "vless", InboundAlias: "VLESS", Tags: "premium"},
		{Id: 3, Protocol: "trojan", InboundAlias: "Trojan", Tags: "premium, shared"},
	}, map[string]inboundTagRef{
		"VLESS":  {Tag: "vless-in", Sort: 2},
		"Trojan": {Tag: "trojan-in", Sort: 1},
	})

	refs := index.RefsByTag("premium")
	if len(refs) != 2 {
		t.Fatalf("premium refs = %#v, want two unique inbound refs", refs)
	}
	if refs[0].Tag != "vless-in" || refs[1].Tag != "trojan-in" {
		t.Fatalf("premium refs = %#v, want vless and trojan refs in first-seen order", refs)
	}
	refs = index.RefsByTag("shared")
	if len(refs) != 2 {
		t.Fatalf("shared refs = %#v, want two inbound refs", refs)
	}
	if refs[0].Tag != "vless-in" || refs[1].Tag != "trojan-in" {
		t.Fatalf("shared refs = %#v, want both inbound refs without collapsing across node tags", refs)
	}
}

func TestXrayInboundNodeIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newXrayInboundNodeIndex([]node.Node{
		{Id: 1, InboundAlias: "VLESS", Tags: "premium"},
		{Id: 2, InboundAlias: "Trojan", Tags: "premium,edge"},
	}, map[string]inboundTagRef{
		"VLESS":  {Tag: "vless-in", Sort: 2},
		"Trojan": {Tag: "trojan-in", Sort: 1},
	})
	if index.byID != nil || index.byTag != nil {
		t.Fatalf("small inbound node index should not allocate maps: %#v", index)
	}
	if ref, ok := index.RefByID(2); !ok || ref.Tag != "trojan-in" {
		t.Fatalf("inbound node index RefByID(2) = %#v, ok=%v", ref, ok)
	}
	if refs := index.RefsByTag("premium"); len(refs) != 2 {
		t.Fatalf("premium refs = %#v, want 2 refs", refs)
	}
	if refs := index.RefsByTag("edge"); len(refs) != 1 || refs[0].Tag != "trojan-in" {
		t.Fatalf("edge refs = %#v, want trojan ref", refs)
	}
}

func TestXrayInboundNodeIndexUpgradesLargeInputToMaps(t *testing.T) {
	nodes := make([]node.Node, 0, smallXrayUniqueThreshold+1)
	refs := make(map[string]inboundTagRef, smallXrayUniqueThreshold+1)
	for id := int64(1); id <= smallXrayUniqueThreshold+1; id++ {
		alias := "inbound-" + strconv.FormatInt(id, 10)
		nodes = append(nodes, node.Node{Id: id, InboundAlias: alias, Tags: "tag-" + strconv.FormatInt(id, 10)})
		refs[alias] = inboundTagRef{Tag: "xray-" + strconv.FormatInt(id, 10), Sort: int(id)}
	}
	index := newXrayInboundNodeIndex(nodes, refs)
	if index.byID == nil || index.byTag == nil {
		t.Fatalf("large inbound node index should allocate maps: %#v", index)
	}
	lastID := int64(smallXrayUniqueThreshold + 1)
	if ref, ok := index.RefByID(lastID); !ok || ref.Tag != "xray-"+strconv.FormatInt(lastID, 10) {
		t.Fatalf("inbound node index RefByID(last) = %#v, ok=%v", ref, ok)
	}
	refsByTag := index.RefsByTag("tag-" + strconv.FormatInt(lastID, 10))
	if len(refsByTag) != 1 || refsByTag[0].Tag != "xray-"+strconv.FormatInt(lastID, 10) {
		t.Fatalf("inbound node index RefsByTag(last) = %#v", refsByTag)
	}
}

func TestXrayInboundNodeIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 40
	index := newXrayInboundNodeIndexWithHint(capHint)
	if index.idRows != nil || index.tagRows != nil || index.byID != nil || index.byTag != nil {
		t.Fatalf("large hinted inbound node index should not allocate before first ref: %#v", index)
	}
	index.SetNodeRef(7, inboundTagRef{Tag: "vless-in", Sort: 2})
	index.SetNodeRef(7, inboundTagRef{Tag: "vless-updated", Sort: 3})
	if index.byID != nil || len(index.idRows) != 1 {
		t.Fatalf("duplicate hinted inbound node id index should stay linear: %#v", index)
	}
	if ref, ok := index.RefByID(7); !ok || ref.Tag != "vless-updated" {
		t.Fatalf("RefByID(7) = %#v, %v", ref, ok)
	}
	for id := int64(8); id <= int64(smallXrayUniqueThreshold+7); id++ {
		index.SetNodeRef(id, inboundTagRef{Tag: "tag-" + strconv.FormatInt(id, 10), Sort: int(id)})
	}
	if index.byID == nil {
		t.Fatal("hinted inbound node id index should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := len(index.idRows) * 2; cap(index.idRows) > maxCap {
		t.Fatalf("inbound node id row cap = %d, want <= %d", cap(index.idRows), maxCap)
	}
	index.AddTagRef("premium", inboundTagRef{Tag: "vless-updated", Sort: 3})
	index.AddTagRef("premium", inboundTagRef{Tag: "trojan-in", Sort: 1})
	if index.byTag != nil || len(index.tagRows) != 1 {
		t.Fatalf("duplicate hinted inbound node tag index should stay linear: %#v", index)
	}
	if refs := index.RefsByTag("premium"); len(refs) != 2 {
		t.Fatalf("RefsByTag(premium) = %#v, want two refs", refs)
	}
	for i := 2; i <= smallXrayUniqueThreshold+1; i++ {
		index.AddTagRef("tag-"+strconv.Itoa(i), inboundTagRef{Tag: "inbound-" + strconv.Itoa(i), Sort: i})
	}
	if index.byTag == nil {
		t.Fatal("hinted inbound node tag index should allocate map after actual unique tags exceed threshold")
	}
	if maxCap := len(index.tagRows) * 2; cap(index.tagRows) > maxCap {
		t.Fatalf("inbound node tag row cap = %d, want <= %d", cap(index.tagRows), maxCap)
	}
}

func TestXrayInboundNodeIndexAddTagRefUpdatesDuplicateInboundRefs(t *testing.T) {
	index := &xrayInboundNodeIndex{}
	index.AddTagRef("premium", inboundTagRef{Tag: "vless-in", Sort: 2})
	index.AddTagRef("premium", inboundTagRef{Tag: "vless-in", Sort: 1})
	index.AddTagRef("premium", inboundTagRef{Tag: "trojan-in", Sort: 3})

	refs := index.RefsByTag("premium")
	if len(refs) != 2 {
		t.Fatalf("linear premium refs = %#v, want two unique inbound refs", refs)
	}
	if refs[0].Tag != "vless-in" || refs[0].Sort != 1 {
		t.Fatalf("duplicate linear ref should be updated in place: %#v", refs)
	}

	indexed := newXrayInboundNodeIndexWithHint(smallXrayUniqueThreshold + 1)
	indexed.AddTagRef("premium", inboundTagRef{Tag: "vless-in", Sort: 2})
	indexed.AddTagRef("premium", inboundTagRef{Tag: "vless-in", Sort: 1})
	if indexed.byTag != nil {
		t.Fatal("large hinted inbound node tag index should not allocate map for one unique tag")
	}
	refs = indexed.RefsByTag("premium")
	if len(refs) != 1 || refs[0].Tag != "vless-in" || refs[0].Sort != 1 {
		t.Fatalf("hinted duplicate ref should be updated in place: %#v", refs)
	}
}

func TestInboundTagKeySetUsesLinearDedupForSmallInput(t *testing.T) {
	var set inboundTagKeySet
	first := inboundTagKey{NodeTag: "premium", InboundTag: "vless-in"}
	second := inboundTagKey{NodeTag: "shared", InboundTag: "vless-in"}

	if !set.Add(first) || !set.Add(second) {
		t.Fatalf("expected new inbound tag keys to be added: %#v", set)
	}
	if set.Add(first) {
		t.Fatalf("duplicate inbound tag key should not be added: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small inbound tag key set should not allocate map")
	}
	if len(set.items) != 2 {
		t.Fatalf("set items = %#v, want two unique keys", set.items)
	}
}

func TestInboundTagKeySetUpgradesLargeInputToMap(t *testing.T) {
	var set inboundTagKeySet
	for i := 1; i <= smallXrayUniqueThreshold+1; i++ {
		key := inboundTagKey{NodeTag: "tag-" + strconv.Itoa(i), InboundTag: "vless-in"}
		if !set.Add(key) {
			t.Fatalf("expected key %d to be added", i)
		}
	}
	if set.Add(inboundTagKey{NodeTag: "tag-1", InboundTag: "vless-in"}) {
		t.Fatal("duplicate inbound tag key should not be added after map upgrade")
	}
	if set.index == nil {
		t.Fatal("large inbound tag key set should upgrade to map")
	}
	if len(set.items) != smallXrayUniqueThreshold+1 {
		t.Fatalf("set length = %d, want %d", len(set.items), smallXrayUniqueThreshold+1)
	}
}

func TestInboundTagKeySetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := newInboundTagKeySet(smallXrayUniqueThreshold + 40)
	if set.items != nil || set.index != nil {
		t.Fatalf("large hinted inbound tag key set should not allocate before first key: %#v", set)
	}
	key := inboundTagKey{NodeTag: "premium", InboundTag: "vless-in"}
	if !set.Add(key) {
		t.Fatal("expected first inbound tag key to be added")
	}
	if set.Add(key) {
		t.Fatal("duplicate inbound tag key should not be added")
	}
	if len(set.items) != 1 || set.index != nil {
		t.Fatalf("duplicate hinted inbound tag key set should stay linear: %#v", set)
	}
	for i := 2; i <= smallXrayUniqueThreshold+1; i++ {
		if !set.Add(inboundTagKey{NodeTag: "tag-" + strconv.Itoa(i), InboundTag: "vless-in"}) {
			t.Fatalf("expected key %d to be added", i)
		}
	}
	if set.index == nil {
		t.Fatal("hinted inbound tag key set should allocate map after actual unique keys exceed threshold")
	}
	if maxCap := len(set.items) * 2; cap(set.items) > maxCap {
		t.Fatalf("inbound tag key set cap = %d, want <= %d", cap(set.items), maxCap)
	}
}

func TestCollectXrayNodeScopesDeduplicatesTags(t *testing.T) {
	nodeIDs, nodeTags := collectXrayNodeScopes([]node.Node{
		{Id: 3, Tags: "premium, shared"},
		{Id: 4, Tags: "premium,, edge "},
		{Id: 3, Tags: "edge"},
		{Id: 0, Tags: "ignored-id"},
		{Id: 5},
	})
	if len(nodeIDs) != 3 || nodeIDs[0] != 3 || nodeIDs[1] != 4 || nodeIDs[2] != 5 {
		t.Fatalf("node ids = %#v, want positive unique ids in node order", nodeIDs)
	}
	wantTags := []string{"premium", "shared", "edge", "ignored-id"}
	if len(nodeTags) != len(wantTags) {
		t.Fatalf("node tags = %#v, want %#v", nodeTags, wantTags)
	}
	for i := range wantTags {
		if nodeTags[i] != wantTags[i] {
			t.Fatalf("node tags = %#v, want %#v", nodeTags, wantTags)
		}
	}
}

func TestCollectXrayNodeScopesSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	nodes := make([]node.Node, viewutil.SmallRowsCapacityThreshold+20)
	nodes[0] = node.Node{Id: 7, Tags: "edge"}

	nodeIDs, nodeTags := collectXrayNodeScopes(nodes)
	if len(nodeIDs) != 1 || nodeIDs[0] != 7 {
		t.Fatalf("node ids = %#v, want [7]", nodeIDs)
	}
	if cap(nodeIDs) != smallXrayUniqueThreshold+1 {
		t.Fatalf("invalid-heavy node ids cap = %d, want %d", cap(nodeIDs), smallXrayUniqueThreshold+1)
	}
	if len(nodeTags) != 1 || nodeTags[0] != "edge" {
		t.Fatalf("node tags = %#v, want [edge]", nodeTags)
	}
	if cap(nodeTags) != smallXrayUniqueThreshold+1 {
		t.Fatalf("invalid-heavy node tags cap = %d, want %d", cap(nodeTags), smallXrayUniqueThreshold+1)
	}
}

func TestCollectXrayNodeScopesSmallInputAvoidsSetAllocation(t *testing.T) {
	nodes := []node.Node{
		{Id: 3, Tags: "premium, shared"},
		{Id: 4, Tags: "premium"},
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = collectXrayNodeScopes(nodes)
	})
	if allocs > 2 {
		t.Fatalf("collectXrayNodeScopes small input allocations = %f, want at most 2", allocs)
	}
}

func TestIndexXrayProductsSkipsNilAndKeepsIDs(t *testing.T) {
	index := indexXrayProducts([]*subscribe.Subscribe{
		{Id: 7, SpeedLimit: 100},
		nil,
		{Id: 0, SpeedLimit: 999},
		{Id: 9, SpeedLimit: 200},
		{Id: 7, SpeedLimit: 300},
	})
	if index.byID != nil {
		t.Fatal("small product index should not allocate map")
	}
	if index.Len() != 2 || index.Get(7).SpeedLimit != 100 || index.Get(9).SpeedLimit != 200 || index.Get(100) != nil {
		t.Fatalf("product index = %#v", index)
	}
	productIDs := index.IDs()
	if len(productIDs) != 2 || productIDs[0] != 7 || productIDs[1] != 9 {
		t.Fatalf("product ids = %#v, want [7 9]", productIDs)
	}
}

func TestIndexXrayProductsUpgradesLargeInputToMap(t *testing.T) {
	products := make([]*subscribe.Subscribe, 0, smallXrayUniqueThreshold+1)
	for id := int64(1); id <= smallXrayUniqueThreshold+1; id++ {
		products = append(products, &subscribe.Subscribe{Id: id, SpeedLimit: id * 10})
	}
	index := indexXrayProducts(products)
	if cap(index.products) != smallXrayUniqueThreshold+1 {
		t.Fatalf("product index cap = %d, want %d", cap(index.products), smallXrayUniqueThreshold+1)
	}
	if index.byID == nil {
		t.Fatal("large product index should allocate map")
	}
	if index.Len() != smallXrayUniqueThreshold+1 {
		t.Fatalf("product index len = %d, want %d", index.Len(), smallXrayUniqueThreshold+1)
	}
	lastID := int64(smallXrayUniqueThreshold + 1)
	if got := index.Get(lastID); got == nil || got.SpeedLimit != lastID*10 {
		t.Fatalf("product index get(last) = %#v", got)
	}
}

func TestXrayProductIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	index := xrayProductIndex{
		capHint: smallXrayUniqueThreshold + 40,
	}
	if index.products != nil || index.ids != nil || index.byID != nil {
		t.Fatalf("large hinted product index should not allocate before first product: %#v", index)
	}
	index.Add(&subscribe.Subscribe{Id: 7, SpeedLimit: 100})
	index.Add(&subscribe.Subscribe{Id: 7, SpeedLimit: 999})
	if index.byID != nil || index.Len() != 1 || index.Get(7).SpeedLimit != 100 {
		t.Fatalf("duplicate hinted product index should stay linear: %#v", index)
	}
	for id := int64(8); id <= int64(smallXrayUniqueThreshold+7); id++ {
		index.Add(&subscribe.Subscribe{Id: id, SpeedLimit: id * 10})
	}
	if index.byID == nil {
		t.Fatal("hinted product index should allocate map after actual unique products exceed threshold")
	}
	if maxCap := index.Len() * 2; cap(index.products) > maxCap || cap(index.ids) > maxCap {
		t.Fatalf("product index caps = %d/%d, want <= %d", cap(index.products), cap(index.ids), maxCap)
	}
}

func TestXrayAllowedInboundTagCacheUsesLinearPathForSmallInput(t *testing.T) {
	var cache xrayAllowedInboundTagCache
	cache.Set(7, []string{"vless"})
	cache.Set(9, nil)
	cache.Set(7, []string{"trojan"})

	if cache.byProduct != nil {
		t.Fatal("small allowed inbound tag cache should not allocate map")
	}
	if cache.Len() != 2 {
		t.Fatalf("allowed inbound tag cache len = %d, want 2", cache.Len())
	}
	if got, ok := cache.Get(7); !ok || len(got) != 1 || got[0] != "trojan" {
		t.Fatalf("allowed inbound tag cache get(7) = %#v, ok=%v", got, ok)
	}
	if got, ok := cache.Get(9); !ok || got != nil {
		t.Fatalf("allowed inbound tag cache should cache empty tags: %#v, ok=%v", got, ok)
	}
	if got, ok := cache.Get(99); ok || got != nil {
		t.Fatalf("allowed inbound tag cache get(99) = %#v, ok=%v", got, ok)
	}
}

func TestXrayAllowedInboundTagCacheUpgradesLargeInputToMap(t *testing.T) {
	var cache xrayAllowedInboundTagCache
	for id := int64(1); id <= smallXrayUniqueThreshold+1; id++ {
		cache.Set(id, []string{"tag-" + strconv.FormatInt(id, 10)})
	}
	if cache.byProduct == nil {
		t.Fatal("large allowed inbound tag cache should allocate map")
	}
	if cache.Len() != smallXrayUniqueThreshold+1 {
		t.Fatalf("allowed inbound tag cache len = %d, want %d", cache.Len(), smallXrayUniqueThreshold+1)
	}
	lastID := int64(smallXrayUniqueThreshold + 1)
	if got, ok := cache.Get(lastID); !ok || len(got) != 1 || got[0] != "tag-"+strconv.FormatInt(lastID, 10) {
		t.Fatalf("allowed inbound tag cache get(last) = %#v, ok=%v", got, ok)
	}
}

func TestXrayAllowedInboundTagCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newXrayAllowedInboundTagCache(smallXrayUniqueThreshold + 40)
	if cache.rows != nil || cache.byProduct != nil {
		t.Fatalf("large hinted allowed inbound tag cache should not allocate before first product: %#v", cache)
	}
	cache.Set(7, []string{"vless"})
	cache.Set(7, []string{"trojan"})
	if cache.byProduct != nil || cache.Len() != 1 {
		t.Fatalf("duplicate hinted allowed inbound tag cache should stay linear: %#v", cache)
	}
	if tags, ok := cache.Get(7); !ok || len(tags) != 1 || tags[0] != "trojan" {
		t.Fatalf("allowed inbound tag cache get(7) = %#v, ok=%v", tags, ok)
	}
	for id := int64(8); id <= int64(smallXrayUniqueThreshold+7); id++ {
		cache.Set(id, []string{"tag-" + strconv.FormatInt(id, 10)})
	}
	if cache.byProduct == nil {
		t.Fatal("hinted allowed inbound tag cache should allocate map after actual unique products exceed threshold")
	}
	if maxCap := cache.Len() * 2; cap(cache.rows) > maxCap {
		t.Fatalf("allowed inbound tag cache cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestCollectXrayUserIDsDeduplicatesAndSkipsInvalidSubscriptions(t *testing.T) {
	got := collectXrayUserIDs([]*usermodel.Subscribe{
		nil,
		{UserId: 0},
		{UserId: 5},
		{UserId: 7},
		{UserId: 5},
	})
	want := []int64{5, 7}
	if len(got) != len(want) {
		t.Fatalf("user ids = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("user ids = %#v, want %#v", got, want)
		}
	}
	if cap(got) != 5 {
		t.Fatalf("user ids cap = %d, want 5", cap(got))
	}
}

func TestCollectXrayUserIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	subs := make([]*usermodel.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	subs[0] = &usermodel.Subscribe{UserId: 7}

	got := collectXrayUserIDs(subs)
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("user ids = %#v, want [7]", got)
	}
	if cap(got) != smallXrayUniqueThreshold+1 {
		t.Fatalf("invalid-heavy user ids cap = %d, want %d", cap(got), smallXrayUniqueThreshold+1)
	}
}

func TestCollectXrayUserIDsSmallInputAvoidsSetAllocation(t *testing.T) {
	subs := []*usermodel.Subscribe{{UserId: 5}, {UserId: 7}, {UserId: 5}}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = collectXrayUserIDs(subs)
	})
	if allocs > 1 {
		t.Fatalf("collectXrayUserIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestAppendUniqueXrayInt64UsesCapacityHintForLargeInput(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 1
	values := make([]int64, 0, xrayUniqueInitialCap(capHint))
	var seen map[int64]struct{}
	values, seen = appendUniqueXrayInt64(values, 7, seen, capHint)
	if seen != nil {
		t.Fatal("large hinted xray int64 set should not allocate map for one unique value")
	}
	values, seen = appendUniqueXrayInt64(values, 7, seen, capHint)
	if len(values) != 1 || seen != nil {
		t.Fatalf("duplicate hinted xray int64 set should stay linear: values=%#v seen=%#v", values, seen)
	}
	for id := int64(8); id <= int64(smallXrayUniqueThreshold+7); id++ {
		values, seen = appendUniqueXrayInt64(values, id, seen, capHint)
	}
	if seen == nil {
		t.Fatal("hinted xray int64 set should allocate map after actual unique values exceed threshold")
	}
}

func TestAppendUniqueXrayInt64BoundsSparseUpgradeCapacity(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 40
	values := make([]int64, 0, xrayUniqueInitialCap(capHint))
	var seen map[int64]struct{}
	for id := int64(1); id <= int64(smallXrayUniqueThreshold+1); id++ {
		values, seen = appendUniqueXrayInt64(values, id, seen, capHint)
	}
	for index := 0; index < capHint; index++ {
		values, seen = appendUniqueXrayInt64(values, 1, seen, capHint)
	}

	if seen == nil {
		t.Fatal("xray int64 set should allocate map after actual unique values exceed threshold")
	}
	if len(values) != smallXrayUniqueThreshold+1 {
		t.Fatalf("xray int64 values len = %d, want %d", len(values), smallXrayUniqueThreshold+1)
	}
	if maxCap := (smallXrayUniqueThreshold + 1) * 2; cap(values) > maxCap {
		t.Fatalf("xray int64 values cap = %d, want <= %d", cap(values), maxCap)
	}
}

func TestAppendUniqueXrayStringUsesCapacityHintForLargeInput(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 1
	values := make([]string, 0, xrayUniqueInitialCap(capHint))
	var seen map[string]struct{}
	values, seen = appendUniqueXrayString(values, "premium", seen, capHint)
	if seen != nil {
		t.Fatal("large hinted xray string set should not allocate map for one unique value")
	}
	values, seen = appendUniqueXrayString(values, "premium", seen, capHint)
	if len(values) != 1 || seen != nil {
		t.Fatalf("duplicate hinted xray string set should stay linear: values=%#v seen=%#v", values, seen)
	}
	for i := 2; i <= smallXrayUniqueThreshold+1; i++ {
		values, seen = appendUniqueXrayString(values, "tag-"+strconv.Itoa(i), seen, capHint)
	}
	if seen == nil {
		t.Fatal("hinted xray string set should allocate map after actual unique values exceed threshold")
	}
}

func TestAppendUniqueXrayStringBoundsSparseUpgradeCapacity(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 40
	values := make([]string, 0, xrayUniqueInitialCap(capHint))
	var seen map[string]struct{}
	for index := 1; index <= smallXrayUniqueThreshold+1; index++ {
		values, seen = appendUniqueXrayString(values, "tag-"+strconv.Itoa(index), seen, capHint)
	}
	for index := 0; index < capHint; index++ {
		values, seen = appendUniqueXrayString(values, "tag-1", seen, capHint)
	}

	if seen == nil {
		t.Fatal("xray string set should allocate map after actual unique values exceed threshold")
	}
	if len(values) != smallXrayUniqueThreshold+1 {
		t.Fatalf("xray string values len = %d, want %d", len(values), smallXrayUniqueThreshold+1)
	}
	if maxCap := (smallXrayUniqueThreshold + 1) * 2; cap(values) > maxCap {
		t.Fatalf("xray string values cap = %d, want <= %d", cap(values), maxCap)
	}
}

func TestIndexXrayUsersUsesLinearPathForSmallInput(t *testing.T) {
	index := indexXrayUsers([]*usermodel.User{
		{Id: 5},
		nil,
		{Id: 7},
		{Id: 5, Enable: boolPtr(false)},
	})
	if index.byID != nil {
		t.Fatal("small user index should not allocate map")
	}
	if index.Len() != 2 {
		t.Fatalf("user index len = %d, want 2", index.Len())
	}
	if got := index.Get(5); got == nil || got.Id != 5 || got.Enable != nil {
		t.Fatalf("user index get(5) = %#v, want first user", got)
	}
	if got := index.Get(99); got != nil {
		t.Fatalf("user index get(99) = %#v, want nil", got)
	}
}

func TestIndexXrayUsersUpgradesLargeInputToMap(t *testing.T) {
	users := make([]*usermodel.User, 0, smallXrayUniqueThreshold+1)
	for id := int64(1); id <= smallXrayUniqueThreshold+1; id++ {
		users = append(users, &usermodel.User{Id: id})
	}
	index := indexXrayUsers(users)
	if cap(index.users) != smallXrayUniqueThreshold+1 {
		t.Fatalf("user index cap = %d, want %d", cap(index.users), smallXrayUniqueThreshold+1)
	}
	if index.byID == nil {
		t.Fatal("large user index should allocate map")
	}
	if index.Len() != smallXrayUniqueThreshold+1 {
		t.Fatalf("user index len = %d, want %d", index.Len(), smallXrayUniqueThreshold+1)
	}
	lastID := int64(smallXrayUniqueThreshold + 1)
	if got := index.Get(lastID); got == nil || got.Id != lastID {
		t.Fatalf("user index get(last) = %#v", got)
	}
}

func TestXrayUserIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	index := xrayUserIndex{
		capHint: smallXrayUniqueThreshold + 40,
	}
	if index.users != nil || index.byID != nil {
		t.Fatalf("large hinted user index should not allocate before first user: %#v", index)
	}
	index.Add(&usermodel.User{Id: 5})
	disabled := false
	index.Add(&usermodel.User{Id: 5, Enable: &disabled})
	if index.byID != nil || index.Len() != 1 || index.Get(5).Enable != nil {
		t.Fatalf("duplicate hinted user index should stay linear: %#v", index)
	}
	for id := int64(6); id <= int64(smallXrayUniqueThreshold+5); id++ {
		index.Add(&usermodel.User{Id: id})
	}
	if index.byID == nil {
		t.Fatal("hinted user index should allocate map after actual unique users exceed threshold")
	}
	if maxCap := index.Len() * 2; cap(index.users) > maxCap {
		t.Fatalf("user index cap = %d, want <= %d", cap(index.users), maxCap)
	}
}

func TestAppendUniqueInboundRefSmallInputAvoidsSetAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		var refs []inboundTagRef
		var seen map[string]struct{}
		refs, seen = appendUniqueInboundRef(refs, inboundTagRef{Tag: "a"}, seen, 3)
		refs, seen = appendUniqueInboundRef(refs, inboundTagRef{Tag: "b"}, seen, 3)
		refs, seen = appendUniqueInboundRef(refs, inboundTagRef{Tag: "a"}, seen, 3)
		_ = refs
		_ = seen
	})
	if allocs > 1 {
		t.Fatalf("appendUniqueInboundRef small input allocations = %f, want at most 1", allocs)
	}
}

func TestAppendUniqueInboundRefBoundsSparseCapacity(t *testing.T) {
	capHint := smallXrayUniqueThreshold + 40
	var refs []inboundTagRef
	var seen map[string]struct{}
	for index := 0; index <= smallXrayUniqueThreshold; index++ {
		refs, seen = appendUniqueInboundRef(refs, inboundTagRef{Tag: "tag-" + strconv.Itoa(index)}, seen, capHint)
	}
	for index := 0; index < capHint; index++ {
		refs, seen = appendUniqueInboundRef(refs, inboundTagRef{Tag: "tag-0"}, seen, capHint)
	}
	if seen == nil {
		t.Fatal("inbound refs should allocate set after actual unique refs exceed threshold")
	}
	if maxCap := len(refs) * 2; cap(refs) > maxCap {
		t.Fatalf("inbound refs cap = %d, want <= %d", cap(refs), maxCap)
	}
}

func TestBuildXrayUserRowsAppliesSnapshotAndSkipsInvalidRows(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	enabled := true
	index := newXrayInboundNodeIndex([]node.Node{
		{Id: 1, Protocol: "vless", InboundAlias: "VLESS", Tags: "premium"},
	}, map[string]inboundTagRef{
		"VLESS": {Tag: "vless-in", Sort: 1},
	})

	got := buildXrayUserRows([]*usermodel.Subscribe{
		{
			Id:          11,
			UserId:      5,
			SubscribeId: 7,
			UUID:        "uuid-11",
			ExpireTime:  now.Add(time.Hour),
			ProductSnapshot: ordersnapshot.Marshal(ordersnapshot.Snapshot{
				Kind:        ordersnapshot.KindSubscribe,
				Name:        "Snapshot Plan",
				SpeedLimit:  300,
				DeviceLimit: 8,
			}),
		},
		nil,
		{Id: 12, UserId: 5, SubscribeId: 7},
	}, xrayUserBuildContext{
		productIndex: indexXrayProducts([]*subscribe.Subscribe{
			{Id: 7, Nodes: "1", SpeedLimit: 100, DeviceLimit: 2},
		}),
		inboundNodeIndex: index,
		userIndex:        indexXrayUsers([]*usermodel.User{{Id: 5, Enable: &enabled}}),
		now:              now,
	})

	if len(got) != 1 {
		t.Fatalf("xray users = %d, want 1: %#v", len(got), got)
	}
	if cap(got) != 3 {
		t.Fatalf("xray users capacity = %d, want 3", cap(got))
	}
	user := got[0]
	if user.UserSubscribeID != 11 || user.UserID != 5 || user.Email != "u_11" || user.UUID != "uuid-11" || user.Password != "uuid-11" {
		t.Fatalf("xray user identity fields not mapped: %#v", user)
	}
	if len(user.Inbounds) != 1 || user.Inbounds[0] != "vless-in" {
		t.Fatalf("xray user inbounds = %#v, want [vless-in]", user.Inbounds)
	}
	if user.SpeedLimit != 300 || user.DeviceLimit != 8 {
		t.Fatalf("snapshot limits not applied: %#v", user)
	}
	if !user.Enabled || user.ExpiredAt != now.Add(time.Hour).UnixMilli() {
		t.Fatalf("runtime state not mapped: %#v", user)
	}
}

func TestBuildXrayUserRowsSkipInvalidWithoutInputSizedCapacity(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	enabled := true
	index := newXrayInboundNodeIndex([]node.Node{{Id: 1, Protocol: "vless", InboundAlias: "VLESS"}}, map[string]inboundTagRef{
		"VLESS": {Tag: "vless-in", Sort: 1},
	})
	subs := make([]*usermodel.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	subs[0] = &usermodel.Subscribe{Id: 1, UserId: 5, SubscribeId: 7}
	subs[len(subs)-1] = &usermodel.Subscribe{Id: 2, UserId: 5, SubscribeId: 7, UUID: "uuid-2"}

	got := buildXrayUserRows(subs, xrayUserBuildContext{
		productIndex:     indexXrayProducts([]*subscribe.Subscribe{{Id: 7, Nodes: "1"}}),
		inboundNodeIndex: index,
		userIndex:        indexXrayUsers([]*usermodel.User{{Id: 5, Enable: &enabled}}),
		now:              now,
	})
	if len(got) != 1 {
		t.Fatalf("xray users = %d, want 1: %#v", len(got), got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy xray users cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].UserSubscribeID != 2 {
		t.Fatalf("xray user row order changed: %#v", got)
	}
}

func TestBuildXrayUserRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	enabled := true
	index := newXrayInboundNodeIndex([]node.Node{{Id: 1, Protocol: "vless", InboundAlias: "VLESS"}}, map[string]inboundTagRef{
		"VLESS": {Tag: "vless-in", Sort: 1},
	})
	subs := make([]*usermodel.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	for i := 0; i <= viewutil.SmallRowsCapacityThreshold; i++ {
		subs[i] = &usermodel.Subscribe{Id: int64(i + 1), UserId: 5, SubscribeId: 7, UUID: "uuid-" + strconv.Itoa(i+1)}
	}

	got := buildXrayUserRows(subs, xrayUserBuildContext{
		productIndex:     indexXrayProducts([]*subscribe.Subscribe{{Id: 7, Nodes: "1"}}),
		inboundNodeIndex: index,
		userIndex:        indexXrayUsers([]*usermodel.User{{Id: 5, Enable: &enabled}}),
		now:              now,
	})
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("xray users = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("xray users cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].UserSubscribeID != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("xray user row order changed: %#v", got)
	}
}

func TestBuildXrayUserRowsKeepsEmptyListShape(t *testing.T) {
	got := buildXrayUserRows(nil, xrayUserBuildContext{})
	if got == nil || len(got) != 0 {
		t.Fatalf("empty xray users should return non-nil empty slice: %#v", got)
	}
}

func TestXraySnapshotLimitsCacheReusesTrimmedSnapshot(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		Name:        "Snapshot Plan",
		SpeedLimit:  300,
		DeviceLimit: 8,
	})
	cache := newXraySnapshotLimitsCache(2)

	speed, devices, ok := cache.Limits(" " + raw + " ")
	if !ok || speed != 300 || devices != 8 {
		t.Fatalf("snapshot limits = %d/%d/%v, want 300/8/true", speed, devices, ok)
	}
	speed, devices, ok = cache.Limits(raw)
	if !ok || speed != 300 || devices != 8 {
		t.Fatalf("cached snapshot limits = %d/%d/%v, want 300/8/true", speed, devices, ok)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate trimmed snapshot should use one cache row, got %d", len(cache.rows))
	}
	if cache.byRaw != nil {
		t.Fatal("small snapshot limits cache should not allocate map")
	}
}

func TestXraySnapshotLimitsCacheCachesInvalidSnapshot(t *testing.T) {
	cache := newXraySnapshotLimitsCache(2)

	if speed, devices, ok := cache.Limits(`{"kind":`); ok || speed != 0 || devices != 0 {
		t.Fatalf("invalid snapshot limits = %d/%d/%v, want zero/false", speed, devices, ok)
	}
	if speed, devices, ok := cache.Limits(` {"kind": `); ok || speed != 0 || devices != 0 {
		t.Fatalf("trim-equivalent invalid snapshot limits = %d/%d/%v, want zero/false", speed, devices, ok)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate invalid snapshot should use one cache row, got %d", len(cache.rows))
	}
}

func TestXraySnapshotLimitsCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newXraySnapshotLimitsCache(smallXrayUniqueThreshold + 40)
	if cache.rows != nil || cache.byRaw != nil {
		t.Fatalf("large hinted snapshot limits cache should not allocate before first snapshot: %#v", cache)
	}
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		Name:        "Snapshot Plan 1",
		SpeedLimit:  1,
		DeviceLimit: 11,
	})
	cache.Limits(raw)
	cache.Limits(raw)
	if cache.byRaw != nil || len(cache.rows) != 1 {
		t.Fatalf("duplicate hinted snapshot limits cache should stay linear: %#v", cache)
	}
	for i := int64(2); i <= smallXrayUniqueThreshold+1; i++ {
		cache.Limits(ordersnapshot.Marshal(ordersnapshot.Snapshot{
			Kind:        ordersnapshot.KindSubscribe,
			Name:        "Snapshot Plan " + strconv.FormatInt(i, 10),
			SpeedLimit:  i,
			DeviceLimit: i + 10,
		}))
	}
	if cache.byRaw == nil {
		t.Fatal("snapshot limits cache should allocate map after actual unique snapshots exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("snapshot limits cache cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestXraySnapshotLimitsCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newXraySnapshotLimitsCache(smallXrayUniqueThreshold + 1)
	for i := int64(1); i <= smallXrayUniqueThreshold+1; i++ {
		raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
			Kind:        ordersnapshot.KindSubscribe,
			Name:        "Snapshot Plan " + strconv.FormatInt(i, 10),
			SpeedLimit:  i,
			DeviceLimit: i + 10,
		})
		speed, devices, ok := cache.Limits(raw)
		if !ok || speed != i || devices != i+10 {
			t.Fatalf("snapshot limits %d = %d/%d/%v", i, speed, devices, ok)
		}
	}

	if len(cache.rows) != smallXrayUniqueThreshold+1 || len(cache.byRaw) != smallXrayUniqueThreshold+1 {
		t.Fatalf("snapshot limits cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.byRaw), smallXrayUniqueThreshold+1)
	}
}

func TestXrayRuntimeLimitsUsesSnapshotLimitsCache(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		Name:        "Snapshot Plan",
		SpeedLimit:  300,
		DeviceLimit: 8,
	})
	cache := newXraySnapshotLimitsCache(2)

	speed, devices := xrayRuntimeLimitsWithCache(
		&usermodel.Subscribe{ProductSnapshot: raw},
		&subscribe.Subscribe{SpeedLimit: 100, DeviceLimit: 2},
		&cache,
	)
	if speed != 300 || devices != 8 {
		t.Fatalf("runtime limits = %d/%d, want snapshot limits 300/8", speed, devices)
	}

	speed, devices = xrayRuntimeLimitsWithCache(
		&usermodel.Subscribe{ProductSnapshot: `{bad`},
		&subscribe.Subscribe{SpeedLimit: 100, DeviceLimit: 2},
		&cache,
	)
	if speed != 100 || devices != 2 {
		t.Fatalf("invalid snapshot should fall back to product limits, got %d/%d", speed, devices)
	}
}

func TestXrayUserQueriesKeepStableEmptySlicesWithoutDatabase(t *testing.T) {
	logic := &QueryServerXrayUsersLogic{}

	products, err := logic.queryProductsByNodes(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if products == nil || len(products) != 0 {
		t.Fatalf("empty products = %#v, want non-nil empty slice", products)
	}

	subs, err := logic.queryActiveSubscriptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if subs == nil || len(subs) != 0 {
		t.Fatalf("empty active subscriptions = %#v, want non-nil empty slice", subs)
	}
}

func TestXrayRuntimeStateDisablesUnavailableUsers(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	disabled := false
	cases := []struct {
		name string
		sub  usermodel.Subscribe
		user usermodel.User
	}{
		{
			name: "expired",
			sub:  usermodel.Subscribe{ExpireTime: now.Add(-time.Minute)},
		},
		{
			name: "traffic exhausted",
			sub:  usermodel.Subscribe{Traffic: 100, Upload: 50, Download: 50},
		},
		{
			name: "deleted user",
			user: usermodel.User{DeletedAt: gorm.DeletedAt{Time: now, Valid: true}},
		},
		{
			name: "disabled user",
			user: usermodel.User{Enable: &disabled},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enabled, _ := xrayRuntimeState(&tc.sub, &tc.user, now)
			if enabled {
				t.Fatalf("runtime state should be disabled for %s", tc.name)
			}
		})
	}
}

func TestBuildXrayUsersResponseSortsAndHonorsVersion(t *testing.T) {
	users := []types.ServerXrayUser{
		{UserSubscribeID: 20, UserID: 2, UUID: "u-20", Password: "u-20", Enabled: true},
		{UserSubscribeID: 10, UserID: 1, UUID: "u-10", Password: "u-10", Enabled: true},
	}
	resp := buildXrayUsersResponse("", users)
	if !resp.Full || len(resp.Users) != 2 {
		t.Fatalf("response = %#v, want full response with users", resp)
	}
	if resp.Users[0].UserSubscribeID != 10 || resp.Users[1].UserSubscribeID != 20 {
		t.Fatalf("users were not sorted by subscription id: %#v", resp.Users)
	}
	unchanged := buildXrayUsersResponse(resp.Version, resp.Users)
	if unchanged.Full || unchanged.Users != nil {
		t.Fatalf("unchanged response = %#v, want delta response without users", unchanged)
	}
}

func TestBuildXrayUsersResponseKeepsEmptyFullUsersAsArray(t *testing.T) {
	resp := buildXrayUsersResponse("", nil)
	if !resp.Full || resp.Users == nil || len(resp.Users) != 0 {
		t.Fatalf("empty full response = %#v, want non-nil empty users", resp)
	}
	unchanged := buildXrayUsersResponse(resp.Version, nil)
	if unchanged.Full || unchanged.Users != nil {
		t.Fatalf("unchanged empty response = %#v, want no users payload", unchanged)
	}
}

func TestXrayUsersCacheEncryptsPayloadAndPreservesDeltaResponse(t *testing.T) {
	logic, redisClient, mini := newXrayUsersCacheTestLogic(t, xrayUsersCacheTestCodec(t))
	entry := newXrayUsersCacheEntry([]types.ServerXrayUser{
		{
			UserSubscribeID: 7,
			UserID:          11,
			Email:           "u_7",
			UUID:            "private-uuid",
			Password:        "private-uuid",
			Inbounds:        []string{"vless-in"},
			Enabled:         true,
		},
	})

	logic.writeXrayUsersCache(9, entry)
	key := xrayUsersCacheKey(9)
	raw, err := mini.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !privacycodec.IsSealed(raw) || strings.Contains(raw, "private-uuid") {
		t.Fatalf("cached payload is not encrypted: %q", raw)
	}
	if ttl := mini.TTL(key); ttl <= 0 || ttl > xrayUsersCacheTTL {
		t.Fatalf("cache TTL = %s, want (0, %s]", ttl, xrayUsersCacheTTL)
	}

	got, ok := logic.readXrayUsersCache(9)
	if !ok || got.Version != entry.Version || len(got.Users) != 1 || got.Users[0].UUID != "private-uuid" {
		t.Fatalf("cached entry = %#v, %v", got, ok)
	}
	unchanged := got.Response(got.Version)
	if unchanged.Full || unchanged.Users != nil {
		t.Fatalf("unchanged response = %#v, want version-only delta", unchanged)
	}
	full := got.Response("")
	if !full.Full || len(full.Users) != 1 {
		t.Fatalf("full response = %#v", full)
	}
	if count, err := redisClient.Exists(context.Background(), key).Result(); err != nil || count != 1 {
		t.Fatalf("cache key count = %d, %v", count, err)
	}
}

func TestXrayUsersCacheRejectsPlaintextAndDisabledPrivacy(t *testing.T) {
	logic, redisClient, _ := newXrayUsersCacheTestLogic(t, xrayUsersCacheTestCodec(t))
	key := xrayUsersCacheKey(12)
	if err := redisClient.Set(context.Background(), key, `{"version":"plain","users":[]}`, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if entry, ok := logic.readXrayUsersCache(12); ok || entry != nil {
		t.Fatalf("plaintext cache was accepted: %#v", entry)
	}
	if count, err := redisClient.Exists(context.Background(), key).Result(); err != nil || count != 0 {
		t.Fatalf("plaintext cache key count = %d, %v; want deleted", count, err)
	}

	disabled, disabledRedis, _ := newXrayUsersCacheTestLogic(t, privacycodec.DisabledCodec())
	disabled.writeXrayUsersCache(13, newXrayUsersCacheEntry([]types.ServerXrayUser{{
		UserSubscribeID: 13,
		UUID:            "must-not-be-cached",
	}}))
	if count, err := disabledRedis.Exists(context.Background(), xrayUsersCacheKey(13)).Result(); err != nil || count != 0 {
		t.Fatalf("disabled privacy cache key count = %d, %v; want zero", count, err)
	}
}

func TestCachedXrayUsersCoalescesConcurrentLoads(t *testing.T) {
	logic, _, _ := newXrayUsersCacheTestLogic(t, xrayUsersCacheTestCodec(t))
	var loads atomic.Int32
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	var startOnce sync.Once
	load := func() ([]types.ServerXrayUser, error) {
		loads.Add(1)
		startOnce.Do(func() { close(loadStarted) })
		<-releaseLoad
		return []types.ServerXrayUser{{UserSubscribeID: 21, UUID: "uuid-21", Enabled: true}}, nil
	}

	const callers = 24
	start := make(chan struct{})
	ready := make(chan struct{}, callers)
	results := make(chan *xrayUsersCacheEntry, callers)
	errs := make(chan error, callers)
	for range callers {
		go func() {
			ready <- struct{}{}
			<-start
			entry, err := logic.cachedXrayUsers(21, load)
			results <- entry
			errs <- err
		}()
	}
	for range callers {
		<-ready
	}
	close(start)
	select {
	case <-loadStarted:
	case <-time.After(time.Second):
		t.Fatal("loader did not start")
	}
	time.Sleep(25 * time.Millisecond)
	close(releaseLoad)

	for range callers {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		entry := <-results
		if entry == nil || len(entry.Users) != 1 || entry.Users[0].UserSubscribeID != 21 {
			t.Fatalf("entry = %#v", entry)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("loader calls = %d, want 1", got)
	}

	entry, err := logic.cachedXrayUsers(21, func() ([]types.ServerXrayUser, error) {
		loads.Add(1)
		return nil, nil
	})
	if err != nil || entry == nil || loads.Load() != 1 {
		t.Fatalf("second cache lookup = %#v, %v; loader calls = %d", entry, err, loads.Load())
	}
}

func newXrayUsersCacheTestLogic(
	t *testing.T,
	codec *privacycodec.Codec,
) (*QueryServerXrayUsersLogic, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	serviceContext := &svc.ServiceContext{
		Redis:                 client,
		Privacy:               codec,
		XrayUsersSingleFlight: syncx.NewSingleFlight(),
	}
	return NewQueryServerXrayUsersLogic(context.Background(), serviceContext), client, mini
}

func xrayUsersCacheTestCodec(t *testing.T) *privacycodec.Codec {
	t.Helper()
	codec, err := privacycodec.NewCodec(privacycodec.Keyring{
		ActiveKeyID: "cache",
		EncryptionKey: map[string]string{
			"cache": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)),
		},
		IndexKey: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func TestUsersVersionIncludesRuntimeFields(t *testing.T) {
	base := []types.ServerXrayUser{
		{
			UserSubscribeID: 10,
			UserID:          1,
			Email:           "u_10",
			UUID:            "uuid-10",
			Password:        "uuid-10",
			Level:           0,
			Inbounds:        []string{"vless"},
			SpeedLimit:      100,
			DeviceLimit:     2,
			Enabled:         true,
			ExpiredAt:       123,
		},
	}
	version := usersVersion(base)
	changedInbound := []types.ServerXrayUser{base[0]}
	changedInbound[0].Inbounds = []string{"trojan"}
	if got := usersVersion(changedInbound); got == version {
		t.Fatal("version did not change when inbound tags changed")
	}
	changedEnabled := []types.ServerXrayUser{base[0]}
	changedEnabled[0].Enabled = false
	if got := usersVersion(changedEnabled); got == version {
		t.Fatal("version did not change when enabled changed")
	}
}

func TestForEachXrayCSVItemSkipsBlankItems(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", value: "", want: nil},
		{name: "spaces", value: "  ", want: nil},
		{name: "single", value: "1", want: []string{"1"}},
		{name: "two", value: "1,2", want: []string{"1", "2"}},
		{name: "skips empty trailing item", value: " 1, 2, ", want: []string{"1", "2"}},
		{name: "skips empty middle item", value: "1,, 2", want: []string{"1", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			forEachXrayCSVItem(tt.value, func(item string) {
				got = append(got, item)
			})
			if len(got) != len(tt.want) {
				t.Fatalf("items = %#v, want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("items = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func assertColumns(t *testing.T, columns, required, forbidden []string) {
	t.Helper()
	columnSet := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnSet[column] = struct{}{}
	}
	for _, column := range required {
		if _, ok := columnSet[column]; !ok {
			t.Fatalf("columns must include %q", column)
		}
	}
	for _, column := range forbidden {
		if _, ok := columnSet[column]; ok {
			t.Fatalf("columns must not include %q", column)
		}
	}
}
