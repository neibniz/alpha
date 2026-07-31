package traffic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"

	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/nodeMultiplier"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//goland:noinspection GoNameStartsWithPackageName
type TrafficStatisticsLogic struct {
	svc *svc.ServiceContext
}

const smallTrafficStatisticsIndexThreshold = 8

const (
	trafficReportReceiptRetention = 35 * 24 * time.Hour
	trafficReportCleanupInterval  = 24 * time.Hour
	trafficReportCleanupLockKey   = "traffic:report:receipt:cleanup"
)

// These legacy namespace seeds are persisted through deterministic event IDs and must remain stable.
var trafficReportEventNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("alpha-panel/traffic-report-event"))
var legacyTrafficReportNamespace = uuid.NewSHA1(uuid.NameSpaceOID, []byte("alpha-panel/legacy-traffic-report"))

var trafficStatisticsSubscribeColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"token",
	"expire_time",
	"traffic",
	"download",
	"upload",
	"traffic_reset_at",
	"status",
}

type trafficStatisticsReport struct {
	SubscribeID int64
	Download    int64
	Upload      int64
}

type trafficStatisticsTotal struct {
	Download int64
	Upload   int64
}

type trafficStatisticsTotalRow struct {
	SubscribeID int64
	Total       trafficStatisticsTotal
}

type trafficStatisticsTotals struct {
	rows    []trafficStatisticsTotalRow
	byID    map[int64]int
	capHint int
}

type subscribeTrafficDeltaRow struct {
	SubscribeID int64
	Download    int64
	Upload      int64
}

type dailyTrafficBatch struct {
	UserRows []trafficModel.DailyUserTraffic
	Server   trafficModel.DailyServerTraffic
	Stat     trafficModel.DailyTrafficStat
}

type acceptedTrafficReport struct {
	ReportID  uuid.UUID
	SampledAt time.Time
	Totals    trafficStatisticsTotals
}

func NewTrafficStatisticsLogic(svc *svc.ServiceContext) *TrafficStatisticsLogic {
	return &TrafficStatisticsLogic{
		svc: svc,
	}
}

