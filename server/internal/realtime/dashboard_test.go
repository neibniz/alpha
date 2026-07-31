package realtime

import (
	"testing"
	"time"
)

func TestDashboardManagerPublishesToSubscribers(t *testing.T) {
	manager := NewDashboardManager()
	sub, unsubscribe := manager.Subscribe()
	defer unsubscribe()

	manager.Publish(DashboardEventTraffic)

	select {
	case msg := <-sub.C:
		if msg.Reason != DashboardEventTraffic {
			t.Fatalf("message reason = %q, want %q", msg.Reason, DashboardEventTraffic)
		}
		if msg.Seq == 0 {
			t.Fatal("message seq was not assigned")
		}
		if msg.UpdatedAt == 0 {
			t.Fatal("message updated_at was not assigned")
		}
	case <-time.After(time.Second):
		t.Fatal("dashboard message was not delivered")
	}
}

func TestDeliverDashboardMessageDropsOldestWhenFull(t *testing.T) {
	sub := &DashboardSubscriber{C: make(chan DashboardMessage, 1)}
	oldMsg := DashboardMessage{Reason: DashboardEventActivity, Seq: 1}
	newMsg := DashboardMessage{Reason: DashboardEventBusiness, Seq: 2}

	sub.C <- oldMsg
	deliverDashboardMessage(sub, newMsg)

	select {
	case got := <-sub.C:
		if got != newMsg {
			t.Fatalf("remaining message = %#v, want latest %#v", got, newMsg)
		}
	default:
		t.Fatal("dashboard channel is empty")
	}
}

func TestDashboardSubscriberSetUsesLinearPathForSmallInput(t *testing.T) {
	var set dashboardSubscriberSet
	first := &DashboardSubscriber{}
	second := &DashboardSubscriber{}

	set.Add(first)
	set.Add(first)
	set.Add(second)

	if set.index != nil {
		t.Fatal("small dashboard subscriber set should not allocate map")
	}
	got := 0
	set.ForEach(func(*DashboardSubscriber) {
		got++
	})
	if got != 2 {
		t.Fatalf("subscriber count = %d, want 2", got)
	}

	set.Remove(first)
	got = 0
	set.ForEach(func(sub *DashboardSubscriber) {
		if sub == first {
			t.Fatal("removed subscriber was visited")
		}
		got++
	})
	if got != 1 {
		t.Fatalf("subscriber count after remove = %d, want 1", got)
	}
}

func TestDashboardSubscriberSetUpgradesLargeInputToMap(t *testing.T) {
	var set dashboardSubscriberSet
	subs := make([]*DashboardSubscriber, 0, smallDashboardSubscriberSetThreshold+1)
	for i := 0; i <= smallDashboardSubscriberSetThreshold; i++ {
		sub := &DashboardSubscriber{}
		subs = append(subs, sub)
		set.Add(sub)
	}

	if set.index == nil {
		t.Fatal("large dashboard subscriber set should allocate map")
	}
	if set.items != nil {
		t.Fatalf("map-backed dashboard subscriber set should release slice storage: %#v", set.items)
	}
	set.Remove(subs[0])

	visited := make(map[*DashboardSubscriber]struct{})
	set.ForEach(func(sub *DashboardSubscriber) {
		visited[sub] = struct{}{}
	})
	if _, ok := visited[subs[0]]; ok {
		t.Fatal("removed subscriber was visited after map upgrade")
	}
	if len(visited) != smallDashboardSubscriberSetThreshold {
		t.Fatalf("subscriber count after remove = %d, want %d", len(visited), smallDashboardSubscriberSetThreshold)
	}
}

func TestDashboardSubscriberSetMapModeDoesNotRetainRemovedSubscribers(t *testing.T) {
	var set dashboardSubscriberSet
	subs := make([]*DashboardSubscriber, 0, smallDashboardSubscriberSetThreshold+1)
	for i := 0; i <= smallDashboardSubscriberSetThreshold; i++ {
		sub := &DashboardSubscriber{}
		subs = append(subs, sub)
		set.Add(sub)
	}

	removed := subs[0]
	set.Remove(removed)
	set.Add(subs[1])
	if set.items != nil {
		t.Fatalf("map-backed subscriber set should not rebuild slice storage: %#v", set.items)
	}
	if _, ok := set.index[removed]; ok {
		t.Fatal("removed subscriber should not remain in map index")
	}

	visited := 0
	set.ForEach(func(sub *DashboardSubscriber) {
		if sub == removed {
			t.Fatal("removed subscriber was visited")
		}
		visited++
	})
	if visited != smallDashboardSubscriberSetThreshold {
		t.Fatalf("visited subscribers = %d, want %d", visited, smallDashboardSubscriberSetThreshold)
	}
}
