package subscription

import (
	"context"
	"time"

	queue "github.com/neibniz/alpha/server/queue/types"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CheckSubscriptionLogic struct {
	svc *svc.ServiceContext
}

const (
	smallCheckSubscriptionIndexThreshold      = 8
	smallCheckSubscriptionEmailIndexThreshold = smallCheckSubscriptionIndexThreshold
	checkSubscriptionBatchSize                = 500
	checkSubscriptionMaxBatches               = 20
)

type subscriptionDueKind uint8

const (
	subscriptionDueExpired subscriptionDueKind = iota + 1
	subscriptionDueTraffic
)

var checkSubscriptionCacheColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"token",
	"expire_time",
}

var checkSubscriptionEmailAuthColumns = []string{
	"user_id",
	"auth_identifier",
}

type subscriptionFinishUpdate struct {
	Status     uint8
	FinishedAt time.Time
}

func (u subscriptionFinishUpdate) updates() map[string]any {
	return map[string]any{
		"status":      u.Status,
		"finished_at": u.FinishedAt,
	}
}

func NewCheckSubscriptionLogic(svc *svc.ServiceContext) *CheckSubscriptionLogic {
	return &CheckSubscriptionLogic{
		svc: svc,
	}
}

func (l *CheckSubscriptionLogic) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	logger.Infof("[CheckSubscription] Start check subscription: %s", time.Now().Format("2006-01-02 15:04:05"))
	now := time.Now().UTC()
	// Expiry has priority when a subscription is both expired and over quota.
	if err := l.processDueSubscriptions(ctx, subscriptionDueExpired, now); err != nil {
		logger.Errorw("[CheckSubscription] expiry batches failed", logger.Field("error", err.Error()))
		return err
	}
	if err := l.processDueSubscriptions(ctx, subscriptionDueTraffic, now); err != nil {
		logger.Errorw("[CheckSubscription] traffic batches failed", logger.Field("error", err.Error()))
		return err
	}
	return nil
}

func (l *CheckSubscriptionLogic) processDueSubscriptions(ctx context.Context, kind subscriptionDueKind, now time.Time) error {
	total := 0
	for batch := 0; batch < checkSubscriptionMaxBatches; batch++ {
		list, err := l.finishNextSubscriptionBatch(ctx, kind, now)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			break
		}
		total += len(list)
		if kind == subscriptionDueExpired {
			if err := l.sendExpiredNotify(ctx, list); err != nil {
				logger.Errorw("[CheckSubscription] queue expiry notifications failed", logger.Field("error", err.Error()))
			}
		} else if err := l.sendTrafficNotify(ctx, list); err != nil {
			logger.Errorw("[CheckSubscription] queue traffic notifications failed", logger.Field("error", err.Error()))
		}
		if err := l.svc.UserModel.ClearSubscribeCache(ctx, list...); err != nil {
			logger.Errorw("[CheckSubscription] clear subscription cache failed", logger.Field("error", err.Error()))
		}
		l.clearServerCache(ctx, list...)
		if len(list) < checkSubscriptionBatchSize {
			break
		}
	}
	logger.Infow("[CheckSubscription] processed due subscriptions",
		logger.Field("kind", kind.String()),
		logger.Field("count", total),
	)
	return nil
}

func (l *CheckSubscriptionLogic) finishNextSubscriptionBatch(ctx context.Context, kind subscriptionDueKind, now time.Time) ([]*user.Subscribe, error) {
	list := make([]*user.Subscribe, 0, checkSubscriptionBatchSize)
	err := l.svc.UserModel.Transaction(ctx, func(db *gorm.DB) error {
		query := db.WithContext(ctx).
			Model(&user.Subscribe{}).
			Select(checkSubscriptionCacheColumns).
			Limit(checkSubscriptionBatchSize).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		switch kind {
		case subscriptionDueExpired:
			query = query.
				Where("status IN ?", []int64{int64(subscriptionstate.StatusActive), int64(subscriptionstate.StatusExhausted)}).
				Where("expire_time <> to_timestamp(0) AND expire_time <= ?", now).
				Order("expire_time ASC, id ASC")
		case subscriptionDueTraffic:
			query = query.
				Where("status = ?", subscriptionstate.StatusActive).
				Where("traffic > 0 AND used_traffic >= traffic").
				Where("expire_time = to_timestamp(0) OR expire_time > ?", now).
				Order("id ASC")
		default:
			return nil
		}
		if err := query.Find(&list).Error; err != nil || len(list) == 0 {
			return err
		}
		ids := make([]int64, 0, len(list))
		for _, item := range list {
			if item != nil {
				ids = append(ids, item.Id)
			}
		}
		status := subscriptionstate.StatusExhausted
		currentStatuses := []int64{int64(subscriptionstate.StatusActive)}
		if kind == subscriptionDueExpired {
			status = subscriptionstate.StatusExpired
			currentStatuses = []int64{int64(subscriptionstate.StatusActive), int64(subscriptionstate.StatusExhausted)}
		}
		return db.WithContext(ctx).
			Model(&user.Subscribe{}).
			Where("id IN ? AND status IN ?", ids, currentStatuses).
			Updates(subscriptionFinishUpdate{Status: status, FinishedAt: now}.updates()).Error
	})
	return list, err
}