func (l *TrafficStatisticsLogic) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := decodeTrafficStatisticsTask(task)
	if err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Unmarshal payload failed",
			logger.Field("error", err.Error()),
			logger.Field("payload_bytes", len(task.Payload())),
		)
		return nil
	}
	reportPayloads := normalizeTaskTrafficReports(payload, task.Payload(), time.Now())
	if len(reportPayloads) == 0 {
		logger.WithContext(ctx).Error("[TrafficStatistics] Payload is empty")
		return nil
	}
	processedAt := time.Now()
	changed := false
	var updatedSubscribes []*user.Subscribe
	var exhaustedProductIDs []int64
	if err := l.svc.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		accepted, totals, err := l.acceptTrafficReports(ctx, tx, payload.ServerId, reportPayloads, processedAt)
		if err != nil {
			return err
		}
		if len(accepted) == 0 || totals.Len() == 0 {
			return nil
		}
		subscribeIDs := totals.SortedSubscribeIDs()
		subscribes, err := l.queryTrafficSubscribes(ctx, tx, payload.ServerId, subscribeIDs)
		if err != nil {
			return err
		}
		if subscribes.Len() < len(subscribeIDs) {
			logger.WithContext(ctx).Error("[TrafficStatistics] rejected unassigned or unknown subscriptions",
				logger.Field("server_id", payload.ServerId),
				logger.Field("reported_count", len(subscribeIDs)),
				logger.Field("authorized_count", subscribes.Len()),
			)
		}

		trafficLogs := make([]trafficModel.TrafficLog, 0, totals.Len())
		outboxRows := make([]trafficModel.EventOutbox, 0, totals.Len())
		for _, report := range accepted {
			reportRows := trafficStatisticsReportsFromTotals(report.Totals)
			logs := buildTrafficLogRows(ctx, report.SampledAt, payload.ServerId, reportRows, subscribes)
			trafficLogs = append(trafficLogs, logs...)
			outboxRows = append(outboxRows, buildTrafficEventOutboxRows(report.ReportID, logs)...)
		}
		if len(trafficLogs) == 0 {
			return nil
		}

		currentTotals := buildCurrentTrafficTotals(accepted, subscribes)
		deltaRows := buildSubscribeTrafficDeltaRows(subscribes, currentTotals)
		if err := updateSubscribeTrafficDeltas(ctx, tx, deltaRows); err != nil {
			logger.WithContext(ctx).Error("[TrafficStatistics] Batch update user subscribes with log failed",
				logger.Field("subscribe_count", len(deltaRows)),
				logger.Field("error", err.Error()),
			)
			return err
		}
		exhaustedIDs, productIDs := newlyExhaustedTrafficSubscriptions(subscribes, deltaRows, processedAt)
		if err := markTrafficSubscriptionsExhausted(ctx, tx, exhaustedIDs, processedAt); err != nil {
			return err
		}
		exhaustedProductIDs = productIDs
		for _, row := range deltaRows {
			if sub := subscribes.Get(row.SubscribeID); sub != nil {
				updatedSubscribes = append(updatedSubscribes, sub)
			}
		}

		if err = tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&outboxRows, 1000).Error; err != nil {
			logger.WithContext(ctx).Error("[TrafficStatistics] Create traffic analytics outbox failed",
				logger.Field("log_count", len(trafficLogs)),
				logger.Field("error", err.Error()),
			)
			return err
		}

		for _, dailyBatch := range buildDailyTrafficBatches(accepted, payload.ServerId, subscribes) {
			if err := l.upsertDailyTrafficBatch(ctx, tx, processedAt, dailyBatch); err != nil {
				return err
			}
		}
		changed = true
		return nil
	}); err != nil {
		return err
	}
	if changed && len(updatedSubscribes) > 0 {
		if err := l.svc.UserModel.ClearSubscribeCache(ctx, updatedSubscribes...); err != nil {
			logger.WithContext(ctx).Error("[TrafficStatistics] Clear user subscribe cache failed",
				logger.Field("subscribe_count", len(updatedSubscribes)),
				logger.Field("error", err.Error()),
			)
		}
	}
	if len(exhaustedProductIDs) > 0 {
		if err := l.svc.SubscribeModel.ClearCache(ctx, exhaustedProductIDs...); err != nil {
			logger.WithContext(ctx).Error("[TrafficStatistics] Clear exhausted subscription server caches failed",
				logger.Field("product_count", len(exhaustedProductIDs)),
				logger.Field("error", err.Error()),
			)
		}
	}
	if changed && l.svc.DashboardRealtime != nil {
		l.svc.DashboardRealtime.Publish(realtime.DashboardEventTraffic)
	}
	if changed {
		l.enqueueAnalyticsSync(ctx)
	}
	l.cleanupTrafficReportReceipts(ctx, processedAt)
	return nil
}

func decodeTrafficStatisticsTask(task *asynq.Task) (types.TrafficStatistics, error) {
	if task.Type() == types.ForthwithTrafficStatisticsV2 {
		payload, err := agentprotocol.UnmarshalTrafficTask(task.Payload())
		if err != nil {
			return types.TrafficStatistics{}, err
		}
		return types.TrafficStatistics{
			ServerId: payload.ServerID,
			Protocol: payload.Protocol,
			Reports:  payload.Reports,
		}, nil
	}
	var payload types.TrafficStatistics
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return types.TrafficStatistics{}, err
	}
	return payload, nil
}

func normalizeTaskTrafficReports(payload types.TrafficStatistics, rawPayload []byte, now time.Time) []types.TrafficReport {
	if len(payload.Reports) > 0 {
		return payload.Reports
	}
	if len(payload.Logs) == 0 {
		return nil
	}
	return []types.TrafficReport{{
		ReportId:        uuid.NewSHA1(legacyTrafficReportNamespace, rawPayload).String(),
		SampledAt:       now.UnixMilli(),
		IntervalSeconds: 30,
		Sequence:        1,
		Traffic:         payload.Logs,
	}}
}

