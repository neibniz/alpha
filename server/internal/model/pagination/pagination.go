package pagination

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

func Normalize(page, size int) (int, int) {
	return NormalizeWithLimit(page, size, DefaultPageSize, MaxPageSize)
}

func NormalizeWithLimit(page, size, defaultSize, maxSize int) (int, int) {
	if defaultSize < 1 {
		defaultSize = DefaultPageSize
	}
	if maxSize < 1 {
		maxSize = MaxPageSize
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}

func Offset(page, size int) int {
	if page <= 1 || size <= 0 {
		return 0
	}
	pageIndex := page - 1
	maxInt := int(^uint(0) >> 1)
	if pageIndex > maxInt/size {
		return maxInt
	}
	return pageIndex * size
}

// PageWindow returns the query offset and exact result capacity for a page.
// It reports false when the page cannot contain rows, without overflowing int.
func PageWindow(total int64, page, size int) (offset, capacity int, ok bool) {
	if total <= 0 || page < 1 || size < 1 {
		return 0, 0, false
	}
	pageIndex := int64(page - 1)
	size64 := int64(size)
	if pageIndex > (total-1)/size64 {
		return 0, 0, false
	}
	maxInt := int(^uint(0) >> 1)
	if pageIndex > int64(maxInt/size) {
		return 0, 0, false
	}

	offset = int(pageIndex * size64)
	remaining := total - int64(offset)
	capacity = size
	if remaining < size64 {
		capacity = int(remaining)
	}
	return offset, capacity, true
}

func RowsCap(total int64, size int) int {
	_, capacity, ok := PageWindow(total, 1, size)
	if !ok {
		return 0
	}
	return capacity
}
