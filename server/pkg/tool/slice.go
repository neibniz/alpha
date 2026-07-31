package tool

import (
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/pkg/logger"
)

const smallNormalizeSetThreshold = 8

var (
	emptyInt64Slice  = []int64{}
	emptyStringSlice = []string{}
)

func Int64SliceToStringSlice(slice []int64) []string {
	stringSlice := make([]string, len(slice))
	for i, num := range slice {
		stringSlice[i] = strconv.Itoa(int(num))
	}
	return stringSlice
}
func StringSliceToInt64Slice(slice []string) []int64 {
	int64Slice := make([]int64, len(slice))
	for i, str := range slice {
		num, err := strconv.Atoi(str)
		if err != nil {
			logger.Error("[Tools] StringSliceToInt64Slice",
				logger.Field("error", err.Error()),
				logger.Field("str", str),
			)
			continue
		}
		int64Slice[i] = int64(num)
	}
	return int64Slice
}

func StringToInt64Slice(s string) []int64 {
	intSlice := make([]int64, 0, csvItemCount(s))
	forEachCSVItem(s, func(str string) {
		num, ok := parseCSVInt64(str, "StringToInt64Slice")
		if !ok {
			return
		}
		intSlice = append(intSlice, num)
	})
	return intSlice
}

func StringToPositiveUniqueInt64Slice(s string) []int64 {
	count := csvItemCount(s)
	if count == 0 {
		return emptyInt64Slice
	}
	collector := newComparableDeduper[int64](count)
	forEachCSVItem(s, func(str string) {
		num, ok := parseCSVInt64(str, "StringToPositiveUniqueInt64Slice")
		if !ok || num <= 0 {
			return
		}
		collector.Add(num)
	})
	if collector.Len() == 0 {
		return emptyInt64Slice
	}
	return collector.Values()
}

func NormalizePositiveInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return emptyInt64Slice
	}
	collector := newComparableDeduper[int64](len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		collector.Add(value)
	}
	if collector.Len() == 0 {
		return emptyInt64Slice
	}
	return collector.Values()
}

func Int64CSVContains(s string, target int64) (bool, bool) {
	hasValues := false
	contains := false
	forEachCSVItem(s, func(str string) {
		if contains {
			return
		}
		num, ok := parseCSVInt64(str, "Int64CSVContains")
		if !ok {
			return
		}
		hasValues = true
		if num == target {
			contains = true
		}
	})
	return hasValues, contains
}

func Int64SliceToString(intSlice []int64) string {
	strSlice := make([]string, 0, len(intSlice))
	for _, num := range intSlice {
		strSlice = append(strSlice, strconv.FormatInt(num, 10))
	}
	return strings.Join(strSlice, ",")
}

// string slice to string
func StringSliceToString(stringSlice []string) string {
	return strings.Join(NormalizeStringSlice(stringSlice), ",")
}

func NormalizeStringSlice(stringSlice []string) []string {
	if len(stringSlice) == 0 {
		return emptyStringSlice
	}
	collector := newComparableDeduper[string](len(stringSlice))
	for _, item := range stringSlice {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		collector.Add(item)
	}
	if collector.Len() == 0 {
		return emptyStringSlice
	}
	return collector.Values()
}

func int64SliceContains(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// StringMergeAndRemoveDuplicates Tool function to convert multiple comma separated strings into [] strings and deduplicate them
func StringMergeAndRemoveDuplicates(strs ...string) []string {
	count := csvItemsCount(strs)
	if count == 0 {
		return emptyStringSlice
	}
	collector := newComparableDeduper[string](count)
	for _, str := range strs {
		forEachCSVItem(str, func(item string) {
			collector.Add(item)
		})
	}
	if collector.Len() == 0 {
		return emptyStringSlice
	}
	return collector.Values()
}

func csvItemCount(value string) int {
	count := 0
	forEachCSVItem(value, func(string) {
		count++
	})
	return count
}

func csvItemsCount(values []string) int {
	count := 0
	for _, value := range values {
		count += csvItemCount(value)
	}
	return count
}

func forEachCSVItem(value string, visit func(string)) {
	for len(value) > 0 {
		part := value
		if index := strings.IndexByte(value, ','); index >= 0 {
			part = value[:index]
			value = value[index+1:]
		} else {
			value = ""
		}
		item := strings.TrimSpace(part)
		if item != "" {
			visit(item)
		}
	}
}

func parseCSVInt64(value string, caller string) (int64, bool) {
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		logger.Error("[Tools] "+caller,
			logger.Field("error", err.Error()),
			logger.Field("str", value),
		)
		return 0, false
	}
	return num, true
}

func StringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func RemoveElementBySlice[T comparable](slice []T, element T) []T {
	result := make([]T, 0, len(slice))
	for _, s := range slice {
		if s != element {
			result = append(result, s)
		}
	}
	return result
}

func RemoveDuplicateElements[T comparable](input ...T) []T {
	if len(input) == 0 {
		return []T{}
	}
	skipEmptyString := isStringType[T]()
	collector := newComparableDeduper[T](len(input))
	for _, item := range input {
		// 仅在 T 是 string 类型时跳过空字符串
		if skipEmptyString && isEmptyString(item) {
			continue
		}
		collector.Add(item)
	}
	if collector.Len() == 0 {
		return []T{}
	}
	return collector.Values()
}

type comparableDeduper[T comparable] struct {
	values  []T
	seen    map[T]struct{}
	capHint int
}

func newComparableDeduper[T comparable](capHint int) comparableDeduper[T] {
	return comparableDeduper[T]{capHint: capHint}
}

func (d *comparableDeduper[T]) Add(value T) bool {
	if d.seen != nil {
		if _, ok := d.seen[value]; ok {
			return false
		}
		d.seen[value] = struct{}{}
		d.values = append(d.values, value)
		return true
	}
	if comparableSliceContains(d.values, value) {
		return false
	}
	if d.values == nil {
		d.values = make([]T, 0, normalizeInitialCap(d.capHint))
	}
	d.values = append(d.values, value)
	if len(d.values) > smallNormalizeSetThreshold {
		d.upgradeLookup()
	}
	return true
}

func (d comparableDeduper[T]) Len() int {
	return len(d.values)
}

func (d comparableDeduper[T]) Values() []T {
	return d.values
}

func (d *comparableDeduper[T]) upgradeLookup() {
	if d.seen != nil {
		return
	}
	nextCap := normalizeLookupCap(len(d.values), d.capHint)
	if nextCap > cap(d.values) {
		values := make([]T, len(d.values), nextCap)
		copy(values, d.values)
		d.values = values
	}
	d.seen = make(map[T]struct{}, nextCap)
	for _, value := range d.values {
		d.seen[value] = struct{}{}
	}
}

func normalizeLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func normalizeInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallNormalizeSetThreshold+1 {
		return smallNormalizeSetThreshold + 1
	}
	return capHint
}

func isStringType[T comparable]() bool {
	var zero T
	_, ok := any(zero).(string)
	return ok
}

func isEmptyString[T comparable](item T) bool {
	value, _ := any(item).(string)
	return value == ""
}

func comparableSliceContains[T comparable](slice []T, target T) bool {
	for _, value := range slice {
		if value == target {
			return true
		}
	}
	return false
}

func Contains[T comparable](slice []T, target T) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// RemoveStringElement 移除指定元素
func RemoveStringElement(arr []string, element ...string) []string {
	if len(arr) == 0 {
		return emptyStringSlice
	}
	if len(element) == 0 {
		return append([]string{}, arr...)
	}
	var removeSet map[string]struct{}
	if len(element) > smallNormalizeSetThreshold {
		removeSet = make(map[string]struct{}, len(element))
		for _, item := range element {
			removeSet[item] = struct{}{}
		}
	}
	result := make([]string, 0, len(arr))
	for _, str := range arr {
		if removeSet != nil {
			if _, ok := removeSet[str]; ok {
				continue
			}
		} else if stringSliceContains(element, str) {
			continue
		}
		result = append(result, str)
	}
	if len(result) == 0 {
		return emptyStringSlice
	}
	return result
}