func (l *TrafficStatisticsLogic) acceptTrafficReports(
	ctx context.Context,
	tx *gorm.DB,
	serverID int64,
	reports []types.TrafficReport,
	processedAt time.Time,
) ([]acceptedTrafficReport, trafficStatisticsTotals, error) {
	accepted := make([]acceptedTrafficReport, 0, len(reports))
	totals := newTrafficStatisticsTotals(len(reports))
	for _, report := range reports {
		reportID, err := uuid.Parse(report.ReportId)
		if err != nil || report.SampledAt <= 0 || report.IntervalSeconds < 1 || report.IntervalSeconds > 3600 {
			logger.WithContext(ctx).Error("[TrafficStatistics] Ignore invalid traffic report",
				logger.Field("report_id", report.ReportId),
				logger.Field("sampled_at", report.SampledAt),
				logger.Field("interval_seconds", report.IntervalSeconds),
			)
			continue
		}
		sampledAt := time.UnixMilli(report.SampledAt).UTC()
		result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&trafficModel.ReportReceipt{
			ReportId:        reportID,
			ServerId:        serverID,
			SampledAt:       sampledAt,
			IntervalSeconds: int32(report.IntervalSeconds),
			Sequence:        report.Sequence,
			ProcessedAt:     processedAt,
		})
		if result.Error != nil {
			return nil, trafficStatisticsTotals{}, result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}

		multiplierScale := l.svc.NodeMultiplierManager().GetMultiplierScale(sampledAt)
		_, _, reportTotals := collectTrafficStatisticsBatch(
			report.Traffic,
			l.svc.ConfigSnapshot().Node.TrafficReportThreshold,
			multiplierScale,
		)
		if reportTotals.Len() == 0 {
			continue
		}
		for _, row := range reportTotals.rows {
			totals.Add(row.SubscribeID, row.Total.Download, row.Total.Upload)
		}
		accepted = append(accepted, acceptedTrafficReport{
			ReportID:  reportID,
			SampledAt: sampledAt,
			Totals:    reportTotals,
		})
	}
	return accepted, totals, nil
}

func (l *TrafficStatisticsLogic) cleanupTrafficReportReceipts(ctx context.Context, now time.Time) {
	if l.svc.Redis == nil {
		return
	}
	acquired, err := l.svc.Redis.SetNX(ctx, trafficReportCleanupLockKey, now.Unix(), trafficReportCleanupInterval).Result()
	if err != nil || !acquired {
		return
	}
	if err := l.svc.DB.WithContext(ctx).
		Where("processed_at < ?", now.Add(-trafficReportReceiptRetention)).
		Delete(&trafficModel.ReportReceipt{}).Error; err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Cleanup traffic report receipts failed",
			logger.Field("error", err.Error()),
		)
	}
}

func (l *TrafficStatisticsLogic) enqueueAnalyticsSync(ctx context.Context) {
	task := asynq.NewTask(types.ForthwithTrafficAnalyticsSync, nil, asynq.MaxRetry(5), asynq.Unique(10*time.Second))
	if _, err := l.svc.Queue.EnqueueContext(ctx, task); err != nil && !errors.Is(err, asynq.ErrDuplicateTask) {
		logger.WithContext(ctx).Error("[TrafficStatistics] enqueue analytics sync failed", logger.Field("error", err.Error()))
	}
}

func collectTrafficStatisticsBatch(logs []types.UserTraffic, threshold int64, multiplierScale int64) ([]trafficStatisticsReport, []int64, trafficStatisticsTotals) {
	reports := viewutil.NewRowsCollector[trafficStatisticsReport](len(logs))
	totals := newTrafficStatisticsTotals(len(logs))
	subscribeIDs := viewutil.NewRowsCollector[int64](len(logs))
	for _, item := range logs {
		if item.SID <= 0 || item.Download < 0 || item.Upload < 0 || !trafficExceedsThreshold(item.Download, item.Upload, threshold) {
			continue
		}
		download := nodeMultiplier.ApplyTrafficMultiplier(item.Download, multiplierScale)
		upload := nodeMultiplier.ApplyTrafficMultiplier(item.Upload, multiplierScale)
		if download < 0 || upload < 0 || (download == 0 && upload == 0) {
			continue
		}
		reports.Add(trafficStatisticsReport{
			SubscribeID: item.SID,
			Download:    download,
			Upload:      upload,
		})
		if totals.Add(item.SID, download, upload) {
			subscribeIDs.Add(item.SID)
		}
	}
	return reports.Rows(), subscribeIDs.Rows(), totals
}