func (k subscriptionDueKind) String() string {
	switch k {
	case subscriptionDueExpired:
		return "expired"
	case subscriptionDueTraffic:
		return "traffic"
	default:
		return "unknown"
	}
}

func (l *CheckSubscriptionLogic) sendExpiredNotify(ctx context.Context, subs []*user.Subscribe) error {
	site := l.svc.ConfigSnapshot().Site
	emailIndex, err := l.queryEmailAuthMethods(ctx, subs)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		email, ok := emailIndex.Find(sub.UserId)
		if !ok {
			logger.Errorw("[CheckSubscription] email auth method not found", logger.Field("user_id", sub.UserId))
			continue
		}
		var taskPayload queue.SendEmailPayload
		taskPayload.Type = queue.EmailTypeExpiration
		taskPayload.Email = email
		taskPayload.Subject = "Subscription Expired"
		taskPayload.Content, err = queue.EncodeEmailContent(queue.ExpirationEmailContent{
			SiteLogo:   site.SiteLogo,
			SiteName:   site.SiteName,
			ExpireDate: sub.ExpireTime.Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			logger.Errorw("[CheckSubscription] Encode payload content failed", logger.Field("error", err.Error()))
			continue
		}
		payloadBuy, err := queue.EncodePrivatePayload(l.svc.Privacy, queue.EmailPayloadScope, taskPayload)
		if err != nil {
			logger.Errorw("[CheckSubscription] Marshal payload failed", logger.Field("error", err.Error()))
			continue
		}
		task := asynq.NewTask(queue.ForthwithSendEmail, payloadBuy, asynq.MaxRetry(3))
		taskInfo, err := l.svc.Queue.Enqueue(task)
		if err != nil {
			logger.Errorw("[CheckSubscription] Enqueue task failed", logger.Field("error", err.Error()))
			continue
		}
		logger.Infow("[CheckSubscription] Send email success",
			logger.Field("taskID", taskInfo.ID), logger.Field("User", sub.UserId),
		)
	}
	return nil
}

func (l *CheckSubscriptionLogic) sendTrafficNotify(ctx context.Context, subs []*user.Subscribe) error {
	site := l.svc.ConfigSnapshot().Site
	emailIndex, err := l.queryEmailAuthMethods(ctx, subs)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		email, ok := emailIndex.Find(sub.UserId)
		if !ok {
			logger.Errorw("[CheckSubscription] email auth method not found", logger.Field("user_id", sub.UserId))
			continue
		}
		var taskPayload queue.SendEmailPayload
		taskPayload.Type = queue.EmailTypeTrafficExceed
		taskPayload.Email = email
		taskPayload.Subject = "Subscription Traffic Exceed"
		taskPayload.Content, err = queue.EncodeEmailContent(queue.SiteEmailContent{
			SiteLogo: site.SiteLogo,
			SiteName: site.SiteName,
		})
		if err != nil {
			logger.Errorw("[CheckSubscription] Encode payload content failed", logger.Field("error", err.Error()))
			continue
		}
		payloadBuy, err := queue.EncodePrivatePayload(l.svc.Privacy, queue.EmailPayloadScope, taskPayload)
		if err != nil {
			logger.Errorw("[CheckSubscription] Marshal payload failed", logger.Field("error", err.Error()))
			continue
		}
		task := asynq.NewTask(queue.ForthwithSendEmail, payloadBuy, asynq.MaxRetry(3))
		taskInfo, err := l.svc.Queue.Enqueue(task)
		if err != nil {
			logger.Errorw("[CheckSubscription] Enqueue task failed", logger.Field("error", err.Error()))
			continue
		}
		logger.Infow("[CheckSubscription] Send email success",
			logger.Field("taskID", taskInfo.ID), logger.Field("User", sub.UserId),
		)
	}
	return nil
}

func (l *CheckSubscriptionLogic) queryEmailAuthMethods(ctx context.Context, subs []*user.Subscribe) (checkSubscriptionEmailIndex, error) {
	userIDs := checkSubscriptionUserIDs(subs)
	if len(userIDs) == 0 {
		return checkSubscriptionEmailIndex{}, nil
	}
	methods, err := l.svc.UserModel.FindUserAuthMethodsByType(ctx, "email", userIDs, true)
	if err != nil {
		logger.Errorw("[CheckSubscription] Batch find email auth methods failed", logger.Field("error", err.Error()), logger.Field("user_ids", userIDs))
		return checkSubscriptionEmailIndex{}, err
	}
	return newCheckSubscriptionEmailIndex(methods), nil
}

type checkSubscriptionEmailEntry struct {
	UserID int64
	Email  string
}

type checkSubscriptionEmailIndex struct {
	rows    []checkSubscriptionEmailEntry
	byID    map[int64]int
	capHint int
}

func newCheckSubscriptionEmailIndex(methods []*user.AuthMethods) checkSubscriptionEmailIndex {
	index := checkSubscriptionEmailIndex{
		capHint: len(methods),
	}
	for _, method := range methods {
		index.Add(method)
	}
	return index
}

