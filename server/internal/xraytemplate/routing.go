package xraytemplate

import (
	"encoding/json"
	"fmt"

	"github.com/neibniz/alpha/server/internal/viewutil"
)

const smallRoutingStringSetThreshold = 8

type routingStringSet struct {
	values []string
	index  map[string]struct{}
}

func (s *routingStringSet) Add(value string) bool {
	if s.index != nil {
		if _, ok := s.index[value]; ok {
			return false
		}
		s.index[value] = struct{}{}
		s.values = append(s.values, value)
		return true
	}
	for _, existing := range s.values {
		if existing == value {
			return false
		}
	}
	s.values = append(s.values, value)
	if len(s.values) > smallRoutingStringSetThreshold {
		s.index = make(map[string]struct{}, len(s.values))
		for _, existing := range s.values {
			s.index[existing] = struct{}{}
		}
	}
	return true
}

func (s routingStringSet) Contains(value string) bool {
	if s.index != nil {
		_, ok := s.index[value]
		return ok
	}
	for _, existing := range s.values {
		if existing == value {
			return true
		}
	}
	return false
}

func validateRoutingConfig(config ConfigObject) error {
	if value, ok := config["domainStrategy"]; ok {
		strategy, ok := value.(string)
		if !ok {
			return fmt.Errorf("routing.domainStrategy must be a string")
		}
		if !isRoutingDomainStrategy(strategy) {
			return fmt.Errorf("routing.domainStrategy must be one of AsIs, IPIfNonMatch, IPOnDemand")
		}
	}
	if value, ok := config["rules"]; ok {
		if err := validateRoutingRuleList(value, "routing.rules"); err != nil {
			return err
		}
	}
	if value, ok := config["balancers"]; ok {
		if err := validateRoutingBalancerList(value, "routing.balancers"); err != nil {
			return err
		}
	}
	return nil
}

func isRoutingDomainStrategy(value string) bool {
	switch value {
	case "AsIs", "IPIfNonMatch", "IPOnDemand":
		return true
	default:
		return false
	}
}

func validateRoutingRuleList(value any, path string) error {
	rules, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", path)
	}
	for index, item := range rules {
		rule, ok := item.(ConfigObject)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", path, index)
		}
		rulePath := fmt.Sprintf("%s[%d]", path, index)
		hasOutbound, err := validateOptionalString(rule, "outboundTag", rulePath)
		if err != nil {
			return err
		}
		hasBalancer, err := validateOptionalString(rule, "balancerTag", rulePath)
		if err != nil {
			return err
		}
		if hasOutbound && hasBalancer {
			return fmt.Errorf("%s cannot set outboundTag and balancerTag at the same time", rulePath)
		}
		if nested, ok := rule["rules"]; ok {
			if err := validateRoutingRuleList(nested, rulePath+".rules"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRoutingBalancerList(value any, path string) error {
	balancers, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", path)
	}
	for index, item := range balancers {
		balancer, ok := item.(ConfigObject)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", path, index)
		}
		if err := validateRequiredString(balancer, "tag", fmt.Sprintf("%s[%d]", path, index)); err != nil {
			return err
		}
	}
	return nil
}

func validateRequiredString(data ConfigObject, key string, path string) error {
	_, ok := data[key]
	if !ok {
		return fmt.Errorf("%s.%s is required", path, key)
	}
	_, err := validateOptionalString(data, key, path)
	return err
}

func validateOptionalString(data ConfigObject, key string, path string) (bool, error) {
	value, ok := data[key]
	if !ok {
		return false, nil
	}
	text, ok := value.(string)
	if !ok {
		return true, fmt.Errorf("%s.%s must be a string", path, key)
	}
	if text == "" {
		return true, fmt.Errorf("%s.%s cannot be empty", path, key)
	}
	return true, nil
}

func AppendUniqueConfigList(dst any, src any) []any {
	items := make([]any, 0, configListLen(dst)+configListLen(src))
	items = appendConfigList(items, dst)
	items = appendConfigList(items, src)
	return uniqueConfigList(items)
}

func configListLen(src any) int {
	switch value := src.(type) {
	case []any:
		return len(value)
	case []ConfigObject:
		return len(value)
	case nil:
		return 0
	default:
		return 1
	}
}

func appendConfigList(out []any, src any) []any {
	switch value := src.(type) {
	case []any:
		out = append(out, value...)
	case []ConfigObject:
		for _, item := range value {
			out = append(out, item)
		}
	case nil:
	default:
		out = append(out, value)
	}
	return out
}

func uniqueConfigList(items []any) []any {
	if len(items) < 2 {
		return items
	}
	var seen routingStringSet
	result := viewutil.NewRowsCollector[any](len(items))
	for _, item := range items {
		key := configJSONKey(item)
		if !seen.Add(key) {
			continue
		}
		result.Add(item)
	}
	return result.Rows()
}

func configJSONKey(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
}

func ValidateRoutingReferences(outbounds []ConfigObject, routing ConfigObject) error {
	if len(routing) == 0 {
		return nil
	}
	outboundTags := collectOutboundTags(outbounds)
	balancerTags := collectRoutingBalancerTags(routing["balancers"])
	if rules, ok := routing["rules"]; ok {
		if err := validateRoutingRuleReferences(rules, outboundTags, balancerTags, "routing.rules"); err != nil {
			return err
		}
	}
	return nil
}

func collectOutboundTags(outbounds []ConfigObject) routingStringSet {
	var result routingStringSet
	for _, data := range outbounds {
		if data == nil {
			continue
		}
		tag, ok := data["tag"].(string)
		if ok && tag != "" {
			result.Add(tag)
		}
	}
	return result
}

func collectRoutingBalancerTags(value any) routingStringSet {
	var result routingStringSet
	items, ok := value.([]any)
	if !ok {
		return result
	}
	for _, item := range items {
		data, ok := item.(ConfigObject)
		if !ok {
			continue
		}
		tag, ok := data["tag"].(string)
		if ok && tag != "" {
			result.Add(tag)
		}
	}
	return result
}

func validateRoutingRuleReferences(value any, outboundTags routingStringSet, balancerTags routingStringSet, path string) error {
	rules, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", path)
	}
	for index, item := range rules {
		rule, ok := item.(ConfigObject)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", path, index)
		}
		rulePath := fmt.Sprintf("%s[%d]", path, index)
		if outboundTag, ok := rule["outboundTag"].(string); ok {
			if !outboundTags.Contains(outboundTag) {
				return fmt.Errorf("%s references missing outboundTag %q", rulePath, outboundTag)
			}
		}
		if balancerTag, ok := rule["balancerTag"].(string); ok {
			if !balancerTags.Contains(balancerTag) {
				return fmt.Errorf("%s references missing balancerTag %q", rulePath, balancerTag)
			}
		}
		if nested, ok := rule["rules"]; ok {
			if err := validateRoutingRuleReferences(nested, outboundTags, balancerTags, rulePath+".rules"); err != nil {
				return err
			}
		}
	}
	return nil
}