func trafficStatisticsReportsFromTotals(totals trafficStatisticsTotals) []trafficStatisticsReport {
	ids := totals.SortedSubscribeIDs()
	reports := make([]trafficStatisticsReport, 0, len(ids))
	for _, subscribeID := range ids {
		total, ok := totals.Get(subscribeID)
		if !ok || (total.Download <= 0 && total.Upload <= 0) {
			continue
		}
		reports = append(reports, trafficStatisticsReport{
			SubscribeID: subscribeID,
			Download:    total.Download,
			Upload:      total.Upload,
		})
	}
	return reports
}

func newTrafficStatisticsTotals(capHint int) trafficStatisticsTotals {
	return trafficStatisticsTotals{
		capHint: capHint,
	}
}

func (t *trafficStatisticsTotals) Add(subscribeID int64, download int64, upload int64) bool {
	if subscribeID <= 0 {
		return false
	}
	if t.byID != nil {
		if pos, ok := t.byID[subscribeID]; ok {
			t.rows[pos].Total.Download = saturatingTrafficAdd(t.rows[pos].Total.Download, download)
			t.rows[pos].Total.Upload = saturatingTrafficAdd(t.rows[pos].Total.Upload, upload)
			return false
		}
		t.byID[subscribeID] = len(t.rows)
		t.rows = append(t.rows, trafficStatisticsTotalRow{
			SubscribeID: subscribeID,
			Total: trafficStatisticsTotal{
				Download: download,
				Upload:   upload,
			},
		})
		return true
	}
	for pos := range t.rows {
		if t.rows[pos].SubscribeID == subscribeID {
			t.rows[pos].Total.Download = saturatingTrafficAdd(t.rows[pos].Total.Download, download)
			t.rows[pos].Total.Upload = saturatingTrafficAdd(t.rows[pos].Total.Upload, upload)
			return false
		}
	}
	t.ensureRows()
	t.rows = append(t.rows, trafficStatisticsTotalRow{
		SubscribeID: subscribeID,
		Total: trafficStatisticsTotal{
			Download: download,
			Upload:   upload,
		},
	})
	if len(t.rows) > smallTrafficStatisticsIndexThreshold {
		t.upgradeLookup()
	}
	return true
}

func (t trafficStatisticsTotals) Len() int {
	return len(t.rows)
}

func trafficExceedsThreshold(download, upload, threshold int64) bool {
	if download < 0 || upload < 0 {
		return false
	}
	if threshold < 0 || download > threshold {
		return true
	}
	return upload > threshold-download
}

func saturatingTrafficAdd(current, delta int64) int64 {
	if current < 0 || delta < 0 {
		return 0
	}
	if current > math.MaxInt64-delta {
		return math.MaxInt64
	}
	return current + delta
}

func (t trafficStatisticsTotals) Get(subscribeID int64) (trafficStatisticsTotal, bool) {
	if t.byID != nil {
		pos, ok := t.byID[subscribeID]
		if !ok {
			return trafficStatisticsTotal{}, false
		}
		return t.rows[pos].Total, true
	}
	for _, row := range t.rows {
		if row.SubscribeID == subscribeID {
			return row.Total, true
		}
	}
	return trafficStatisticsTotal{}, false
}

