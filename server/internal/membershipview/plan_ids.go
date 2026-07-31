package membershipview

import modelMembership "github.com/neibniz/alpha/server/internal/model/membership"

type PlanIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func NewPlanIDSet(capHint int) PlanIDSet {
	return PlanIDSet{capHint: capHint}
}

func (s *PlanIDSet) Add(id int64) bool {
	if s == nil || id <= 0 {
		return false
	}
	if s.seen != nil {
		if _, ok := s.seen[id]; ok {
			return false
		}
		s.seen[id] = struct{}{}
		s.ids = append(s.ids, id)
		return true
	}
	for _, existing := range s.ids {
		if existing == id {
			return false
		}
	}
	s.ensureIDs()
	s.ids = append(s.ids, id)
	if len(s.ids) > smallPlanIndexThreshold {
		s.upgradeLookup()
	}
	return true
}

func (s PlanIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *PlanIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, planIndexInitialCap(s.capHint))
}

func (s *PlanIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := planIndexLookupCap(len(s.ids), s.capHint)
	if nextCap > cap(s.ids) {
		ids := make([]int64, len(s.ids), nextCap)
		copy(ids, s.ids)
		s.ids = ids
	}
	s.seen = make(map[int64]struct{}, nextCap)
	for _, existing := range s.ids {
		s.seen[existing] = struct{}{}
	}
}

func NewPlanRows(planIDs []int64) []modelMembership.Plan {
	if len(planIDs) == 0 {
		return []modelMembership.Plan{}
	}
	return make([]modelMembership.Plan, 0, len(planIDs))
}
