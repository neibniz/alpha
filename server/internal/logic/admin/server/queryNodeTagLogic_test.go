package server

import (
	"context"
	"reflect"
	"testing"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQueryDistinctNodeTagsFallbackNormalizesTags(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelNode.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*modelNode.Node{
		{Name: "node-a", Tags: "edge, premium", Protocol: "vless", InboundAlias: "vless", Sort: 1},
		{Name: "node-b", Tags: "edge,,trial", Protocol: "vless", InboundAlias: "vless", Sort: 2},
		{Name: "node-c", Tags: "", Protocol: "vless", InboundAlias: "vless", Sort: 3},
		{Name: "node-d", Tags: " premium ", Protocol: "vless", InboundAlias: "vless", Sort: 4},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := queryDistinctNodeTags(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"edge", "premium", "trial"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestNodeTagCollectorNormalizesWithoutIntermediateTagSlice(t *testing.T) {
	collector := newNodeTagCollector(2)
	collector.AddCSV(" premium,edge,, premium ")
	collector.AddCSV("trial,edge")
	collector.Add("  ")

	got := collector.Tags()
	if want := []string{"edge", "premium", "trial"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestNodeTagCollectorTagsKeepsEmptySliceShape(t *testing.T) {
	collector := newNodeTagCollector(0)

	got := collector.Tags()
	if got == nil || len(got) != 0 {
		t.Fatalf("empty tags = %#v, want non-nil empty slice", got)
	}
}

func TestNodeTagCollectorUsesLinearDedupForSmallInput(t *testing.T) {
	collector := newNodeTagCollector(2)
	if collector.tags != nil || collector.seen != nil {
		t.Fatalf("small tag collector should not allocate before first tag: %#v", collector)
	}
	collector.AddCSV("premium,edge,premium")
	if cap(collector.tags) != 2 {
		t.Fatalf("small tag collector cap = %d, want 2", cap(collector.tags))
	}

	if collector.seen != nil {
		t.Fatal("small tag collector should not allocate map")
	}
	if want := []string{"edge", "premium"}; !reflect.DeepEqual(collector.Tags(), want) {
		t.Fatalf("tags = %#v, want %#v", collector.Tags(), want)
	}
}

func TestNodeTagCollectorUpgradesLargeInputToMap(t *testing.T) {
	collector := newNodeTagCollector(smallNodeTagCollectorThreshold + 1)
	if collector.tags != nil || collector.seen != nil {
		t.Fatalf("large tag collector should not allocate storage from hint alone: %#v", collector)
	}
	collector.Add("tag-1")
	collector.Add("tag-1")
	if collector.seen != nil || len(collector.tags) != 1 {
		t.Fatalf("duplicate first tag should stay on linear path: %#v", collector)
	}
	for _, tag := range []string{
		"tag-2", "tag-3", "tag-4", "tag-5",
		"tag-6", "tag-7", "tag-8", "tag-9",
	} {
		collector.Add(tag)
	}
	collector.Add("tag-1")

	if collector.seen == nil {
		t.Fatal("large tag collector should upgrade to map")
	}
	if len(collector.tags) != smallNodeTagCollectorThreshold+1 {
		t.Fatalf("tag count = %d, want %d", len(collector.tags), smallNodeTagCollectorThreshold+1)
	}
}

func TestNodeTagCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newNodeTagCollector(smallNodeTagCollectorThreshold + 20)
	for _, tag := range []string{
		"tag-1", "tag-2", "tag-3", "tag-4", "tag-5",
		"tag-6", "tag-7", "tag-8", "tag-9",
	} {
		collector.Add(tag)
	}
	if collector.seen == nil {
		t.Fatal("node tag collector should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(collector.tags) * 2; cap(collector.tags) > maxCap {
		t.Fatalf("node tag collector cap = %d, want at most %d", cap(collector.tags), maxCap)
	}
	for i := 0; i < smallNodeTagCollectorThreshold+20; i++ {
		collector.Add("tag-1")
	}
	if maxCap := len(collector.tags) * 2; cap(collector.tags) > maxCap {
		t.Fatalf("node tag collector cap after duplicates = %d, want at most %d", cap(collector.tags), maxCap)
	}
}