func (t trafficStatisticsTotals) SortedSubscribeIDs() []int64 {
	ids := make([]int64, 0, len(t.rows))
	for _, row := range t.rows {
		ids = append(ids, row.SubscribeID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (t *trafficStatisticsTotals) ensureRows() {
	if t.rows != nil || t.capHint <= 0 {
		return
	}
	t.rows = make([]trafficStatisticsTotalRow, 0, trafficStatisticsInitialCap(t.capHint))
}

func (t *trafficStatisticsTotals) upgradeLookup() {
	if t.byID != nil {
		return
	}
	nextCap := trafficLookupCap(len(t.rows), t.capHint)
	if nextCap > cap(t.rows) {
		rows := make([]trafficStatisticsTotalRow, len(t.rows), nextCap)
		copy(rows, t.rows)
		t.rows = rows
	}
	t.byID = make(map[int64]int, nextCap)
	for pos, row := range t.rows {
		t.byID[row.SubscribeID] = pos
	}
}

type trafficSubscribeIndex struct {
	rows    []*user.Subscribe
	byID    map[int64]int
	capHint int
}

func indexTrafficSubscribes(subscribes []*user.Subscribe) trafficSubscribeIndex {
	index := trafficSubscribeIndex{
		capHint: len(subscribes),
	}
	for _, sub := range subscribes {
		index.Add(sub)
	}
	return index
}

func (i *trafficSubscribeIndex) Add(sub *user.Subscribe) {
	if sub == nil || sub.Id <= 0 {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[sub.Id]; ok {
			return
		}
		i.byID[sub.Id] = len(i.rows)
		i.rows = append(i.rows, sub)
		return
	}
	for _, existing := range i.rows {
		if existing != nil && existing.Id == sub.Id {
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, sub)
	if len(i.rows) > smallTrafficStatisticsIndexThreshold {
		i.upgradeLookup()
	}
}

func (i trafficSubscribeIndex) Len() int {
	return len(i.rows)
}

func (i trafficSubscribeIndex) Get(subscribeID int64) *user.Subscribe {
	if i.byID != nil {
		pos, ok := i.byID[subscribeID]
		if !ok {
			return nil
		}
		return i.rows[pos]
	}
	for _, sub := range i.rows {
		if sub != nil && sub.Id == subscribeID {
			return sub
		}
	}
	return nil
}

func (i *trafficSubscribeIndex) ensureRows() {
	if i.rows != nil || i.capHint <= 0 {
		return
	}
	i.rows = make([]*user.Subscribe, 0, trafficStatisticsInitialCap(i.capHint))
}

func (i *trafficSubscribeIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := trafficLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]*user.Subscribe, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byID = make(map[int64]int, nextCap)
	for pos, existing := range i.rows {
		if existing != nil {
			i.byID[existing.Id] = pos
		}
	}
}

func trafficLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func trafficStatisticsInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallTrafficStatisticsIndexThreshold+1 {
		return smallTrafficStatisticsIndexThreshold + 1
	}
	return capHint
}

func buildTrafficLogRows(ctx context.Context, now time.Time, serverID int64, reports []trafficStatisticsReport, subscribes trafficSubscribeIndex) []trafficModel.TrafficLog {
	if len(reports) == 0 {
		return nil
	}
	trafficLogs := viewutil.NewRowsCollector[trafficModel.TrafficLog](len(reports))
	for _, report := range reports {
		sub := subscribes.Get(report.SubscribeID)
		if sub == nil {
			logger.WithContext(ctx).Error("[TrafficStatistics] Find user Subscribe Info failed",
				logger.Field("uid", report.SubscribeID),
				logger.Field("error", gorm.ErrRecordNotFound.Error()),
			)
			continue
		}
		trafficLogs.Add(trafficModel.TrafficLog{
			ServerId:    serverID,
			SubscribeId: report.SubscribeID,
			UserId:      sub.UserId,
			Upload:      report.Upload,
			Download:    report.Download,
			Timestamp:   now,
		})
	}
	return trafficLogs.Rows()
}

func buildTrafficEventOutboxRows(reportID uuid.UUID, rows []trafficModel.TrafficLog) []trafficModel.EventOutbox {
	if len(rows) == 0 {
		return nil
	}
	result := make([]trafficModel.EventOutbox, 0, len(rows))
	createdAt := time.Now()
	for _, row := range rows {
		eventID := uuid.NewSHA1(trafficReportEventNamespace, []byte(fmt.Sprintf(
			"%s:%d:%d",
			reportID.String(),
			row.ServerId,
			row.SubscribeId,
		)))
		result = append(result, trafficModel.EventOutbox{
			EventId:         eventID,
			ServerId:        row.ServerId,
			UserId:          row.UserId,
			UserSubscribeId: row.SubscribeId,
			Download:        row.Download,
			Upload:          row.Upload,
			EventTime:       row.Timestamp,
			CreatedAt:       createdAt,
		})
	}
	return result
}

func (l *TrafficStatisticsLogic) queryTrafficSubscribes(ctx context.Context, db *gorm.DB, serverID int64, subscribeIDs []int64) (trafficSubscribeIndex, error) {
	if len(subscribeIDs) == 0 {
		return trafficSubscribeIndex{}, nil
	}
	productIDs, err := queryTrafficAuthorizedProductIDs(ctx, db, serverID)
	if err != nil {
		return trafficSubscribeIndex{}, err
	}
	if len(productIDs) == 0 {
		return trafficSubscribeIndex{}, nil
	}
	var subscribes []*user.Subscribe
	if err := db.WithContext(ctx).
		Model(&user.Subscribe{}).
		Select(trafficStatisticsSubscribeColumns).
		Where("id IN ?", subscribeIDs).
		Where("subscribe_id IN ?", productIDs).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&subscribes).Error; err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Batch find user subscribes failed",
			logger.Field("subscribe_ids", subscribeIDs),
			logger.Field("error", err.Error()),
		)
		return trafficSubscribeIndex{}, err
	}
	return indexTrafficSubscribes(subscribes), nil
}

