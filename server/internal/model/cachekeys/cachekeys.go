package cachekeys

import "strconv"

const smallMergeSetThreshold = 8

var emptyKeys = []string{}

func Int(prefix string, value int64) string {
	return prefix + strconv.FormatInt(value, 10)
}

func String(prefix, value string) string {
	return prefix + value
}

// Merge combines cache key groups while preserving the first-seen order.
func Merge(groups ...[]string) []string {
	capHint := mergeCapHint(groups)
	if capHint == 0 {
		return emptyKeys
	}
	collector := newMergeCollector(capHint)
	for _, group := range groups {
		for _, key := range group {
			collector.Add(key)
		}
	}
	return collector.Keys()
}

func mergeCapHint(groups [][]string) int {
	count := 0
	for _, group := range groups {
		for _, key := range group {
			if key != "" {
				count++
			}
		}
	}
	return count
}

type mergeCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newMergeCollector(capHint int) mergeCollector {
	return mergeCollector{capHint: capHint}
}

func (c *mergeCollector) Add(key string) {
	if key == "" {
		return
	}
	if c.seen != nil {
		if _, ok := c.seen[key]; ok {
			return
		}
		c.seen[key] = struct{}{}
		c.keys = append(c.keys, key)
		return
	}
	if containsKey(c.keys, key) {
		return
	}
	if c.keys == nil {
		capHint := c.capHint
		if capHint <= 0 || capHint > smallMergeSetThreshold+1 {
			capHint = smallMergeSetThreshold + 1
		}
		c.keys = make([]string, 0, capHint)
	}
	c.keys = append(c.keys, key)
	if len(c.keys) > smallMergeSetThreshold {
		nextCap := mergeCollectorLookupCap(len(c.keys), c.capHint)
		if nextCap > cap(c.keys) {
			keys := make([]string, len(c.keys), nextCap)
			copy(keys, c.keys)
			c.keys = keys
		}
		c.seen = make(map[string]struct{}, nextCap)
		for _, existing := range c.keys {
			c.seen[existing] = struct{}{}
		}
	}
}

func (c mergeCollector) Keys() []string {
	if len(c.keys) == 0 {
		return emptyKeys
	}
	return c.keys
}

func mergeCollectorLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func containsKey(keys []string, key string) bool {
	for _, existing := range keys {
		if existing == key {
			return true
		}
	}
	return false
}
