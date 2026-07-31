package viewutil

const SmallRowsCapacityThreshold = 8

type RowsCollector[T any] struct {
	rows    []T
	capHint int
}

func NewRowsCollector[T any](capHint int) RowsCollector[T] {
	return RowsCollector[T]{capHint: capHint}
}

func (c *RowsCollector[T]) Add(row T) {
	if c.rows == nil {
		c.rows = make([]T, 0, rowsInitialCap(c.capHint))
	}
	if len(c.rows) == cap(c.rows) && len(c.rows) >= SmallRowsCapacityThreshold {
		nextCap := rowsGrowthCap(len(c.rows)+1, c.capHint)
		rows := make([]T, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.rows = append(c.rows, row)
}

func (c RowsCollector[T]) Rows() []T {
	if len(c.rows) == 0 {
		return []T{}
	}
	return c.rows
}

func rowsInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > SmallRowsCapacityThreshold+1 {
		return SmallRowsCapacityThreshold + 1
	}
	return capHint
}

func rowsGrowthCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