func queryTrafficAuthorizedProductIDs(ctx context.Context, db *gorm.DB, serverID int64) ([]int64, error) {
	if serverID <= 0 {
		return nil, nil
	}
	var nodes []node.Node
	if err := db.WithContext(ctx).
		Model(&node.Node{}).
		Select("id", "tags").
		Where("server_id = ? AND enabled = ?", serverID, true).
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	nodeIDs, nodeTags := trafficNodeSelectors(nodes)
	if len(nodeIDs) == 0 && len(nodeTags) == 0 {
		return nil, nil
	}
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if len(nodeIDs) > 0 {
		clauses = append(clauses, "string_to_array(COALESCE(nodes, ''), ',') && ?")
		args = append(args, pq.Array(nodeIDs))
	}
	if len(nodeTags) > 0 {
		clauses = append(clauses, "string_to_array(COALESCE(node_tags, ''), ',') && ?")
		args = append(args, pq.Array(nodeTags))
	}
	var productIDs []int64
	err := db.WithContext(ctx).
		Model(&modelSubscribe.Subscribe{}).
		Where("("+strings.Join(clauses, " OR ")+")", args...).
		Pluck("id", &productIDs).Error
	return productIDs, err
}

func trafficNodeSelectors(nodes []node.Node) ([]string, []string) {
	nodeIDs := make([]string, 0, len(nodes))
	nodeTags := make([]string, 0, len(nodes))
	seenIDs := make(map[string]struct{}, len(nodes))
	seenTags := make(map[string]struct{}, len(nodes))
	for _, item := range nodes {
		if item.Id > 0 {
			id := strconv.FormatInt(item.Id, 10)
			if _, ok := seenIDs[id]; !ok {
				seenIDs[id] = struct{}{}
				nodeIDs = append(nodeIDs, id)
			}
		}
		for _, tag := range strings.Split(item.Tags, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				if _, ok := seenTags[tag]; !ok {
					seenTags[tag] = struct{}{}
					nodeTags = append(nodeTags, tag)
				}
			}
		}
	}
	return nodeIDs, nodeTags
}

func buildSubscribeTrafficDeltaRows(subscribes trafficSubscribeIndex, totals trafficStatisticsTotals) []subscribeTrafficDeltaRow {
	if totals.Len() == 0 || subscribes.Len() == 0 {
		return nil
	}
	ids := totals.SortedSubscribeIDs()
	rows := viewutil.NewRowsCollector[subscribeTrafficDeltaRow](len(ids))
	for _, subscribeID := range ids {
		sub := subscribes.Get(subscribeID)
		if sub == nil {
			continue
		}
		total, ok := totals.Get(subscribeID)
		if !ok {
			continue
		}
		if total.Download == 0 && total.Upload == 0 {
			continue
		}
		rows.Add(subscribeTrafficDeltaRow{
			SubscribeID: subscribeID,
			Download:    total.Download,
			Upload:      total.Upload,
		})
	}
	return rows.Rows()
}

