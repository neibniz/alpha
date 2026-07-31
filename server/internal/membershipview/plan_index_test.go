package membershipview

import (
	"fmt"
	"testing"

	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestPlanIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := NewPlanIndex([]modelMembership.Plan{
		{Id: 1, Name: "Basic"},
		{Id: 2, Name: "Pro"},
	})

	if index.byID != nil {
		t.Fatal("small plan index should not allocate map")
	}
	plan, ok := index.Find(2)
	if !ok || plan.Name != "Pro" {
		t.Fatalf("Find(2) = %#v, %v; want Pro", plan, ok)
	}
	if _, ok := index.Find(3); ok {
		t.Fatal("Find(3) unexpectedly found a plan")
	}
}

func TestPlanIndexUpdatesDuplicatePlanID(t *testing.T) {
	var index PlanIndex
	index.Add(types.MembershipPlan{Id: 1, Name: "Old"})
	index.Add(types.MembershipPlan{Id: 1, Name: "New"})

	if len(index.plans) != 1 {
		t.Fatalf("plan index len = %d, want 1", len(index.plans))
	}
	plan, ok := index.Find(1)
	if !ok || plan.Name != "New" {
		t.Fatalf("Find(1) = %#v, %v; want updated plan", plan, ok)
	}
	if index.byID != nil {
		t.Fatal("small duplicate plan index should stay on linear path")
	}
}

func TestPlanIndexUpgradesLargeInputToMap(t *testing.T) {
	var index PlanIndex
	for i := int64(1); i <= smallPlanIndexThreshold+1; i++ {
		index.Add(types.MembershipPlan{Id: i, Name: fmt.Sprintf("Plan %d", i)})
	}

	if index.byID == nil {
		t.Fatal("large plan index should allocate map")
	}
	plan, ok := index.Find(int64(smallPlanIndexThreshold + 1))
	if !ok || plan.Name != fmt.Sprintf("Plan %d", smallPlanIndexThreshold+1) {
		t.Fatalf("Find(last) = %#v, %v; want upgraded lookup", plan, ok)
	}
}

func TestNewPlanIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	plans := make([]modelMembership.Plan, 0, smallPlanIndexThreshold+20)
	for i := 0; i < cap(plans); i++ {
		plans = append(plans, modelMembership.Plan{Id: 1, Name: fmt.Sprintf("Plan %d", i)})
	}

	index := NewPlanIndex(plans)

	if index.byID != nil {
		t.Fatal("large duplicate NewPlanIndex input should stay on linear path")
	}
	if len(index.plans) != 1 {
		t.Fatalf("plan index len = %d, want 1", len(index.plans))
	}
	if cap(index.plans) != smallPlanIndexThreshold+1 {
		t.Fatalf("plan index cap = %d, want small initial cap", cap(index.plans))
	}
	plan, ok := index.Find(1)
	if !ok || plan.Name != fmt.Sprintf("Plan %d", smallPlanIndexThreshold+19) {
		t.Fatalf("Find(1) = %#v, %v; want latest duplicate plan", plan, ok)
	}
}

func TestNewPlanIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	plans := make([]modelMembership.Plan, 0, smallPlanIndexThreshold+20)
	for i := int64(1); i <= smallPlanIndexThreshold+1; i++ {
		plans = append(plans, modelMembership.Plan{Id: i, Name: fmt.Sprintf("Plan %d", i)})
	}
	for len(plans) < smallPlanIndexThreshold+20 {
		plans = append(plans, modelMembership.Plan{Id: 1, Name: "Updated Plan 1"})
	}

	index := NewPlanIndex(plans)

	if index.byID == nil {
		t.Fatal("large NewPlanIndex input should allocate map after actual unique plans exceed threshold")
	}
	if len(index.plans) != smallPlanIndexThreshold+1 {
		t.Fatalf("plan index len = %d, want %d", len(index.plans), smallPlanIndexThreshold+1)
	}
	if maxCap := len(index.plans) * 2; cap(index.plans) > maxCap {
		t.Fatalf("plan index cap = %d, want at most %d", cap(index.plans), maxCap)
	}
	plan, ok := index.Find(1)
	if !ok || plan.Name != "Updated Plan 1" {
		t.Fatalf("Find(1) = %#v, %v; want updated plan", plan, ok)
	}
}
