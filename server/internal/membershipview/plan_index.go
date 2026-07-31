package membershipview

import (
	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/types"
)

const smallPlanIndexThreshold = 8

type PlanIndex struct {
	plans   []types.MembershipPlan
	byID    map[int64]int
	capHint int
}

func NewPlanIndex(plans []modelMembership.Plan) PlanIndex {
	if len(plans) == 0 {
		return PlanIndex{}
	}
	index := PlanIndex{capHint: len(plans)}
	for i := range plans {
		index.Add(PlanFromModel(&plans[i]))
	}
	return index
}

func (i *PlanIndex) Add(plan types.MembershipPlan) {
	if i.byID != nil {
		if row, ok := i.byID[plan.Id]; ok {
			i.plans[row] = plan
			return
		}
		i.byID[plan.Id] = len(i.plans)
		i.plans = append(i.plans, plan)
		return
	}
	for row := range i.plans {
		if i.plans[row].Id == plan.Id {
			i.plans[row] = plan
			return
		}
	}
	i.ensurePlans()
	i.plans = append(i.plans, plan)
	if len(i.plans) > smallPlanIndexThreshold {
		i.upgradeLookup()
	}
}

func (i PlanIndex) Find(id int64) (*types.MembershipPlan, bool) {
	if i.byID != nil {
		row, ok := i.byID[id]
		if !ok {
			return nil, false
		}
		return &i.plans[row], true
	}
	for row := range i.plans {
		if i.plans[row].Id == id {
			return &i.plans[row], true
		}
	}
	return nil, false
}

func (i *PlanIndex) ensurePlans() {
	if i.plans != nil || i.capHint <= 0 {
		return
	}
	i.plans = make([]types.MembershipPlan, 0, planIndexInitialCap(i.capHint))
}

func (i *PlanIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := planIndexLookupCap(len(i.plans), i.capHint)
	if nextCap > cap(i.plans) {
		plans := make([]types.MembershipPlan, len(i.plans), nextCap)
		copy(plans, i.plans)
		i.plans = plans
	}
	i.byID = make(map[int64]int, nextCap)
	for row := range i.plans {
		i.byID[i.plans[row].Id] = row
	}
}

func planIndexLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func planIndexInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallPlanIndexThreshold+1 {
		return smallPlanIndexThreshold + 1
	}
	return capHint
}