func newlyExhaustedTrafficSubscriptions(subscribes trafficSubscribeIndex, rows []subscribeTrafficDeltaRow, now time.Time) ([]int64, []int64) {
	if len(rows) == 0 {
		return nil, nil
	}
	subscribeIDs := make([]int64, 0, len(rows))
	productIDs := make([]int64, 0, len(rows))
	seenProducts := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		sub := subscribes.Get(row.SubscribeID)
		if sub == nil || sub.Status != subscriptionstate.StatusActive || subscriptionstate.IsExpired(sub.ExpireTime, now) {
			continue
		}
		if subscriptionstate.TrafficExhausted(sub.Traffic, sub.Download, sub.Upload) {
			continue
		}
		download := saturatingTrafficAdd(sub.Download, row.Download)
		upload := saturatingTrafficAdd(sub.Upload, row.Upload)
		if !subscriptionstate.TrafficExhausted(sub.Traffic, download, upload) {
			continue
		}
		subscribeIDs = append(subscribeIDs, sub.Id)
		if sub.SubscribeId > 0 {
			if _, ok := seenProducts[sub.SubscribeId]; !ok {
				seenProducts[sub.SubscribeId] = struct{}{}
				productIDs = append(productIDs, sub.SubscribeId)
			}
		}
	}
	return subscribeIDs, productIDs
}

func markTrafficSubscriptionsExhausted(ctx context.Context, db *gorm.DB, subscribeIDs []int64, now time.Time) error {
	if len(subscribeIDs) == 0 {
		return nil
	}
	return db.WithContext(ctx).
		Model(&user.Subscribe{}).
		Where("id IN ? AND status = ?", subscribeIDs, subscriptionstate.StatusActive).
		Updates(map[string]any{
			"status":      subscriptionstate.StatusExhausted,
			"finished_at": now,
		}).Error
}

// buildCurrentTrafficTotals applies the reset barrier only to mutable counters.
// Historical logs and analytics still retain accepted samples from before a reset.
func buildCurrentTrafficTotals(reports []acceptedTrafficReport, subscribes trafficSubscribeIndex) trafficStatisticsTotals {
	totals := newTrafficStatisticsTotals(len(reports))
	for _, report := range reports {
		for _, row := range report.Totals.rows {
			sub := subscribes.Get(row.SubscribeID)
			if sub == nil || !report.SampledAt.After(sub.TrafficResetAt) {
				continue
			}
			totals.Add(row.SubscribeID, row.Total.Download, row.Total.Upload)
		}
	}
	return totals
}

func updateSubscribeTrafficDeltas(ctx context.Context, db *gorm.DB, rows []subscribeTrafficDeltaRow) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.SubscribeID)
	}
	return db.WithContext(ctx).
		Model(&user.Subscribe{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"download": subscribeTrafficDeltaExpr("download", rows, func(row subscribeTrafficDeltaRow) int64 { return row.Download }),
			"upload":   subscribeTrafficDeltaExpr("upload", rows, func(row subscribeTrafficDeltaRow) int64 { return row.Upload }),
		}).Error
}

func subscribeTrafficDeltaExpr(column string, rows []subscribeTrafficDeltaRow, value func(subscribeTrafficDeltaRow) int64) clause.Expr {
	var sql strings.Builder
	sql.WriteString("CASE id")
	args := make([]any, 0, len(rows)*4)
	for _, row := range rows {
		delta := value(row)
		sql.WriteString(" WHEN ? THEN CASE WHEN ")
		sql.WriteString(column)
		sql.WriteString(" > ? THEN ? ELSE ")
		sql.WriteString(column)
		sql.WriteString(" + ? END")
		args = append(args, row.SubscribeID, math.MaxInt64-delta, int64(math.MaxInt64), delta)
	}
	sql.WriteString(" ELSE ")
	sql.WriteString(column)
	sql.WriteString(" END")
	return gorm.Expr(sql.String(), args...)
}

func buildDailyTrafficBatch(now time.Time, serverID int64, subscribes trafficSubscribeIndex, totals trafficStatisticsTotals) dailyTrafficBatch {
	date := now.Format(time.DateOnly)
	ids := totals.SortedSubscribeIDs()

	batch := dailyTrafficBatch{
		UserRows: make([]trafficModel.DailyUserTraffic, 0, len(ids)),
		Server: trafficModel.DailyServerTraffic{
			Date:     date,
			ServerId: serverID,
		},
		Stat: trafficModel.DailyTrafficStat{
			Date: date,
		},
	}
	for _, subscribeID := range ids {
		total, ok := totals.Get(subscribeID)
		if !ok {
			continue
		}
		sum := saturatingTrafficAdd(total.Download, total.Upload)
		if sum <= 0 {
			continue
		}
		sub := subscribes.Get(subscribeID)
		if sub == nil {
			continue
		}
		batch.UserRows = append(batch.UserRows, trafficModel.DailyUserTraffic{
			Date:        date,
			UserId:      sub.UserId,
			SubscribeId: subscribeID,
			Download:    total.Download,
			Upload:      total.Upload,
			Total:       sum,
		})
		batch.Server.Download = saturatingTrafficAdd(batch.Server.Download, total.Download)
		batch.Server.Upload = saturatingTrafficAdd(batch.Server.Upload, total.Upload)
		batch.Server.Total = saturatingTrafficAdd(batch.Server.Total, sum)
		batch.Stat.Download = saturatingTrafficAdd(batch.Stat.Download, total.Download)
		batch.Stat.Upload = saturatingTrafficAdd(batch.Stat.Upload, total.Upload)
		batch.Stat.Total = saturatingTrafficAdd(batch.Stat.Total, sum)
	}
	return batch
}

