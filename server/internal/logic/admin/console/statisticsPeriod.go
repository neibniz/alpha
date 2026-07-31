package console

import "time"

type statisticsRange struct {
	Period      string
	Granularity string
	Start       time.Time
	End         time.Time
	Step        time.Duration
	BucketSQL   string
	DateFormat  string
}

const (
	orderTypeSubscribe    = 1
	orderTypeRenewal      = 2
	orderTypeResetTraffic = 3
	orderTypeRecharge     = 4
	orderTypeMembership   = 5

	orderStatusPaid     = 2
	orderStatusFinished = 5
)

var paidOrderStatuses = []int64{orderStatusPaid, orderStatusFinished}
var businessOrderTypes = []int64{orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic, orderTypeMembership}

func resolveStatisticsRange(period string, now time.Time) statisticsRange {
	end := startOfDay(now).AddDate(0, 0, 1)
	switch period {
	case "30d", "month":
		return statisticsRange{
			Period:      "30d",
			Granularity: "day",
			Start:       end.AddDate(0, 0, -30),
			End:         end,
			Step:        24 * time.Hour,
			BucketSQL:   "DATE_TRUNC('day', created_at)",
			DateFormat:  time.DateOnly,
		}
	case "1y", "year":
		start := startOfWeek(now.AddDate(-1, 0, 0))
		return statisticsRange{
			Period:      "1y",
			Granularity: "week",
			Start:       start,
			End:         startOfWeek(now).AddDate(0, 0, 7),
			Step:        7 * 24 * time.Hour,
			BucketSQL:   "DATE_TRUNC('week', created_at)",
			DateFormat:  time.DateOnly,
		}
	default:
		return statisticsRange{
			Period:      "7d",
			Granularity: "day",
			Start:       end.AddDate(0, 0, -7),
			End:         end,
			Step:        24 * time.Hour,
			BucketSQL:   "DATE_TRUNC('day', created_at)",
			DateFormat:  time.DateOnly,
		}
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	day := startOfDay(t)
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return day.AddDate(0, 0, 1-weekday)
}

func (r statisticsRange) eachBucket(fn func(time.Time)) {
	if r.Step <= 0 || !r.Start.Before(r.End) {
		return
	}
	for bucket := r.Start; bucket.Before(r.End); bucket = bucket.Add(r.Step) {
		fn(bucket)
	}
}

func (r statisticsRange) bucketCount() int {
	if r.Step <= 0 || !r.Start.Before(r.End) {
		return 0
	}
	duration := r.End.Sub(r.Start)
	count := int(duration / r.Step)
	if duration%r.Step != 0 {
		count++
	}
	return count
}

func (r statisticsRange) bucketIndex(bucket time.Time) (int, bool) {
	return r.bucketIndexWithCount(bucket, r.bucketCount())
}

func (r statisticsRange) bucketIndexWithCount(bucket time.Time, count int) (int, bool) {
	if r.Step <= 0 || !r.Start.Before(r.End) {
		return 0, false
	}
	normalized := bucket.In(r.Start.Location())
	if normalized.Before(r.Start) || !normalized.Before(r.End) {
		return 0, false
	}
	offset := normalized.Sub(r.Start)
	if offset%r.Step != 0 {
		return 0, false
	}
	index := int(offset / r.Step)
	if index < 0 || index >= count {
		return 0, false
	}
	return index, true
}

func (r statisticsRange) buckets() []time.Time {
	count := r.bucketCount()
	if count == 0 {
		return []time.Time{}
	}
	buckets := make([]time.Time, 0, count)
	for bucket := r.Start; bucket.Before(r.End); bucket = bucket.Add(r.Step) {
		buckets = append(buckets, bucket)
	}
	return buckets
}

type statisticsBucketSeries struct {
	r     statisticsRange
	dates []string
}

func newStatisticsBucketSeries(r statisticsRange) statisticsBucketSeries {
	count := r.bucketCount()
	if count == 0 {
		return statisticsBucketSeries{r: r}
	}
	dates := make([]string, count)
	for i, bucket := 0, r.Start; i < count; i, bucket = i+1, bucket.Add(r.Step) {
		dates[i] = bucket.Format(r.DateFormat)
	}
	return statisticsBucketSeries{
		r:     r,
		dates: dates,
	}
}

func (s statisticsBucketSeries) len() int {
	return len(s.dates)
}

func (s statisticsBucketSeries) index(bucket time.Time) (int, bool) {
	return s.r.bucketIndexWithCount(bucket, len(s.dates))
}

func (s statisticsBucketSeries) date(index int) string {
	return s.dates[index]
}
