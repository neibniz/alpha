package node

import "testing"

func TestStatusMarshalShapeAndNilReceiver(t *testing.T) {
	status := &Status{Cpu: 1.5, Mem: 2.5, Disk: 3.5, UpdatedAt: 123}
	if got, want := status.Marshal(), `{"cpu":1.5,"mem":2.5,"disk":3.5,"updated_at":123}`; got != want {
		t.Fatalf("Status.Marshal() = %s, want %s", got, want)
	}

	var nilStatus *Status
	if got := nilStatus.Marshal(); got != "{}" {
		t.Fatalf("nil Status.Marshal() = %s, want {}", got)
	}
}

func TestStatusUnmarshalResetsEmptyAndNull(t *testing.T) {
	for _, raw := range []string{"", "  ", "null", " \n null\t"} {
		status := Status{Cpu: 1, Mem: 2, Disk: 3, UpdatedAt: 4}
		if err := status.Unmarshal(raw); err != nil {
			t.Fatalf("Status.Unmarshal(%q) error = %v", raw, err)
		}
		if status != (Status{}) {
			t.Fatalf("Status.Unmarshal(%q) = %#v, want zero", raw, status)
		}
	}
}

func TestStatusUnmarshalDirectShape(t *testing.T) {
	var status Status
	if err := status.Unmarshal(`{"cpu":1.5,"mem":2.5,"disk":3.5,"updated_at":123}`); err != nil {
		t.Fatalf("Status.Unmarshal() error = %v", err)
	}
	if status.Cpu != 1.5 || status.Mem != 2.5 || status.Disk != 3.5 || status.UpdatedAt != 123 {
		t.Fatalf("Status.Unmarshal() = %#v", status)
	}
}

func TestOnlineUserCacheKeyHelpersKeepExistingFormat(t *testing.T) {
	if got := statusCacheKey(42); got != "node:status:42" {
		t.Fatalf("statusCacheKey() = %q, want node:status:42", got)
	}
	if got := onlineUserSubscribeCacheKey(42, "vless"); got != "node:online:subscribe:42:vless" {
		t.Fatalf("onlineUserSubscribeCacheKey() = %q, want node:online:subscribe:42:vless", got)
	}
	if got := onlineUserServerCacheKey(42); got != "node:online:server:42" {
		t.Fatalf("onlineUserServerCacheKey() = %q, want node:online:server:42", got)
	}
	if got := onlineUserServerSeqCacheKey(42); got != "node:online:server:42:seq" {
		t.Fatalf("onlineUserServerSeqCacheKey() = %q, want node:online:server:42:seq", got)
	}
	if got := onlineUserScore(123456789); got != "123456789" {
		t.Fatalf("onlineUserScore() = %q, want 123456789", got)
	}
}