func buildDailyTrafficBatches(reports []acceptedTrafficReport, serverID int64, subscribes trafficSubscribeIndex) []dailyTrafficBatch {
	if len(reports) == 0 {
		return nil
	}
	totalsByDate := make(map[string]trafficStatisticsTotals)
	for _, report := range reports {
		date := report.SampledAt.UTC().Format(time.DateOnly)
		totals := totalsByDate[date]
		if totals.rows == nil {
			totals = newTrafficStatisticsTotals(report.Totals.Len())
		}
		for _, row := range report.Totals.rows {
			totals.Add(row.SubscribeID, row.Total.Download, row.Total.Upload)
		}
		totalsByDate[date] = totals
	}
	dates := make([]string, 0, len(totalsByDate))
	for date := range totalsByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	batches := make([]dailyTrafficBatch, 0, len(dates))
	for _, date := range dates {
		day, err := time.Parse(time.DateOnly, date)
		if err != nil {
			continue
		}
		batch := buildDailyTrafficBatch(day, serverID, subscribes, totalsByDate[date])
		if len(batch.UserRows) > 0 {
			batches = append(batches, batch)
		}
	}
	return batches
}

func (l *TrafficStatisticsLogic) upsertDailyTrafficBatch(ctx context.Context, db *gorm.DB, now time.Time, batch dailyTrafficBatch) error {
	if len(batch.UserRows) == 0 {
		return nil
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "user_id"}, {Name: "subscribe_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"download":   dailyTrafficSaturatingAdd("daily_user_traffic", "download"),
			"upload":     dailyTrafficSaturatingAdd("daily_user_traffic", "upload"),
			"total":      dailyTrafficSaturatingAdd("daily_user_traffic", "total"),
			"updated_at": now,
		}),
	}).CreateInBatches(&batch.UserRows, 1000).Error; err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Upsert daily user traffic failed",
			logger.Field("row_count", len(batch.UserRows)),
			logger.Field("error", err.Error()),
		)
		return err
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "server_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"download":   dailyTrafficSaturatingAdd("daily_server_traffic", "download"),
			"upload":     dailyTrafficSaturatingAdd("daily_server_traffic", "upload"),
			"total":      dailyTrafficSaturatingAdd("daily_server_traffic", "total"),
			"updated_at": now,
		}),
	}).Create(&batch.Server).Error; err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Upsert daily server traffic failed",
			logger.Field("server_id", batch.Server.ServerId),
			logger.Field("error", err.Error()),
		)
		return err
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"download":   dailyTrafficSaturatingAdd("daily_traffic_stat", "download"),
			"upload":     dailyTrafficSaturatingAdd("daily_traffic_stat", "upload"),
			"total":      dailyTrafficSaturatingAdd("daily_traffic_stat", "total"),
			"updated_at": now,
		}),
	}).Create(&batch.Stat).Error; err != nil {
		logger.WithContext(ctx).Error("[TrafficStatistics] Upsert daily traffic stat failed",
			logger.Field("error", err.Error()),
		)
		return err
	}
	return nil
}

func dailyTrafficSaturatingAdd(table, column string) clause.Expr {
	return clause.Expr{
		SQL: fmt.Sprintf(
			"CASE WHEN %s.%s > ? - EXCLUDED.%s THEN ? ELSE %s.%s + EXCLUDED.%s END",
			table, column, column, table, column, column,
		),
		Vars: []any{int64(math.MaxInt64), int64(math.MaxInt64)},
	}
}
