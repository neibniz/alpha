package tool

func SliceIntersectInt64(slice1, slice2 []int64) []int64 {
	if len(slice1) == 0 || len(slice2) == 0 {
		return emptyInt64Slice
	}
	if len(slice1) <= smallNormalizeSetThreshold {
		var nn []int64
		for _, v := range slice2 {
			if int64OccurrenceCount(slice1, v) != 1 {
				continue
			}
			if nn == nil {
				nn = make([]int64, 0, min(len(slice1), len(slice2)))
			}
			nn = append(nn, v)
		}
		if len(nn) == 0 {
			return emptyInt64Slice
		}
		return nn
	}
	counter := newInt64CandidateCounter(slice2)
	counter.CountSlice(slice1)
	var nn []int64
	for _, v := range slice2 {
		if counter.Count(v) == 1 {
			if nn == nil {
				nn = make([]int64, 0, min(len(counter.values), len(slice2)))
			}
			nn = append(nn, v)
		}
	}
	if len(nn) == 0 {
		return emptyInt64Slice
	}
	return nn
}

// SliceDifferenceInt64 returns values from slice1 except values that appear
// exactly once in slice1 and are present in slice2.
func SliceDifferenceInt64(slice1, slice2 []int64) []int64 {
	if len(slice1) == 0 {
		return emptyInt64Slice
	}
	if len(slice2) == 0 {
		return append([]int64{}, slice1...)
	}
	if len(slice1) <= smallNormalizeSetThreshold {
		var nn []int64
		for _, value := range slice1 {
			if int64OccurrenceCount(slice1, value) == 1 && int64SliceContains(slice2, value) {
				continue
			}
			if nn == nil {
				nn = make([]int64, 0, len(slice1))
			}
			nn = append(nn, value)
		}
		if len(nn) == 0 {
			return emptyInt64Slice
		}
		return nn
	}
	counter := newInt64CandidateCounter(slice2)
	counter.CountSlice(slice1)

	var nn []int64
	for _, value := range slice1 {
		if counter.Count(value) == 1 {
			continue
		}
		if nn == nil {
			nn = make([]int64, 0, len(slice1))
		}
		nn = append(nn, value)
	}
	if len(nn) == 0 {
		return emptyInt64Slice
	}
	return nn
}

type int64CandidateCounter struct {
	values  []int64
	counts  []int
	index   map[int64]int
	capHint int
}

func newInt64CandidateCounter(candidates []int64) int64CandidateCounter {
	counter := int64CandidateCounter{capHint: len(candidates)}
	for _, value := range candidates {
		counter.AddCandidate(value)
	}
	return counter
}

func (c *int64CandidateCounter) AddCandidate(value int64) {
	if c.index != nil {
		if _, ok := c.index[value]; ok {
			return
		}
		c.index[value] = len(c.values)
		c.values = append(c.values, value)
		c.counts = append(c.counts, 0)
		return
	}
	for _, existing := range c.values {
		if existing == value {
			return
		}
	}
	c.ensureRows()
	c.values = append(c.values, value)
	c.counts = append(c.counts, 0)
	if len(c.values) > smallNormalizeSetThreshold {
		c.upgradeLookup()
	}
}

func (c *int64CandidateCounter) ensureRows() {
	if c.values != nil {
		return
	}
	capacity := int64CandidateCounterInitialCap(c.capHint)
	c.values = make([]int64, 0, capacity)
	c.counts = make([]int, 0, capacity)
}

func (c *int64CandidateCounter) upgradeLookup() {
	if c.index != nil {
		return
	}
	nextCap := normalizeLookupCap(len(c.values), c.capHint)
	if nextCap > cap(c.values) {
		values := make([]int64, len(c.values), nextCap)
		copy(values, c.values)
		c.values = values

		counts := make([]int, len(c.counts), nextCap)
		copy(counts, c.counts)
		c.counts = counts
	}
	c.index = make(map[int64]int, nextCap)
	for row, existing := range c.values {
		c.index[existing] = row
	}
}

func int64CandidateCounterInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallNormalizeSetThreshold+1 {
		return smallNormalizeSetThreshold + 1
	}
	return capHint
}

func (c *int64CandidateCounter) CountSlice(values []int64) {
	for _, value := range values {
		c.increment(value)
	}
}

func (c *int64CandidateCounter) increment(value int64) {
	if c.index != nil {
		row, ok := c.index[value]
		if ok {
			c.counts[row]++
		}
		return
	}
	for row, existing := range c.values {
		if existing == value {
			c.counts[row]++
			return
		}
	}
}

func (c int64CandidateCounter) Count(value int64) int {
	if c.index != nil {
		row, ok := c.index[value]
		if !ok {
			return 0
		}
		return c.counts[row]
	}
	for row, existing := range c.values {
		if existing == value {
			return c.counts[row]
		}
	}
	return 0
}

func int64OccurrenceCount(slice []int64, value int64) int {
	count := 0
	for _, item := range slice {
		if item == value {
			count++
		}
	}
	return count
}

// SliceIsExistInt64 checks if a value exists in a slice
func SliceIsExistInt64(slice []int64, value int64) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