func (i *checkSubscriptionEmailIndex) Add(method *user.AuthMethods) {
	if method == nil || method.UserId <= 0 || method.AuthIdentifier == "" {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[method.UserId]; ok {
			return
		}
		i.byID[method.UserId] = len(i.rows)
		i.rows = append(i.rows, checkSubscriptionEmailEntry{
			UserID: method.UserId,
			Email:  method.AuthIdentifier,
		})
		return
	}
	for _, row := range i.rows {
		if row.UserID == method.UserId {
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, checkSubscriptionEmailEntry{
		UserID: method.UserId,
		Email:  method.AuthIdentifier,
	})
	if len(i.rows) > smallCheckSubscriptionEmailIndexThreshold {
		i.upgradeLookup()
	}
}

func (i checkSubscriptionEmailIndex) Find(userID int64) (string, bool) {
	if i.byID != nil {
		pos, ok := i.byID[userID]
		if !ok {
			return "", false
		}
		return i.rows[pos].Email, true
	}
	for _, row := range i.rows {
		if row.UserID == userID {
			return row.Email, true
		}
	}
	return "", false
}

func (i *checkSubscriptionEmailIndex) ensureRows() {
	if i.rows != nil || i.capHint <= 0 {
		return
	}
	i.rows = make([]checkSubscriptionEmailEntry, 0, checkSubscriptionEmailInitialCap(i.capHint))
}

func (i *checkSubscriptionEmailIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := checkSubscriptionLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]checkSubscriptionEmailEntry, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byID = make(map[int64]int, nextCap)
	for pos, row := range i.rows {
		i.byID[row.UserID] = pos
	}
}

func checkSubscriptionEmailInitialCap(capHint int) int {
	return checkSubscriptionIndexInitialCap(capHint)
}

func checkSubscriptionIndexInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallCheckSubscriptionIndexThreshold+1 {
		return smallCheckSubscriptionIndexThreshold + 1
	}
	return capHint
}

func checkSubscriptionUserIDs(subs []*user.Subscribe) []int64 {
	userIDs := newCheckSubscriptionDeduper[int64](len(subs))
	for _, sub := range subs {
		if sub == nil || sub.UserId <= 0 {
			continue
		}
		userIDs.Add(sub.UserId)
	}
	return userIDs.Values()
}

func (l *CheckSubscriptionLogic) clearServerCache(ctx context.Context, userSubs ...*user.Subscribe) {
	snapshots, subscribeIDs := checkSubscriptionCacheTargets(userSubs)
	for _, snapshot := range snapshots {
		if err := subscribecache.ClearBySnapshot(ctx, l.svc.DB, l.svc.Redis, snapshot); err != nil {
			logger.Errorw("[CheckSubscription] Clear snapshot server user cache failed", logger.Field("error", err.Error()))
		}
	}

	for _, subscribeID := range subscribeIDs {
		if err := l.svc.SubscribeModel.ClearCache(ctx, subscribeID); err != nil {
			logger.Errorw("[CheckSubscription] ClearCache failed", logger.Field("error", err.Error()), logger.Field("subscribe_id", subscribeID))
		}
	}
}

func checkSubscriptionCacheTargets(userSubs []*user.Subscribe) ([]string, []int64) {
	snapshots := newCheckSubscriptionDeduper[string](len(userSubs))
	subscribeIDs := newCheckSubscriptionDeduper[int64](len(userSubs))
	for _, sub := range userSubs {
		if sub == nil {
			continue
		}
		if sub.ProductSnapshot != "" {
			snapshots.Add(sub.ProductSnapshot)
		}
		if sub.SubscribeId > 0 {
			subscribeIDs.Add(sub.SubscribeId)
		}
	}
	return snapshots.Values(), subscribeIDs.Values()
}

type checkSubscriptionDeduper[T comparable] struct {
	values  []T
	seen    map[T]struct{}
	capHint int
}

func newCheckSubscriptionDeduper[T comparable](capHint int) checkSubscriptionDeduper[T] {
	return checkSubscriptionDeduper[T]{capHint: capHint}
}

func (d *checkSubscriptionDeduper[T]) Add(value T) bool {
	if d.seen != nil {
		if _, ok := d.seen[value]; ok {
			return false
		}
		d.seen[value] = struct{}{}
		d.values = append(d.values, value)
		return true
	}
	for _, existing := range d.values {
		if existing == value {
			return false
		}
	}
	if d.values == nil {
		d.values = make([]T, 0, checkSubscriptionIndexInitialCap(d.capHint))
	}
	d.values = append(d.values, value)
	if len(d.values) > smallCheckSubscriptionIndexThreshold {
		d.upgradeLookup()
	}
	return true
}

func (d checkSubscriptionDeduper[T]) Values() []T {
	if len(d.values) == 0 {
		return []T{}
	}
	return d.values
}

func (d *checkSubscriptionDeduper[T]) upgradeLookup() {
	if d.seen != nil {
		return
	}
	nextCap := checkSubscriptionLookupCap(len(d.values), d.capHint)
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

func checkSubscriptionLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