func TestShouldSkipOnlineUsersUpdate(t *testing.T) {
	tests := []struct {
		name              string
		mode              string
		instanceID        string
		currentInstanceID string
		seq               int64
		currentSeq        int64
		want              bool
	}{
		{
			name:              "duplicate same instance",
			mode:              "delta",
			instanceID:        "agent-a",
			currentInstanceID: "agent-a",
			seq:               2,
			currentSeq:        2,
			want:              true,
		},
		{
			name:              "stale delta from old instance",
			mode:              "delta",
			instanceID:        "agent-a",
			currentInstanceID: "agent-b",
			seq:               5,
			currentSeq:        1,
			want:              true,
		},
		{
			name:              "full from restarted instance resets sequence",
			mode:              "full",
			instanceID:        "agent-b",
			currentInstanceID: "agent-a",
			seq:               1,
			currentSeq:        50,
			want:              false,
		},
		{
			name:       "full with instance resets legacy numeric sequence",
			mode:       "full",
			instanceID: "agent-b",
			seq:        1,
			currentSeq: 50,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipOnlineUsersUpdate(tt.mode, tt.instanceID, tt.currentInstanceID, tt.seq, tt.currentSeq)
			if got != tt.want {
				t.Fatalf("shouldSkipOnlineUsersUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseOnlineUserSeq(t *testing.T) {
	instanceID, seq, err := parseOnlineUserSeq("agent-a:42")
	if err != nil {
		t.Fatal(err)
	}
	if instanceID != "agent-a" || seq != 42 {
		t.Fatalf("parseOnlineUserSeq() = (%q, %d), want (agent-a, 42)", instanceID, seq)
	}

	instanceID, seq, err = parseOnlineUserSeq("7")
	if err != nil {
		t.Fatal(err)
	}
	if instanceID != "" || seq != 7 {
		t.Fatalf("parseOnlineUserSeq() = (%q, %d), want ('', 7)", instanceID, seq)
	}
}

func TestNormalizeOnlineUserMembersDeduplicatesAndLetsInactiveWin(t *testing.T) {
	active, inactive := normalizeOnlineUserMembers(
		[]int64{7, 0, 7, 8, 9, -1},
		[]int64{9, 9, 0, 10},
	)

	if len(active) != 2 || active[0] != "7" || active[1] != "8" {
		t.Fatalf("active members = %#v, want [7 8]", active)
	}
	if len(inactive) != 2 || inactive[0] != "9" || inactive[1] != "10" {
		t.Fatalf("inactive members = %#v, want [9 10]", inactive)
	}
}

func TestNormalizeOnlineUserMembersSkipsInvalidOnlyInputs(t *testing.T) {
	active, inactive := normalizeOnlineUserMembers([]int64{0, -1}, []int64{0, -2})
	if len(active) != 0 || len(inactive) != 0 {
		t.Fatalf("members = %#v/%#v, want empty", active, inactive)
	}
}

func TestOnlineUserMemberSetUsesLinearPathForSmallInput(t *testing.T) {
	var set onlineUserMemberSet
	if !set.Add(7) || !set.Add(8) {
		t.Fatalf("expected new ids to be added: %#v", set)
	}
	if set.Add(7) {
		t.Fatalf("duplicate id should not be added: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small online user member set should not allocate map")
	}
	if !set.Contains(7) || !set.Contains(8) || set.Contains(9) {
		t.Fatalf("member set contains mismatch: %#v", set)
	}
	members := set.StringMembers()
	if len(members) != 2 || members[0] != "7" || members[1] != "8" {
		t.Fatalf("members = %#v, want [7 8]", members)
	}
}

func TestOnlineUserMemberSetUpgradesLargeInputToMap(t *testing.T) {
	var set onlineUserMemberSet
	for id := int64(1); id <= onlineUserMemberLinearLimit+1; id++ {
		set.Add(id)
	}
	if set.index == nil {
		t.Fatal("large online user member set should allocate map")
	}
	if set.Add(1) {
		t.Fatal("duplicate id should not be added after map upgrade")
	}
	lastID := int64(onlineUserMemberLinearLimit + 1)
	if !set.Contains(lastID) {
		t.Fatalf("member set missing last id %d", lastID)
	}
}

func TestOnlineUserMemberSetUsesCapacityHintAndLazyMapForLargeInput(t *testing.T) {
	set := newOnlineUserMemberSet(onlineUserMemberLinearLimit + 1)
	if set.ids != nil || set.index != nil {
		t.Fatalf("online user member set should not allocate before first member: %#v", set)
	}
	if !set.Add(7) {
		t.Fatal("expected first member to be added")
	}
	if cap(set.ids) != onlineUserMemberLinearLimit+1 {
		t.Fatalf("online user member set cap = %d, want %d", cap(set.ids), onlineUserMemberLinearLimit+1)
	}
	if set.index != nil {
		t.Fatal("online user member set should not allocate map for one unique member")
	}
	if set.Add(7) {
		t.Fatal("duplicate id should not be added")
	}
}

func TestOnlineUserMemberSetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := newOnlineUserMemberSet(onlineUserMemberLinearLimit + 20)
	for id := int64(1); id <= onlineUserMemberLinearLimit+1; id++ {
		set.Add(id)
	}
	if set.index == nil {
		t.Fatal("online user member set should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := len(set.ids) * 2; cap(set.ids) > maxCap {
		t.Fatalf("online user member set cap = %d, want <= %d", cap(set.ids), maxCap)
	}
}

func TestNormalizeOnlineUserMembersSmallInputAvoidsMapAllocation(t *testing.T) {
	active := []int64{7, 8, 7, 9}
	inactive := []int64{9, 10, 9}
	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = normalizeOnlineUserMembers(active, inactive)
	})
	if allocs > 6 {
		t.Fatalf("normalizeOnlineUserMembers small input allocations = %f, want at most 6", allocs)
	}
}
