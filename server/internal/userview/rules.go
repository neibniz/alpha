package userview

import (
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/server/pkg/tool"
)

const smallRulesCacheThreshold = 8

type RulesCache struct {
	rows    []rulesCacheEntry
	index   map[string]int
	capHint int
}

type rulesCacheEntry struct {
	raw   string
	rules []string
}

func RulesFromRaw(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return []string{}
	}
	var rules []string
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return []string{}
	}
	return tool.NormalizeStringSlice(rules)
}

func NewRulesCache(capHint int) *RulesCache {
	if capHint <= 1 {
		return nil
	}
	return &RulesCache{capHint: capHint}
}

func (c *RulesCache) Rules(raw string) []string {
	key := strings.TrimSpace(raw)
	if c == nil {
		return RulesFromRaw(key)
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			return c.rows[pos].rules
		}
		return c.add(key)
	}
	for _, row := range c.rows {
		if row.raw == key {
			return row.rules
		}
	}
	return c.add(key)
}

func (c *RulesCache) add(key string) []string {
	rules := RulesFromRaw(key)
	c.ensureRows()
	c.rows = append(c.rows, rulesCacheEntry{
		raw:   key,
		rules: rules,
	})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > smallRulesCacheThreshold {
		nextCap := rulesCacheCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]rulesCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return rules
}

func (c *RulesCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallRulesCacheThreshold
	} else if capHint > smallRulesCacheThreshold+1 {
		capHint = smallRulesCacheThreshold + 1
	}
	c.rows = make([]rulesCacheEntry, 0, capHint)
}

func rulesCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
