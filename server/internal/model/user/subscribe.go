package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/model/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (m *defaultUserModel) UpdateUserSubscribeCache(ctx context.Context, data *Subscribe) error {
	return m.ClearSubscribeCacheByModels(ctx, data)
}

// QueryActiveSubscriptions returns the number of active subscriptions.
func (m *defaultUserModel) QueryActiveSubscriptions(ctx context.Context, subscribeId ...int64) (map[int64]int64, error) {
	subscribeId = normalizeUserCacheIDs(subscribeId)
	if len(subscribeId) == 0 {
		return nil, nil
	}

	type SubscriptionCount struct {
		SubscribeId int64
		Total       int64
	}
	var result []SubscriptionCount
	err := m.QueryNoCacheCtx(ctx, &result, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Where("subscribe_id IN ? AND status = ?", subscribeId, 1).
			Select("subscribe_id, COUNT(id) as total").
			Group("subscribe_id").
			Scan(v).
			Error
	})

	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	resultMap := make(map[int64]int64, len(result))
	for _, item := range result {
		resultMap[item.SubscribeId] = item.Total
	}

	return resultMap, nil
}

func (m *defaultUserModel) FindOneSubscribeByOrderId(ctx context.Context, orderId int64) (*Subscribe, error) {
	var data Subscribe
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeOrderLookupColumns).
			Where("order_id = ?", orderId).
			First(v).Error
	})
	return &data, err
}

func (m *defaultUserModel) FindOneSubscribe(ctx context.Context, id int64) (*Subscribe, error) {
	var data Subscribe
	key := cachekeys.Int(cacheUserSubscribeIdPrefix, id)
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).Where("id = ?", id).First(&data).Error
	})
	return &data, err
}

func (m *defaultUserModel) FindOneSubscribeForMutation(ctx context.Context, id int64, tx ...*gorm.DB) (*Subscribe, error) {
	var data Subscribe
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		if len(tx) > 0 && tx[0] != nil {
			conn = tx[0].WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"})
		}
		return conn.Model(&Subscribe{}).
			Select(userSubscribeMutationColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (m *defaultUserModel) FindOneSubscribeForPreset(ctx context.Context, id int64) (*Subscribe, error) {
	var data Subscribe
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribePresetColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (m *defaultUserModel) findOneSubscribeForCache(ctx context.Context, id int64) (*Subscribe, error) {
	var data Subscribe
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (m *defaultUserModel) FindUsersSubscribeBySubscribeId(ctx context.Context, subscribeId int64) ([]*Subscribe, error) {
	data := make([]*Subscribe, 0)
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeRuntimeColumns).
			Where("subscribe_id = ? AND status = ?", subscribeId, 1).
			Find(v).Error
	})
	return data, err
}

var userSubscribeListColumns = []string{
	"id",
	"user_id",
	"order_id",
	"subscribe_id",
	"product_name",
	"product_snapshot",
	"start_time",
	"expire_time",
	"finished_at",
	"traffic",
	"download",
	"upload",
	"traffic_reset_at",
	"reset_cycle",
	"next_traffic_reset_at",
	"token",
	"uuid",
	"status",
	"created_at",
	"updated_at",
}

var userSubscribeAdminListColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_name",
	"start_time",
	"expire_time",
	"traffic",
	"download",
	"upload",
	"status",
	"created_at",
	"updated_at",
}

var userSubscribeDetailColumns = append(append([]string{}, userSubscribeListColumns...), "note")

var userSubscribeCredentialColumns = []string{
	"id",
	"token",
}

var userSubscribeOrderActionColumns = []string{
	"id",
	"user_id",
	"order_id",
	"subscribe_id",
	"product_snapshot",
	"expire_time",
	"token",
	"status",
}

var userSubscribeRuntimeColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"expire_time",
	"traffic",
	"download",
	"upload",
	"token",
	"uuid",
	"status",
}

var userSubscribeExpiryColumns = []string{
	"id",
	"subscribe_id",
	"product_snapshot",
	"expire_time",
}

var userSubscribeDeductionColumns = []string{
	"id",
	"order_id",
	"subscribe_id",
	"start_time",
	"expire_time",
	"traffic",
	"download",
	"upload",
	"status",
}

var userSubscribePresetColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
}

var userSubscribeOrderLookupColumns = []string{
	"id",
	"user_id",
	"token",
}

var userSubscribeMutationColumns = []string{
	"id",
	"user_id",
	"order_id",
	"subscribe_id",
	"product_name",
	"product_snapshot",
	"start_time",
	"expire_time",
	"finished_at",
	"traffic",
	"download",
	"upload",
	"traffic_reset_at",
	"token",
	"status",
	"reset_cycle",
	"next_traffic_reset_at",
}

var userSubscribeCacheColumns = []string{
	"id",
	"user_id",
	"token",
}

var userSubscribeActivationUpdateColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"start_time",
	"expire_time",
	"finished_at",
	"download",
	"upload",
	"traffic_reset_at",
	"token",
	"status",
	"reset_cycle",
	"next_traffic_reset_at",
}

var userSubscribeAdminDetailColumns = []string{
	"id",
	"user_id",
	"order_id",
	"subscribe_id",
	"product_name",
	"product_snapshot",
	"start_time",
	"expire_time",
	"finished_at",
	"traffic",
	"download",
	"upload",
	"traffic_reset_at",
	"reset_cycle",
	"next_traffic_reset_at",
	"token",
	"status",
	"note",
	"created_at",
	"updated_at",
}

var userSubscribeAdminDetailUserColumns = []string{
	"id",
	"avatar",
	"balance",
	"refer_code",
	"referer_id",
	"commission",
	"referral_percentage",
	"only_first_purchase",
	"gift_amount",
	"member_expired_at",
	"enable",
	"is_admin",
	"enable_balance_notify",
	"enable_login_notify",
	"enable_subscribe_notify",
	"enable_trade_notify",
	"created_at",
	"updated_at",
	"deleted_at",
}

var userSubscribeProductListColumns = []string{
	"id",
	"name",
	"language",
	"description",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"inventory",
	"traffic",
	"speed_limit",
	"device_limit",
	"quota",
	"nodes",
	"node_tags",
	"show",
	"user_visible",
	"sell",
	"sort",
	"deduction_ratio",
	"allow_deduction",
	"allow_renewal",
	"allow_reset_traffic",
	"reset_cycle",
	"renewal_reset",
	"show_original_price",
	"created_at",
	"updated_at",
}

var userSubscribeProductSummaryColumns = []string{
	"id",
	"name",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"traffic",
	"speed_limit",
	"device_limit",
	"allow_deduction",
	"allow_renewal",
	"allow_reset_traffic",
	"reset_cycle",
	"show_original_price",
}

var userSubscribeAdminListProductColumns = []string{
	"id",
	"name",
	"speed_limit",
	"device_limit",
}

var userSubscribeActionProductColumns = []string{
	"id",
	"name",
	"description",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"traffic",
	"speed_limit",
	"device_limit",
	"quota",
	"nodes",
	"node_tags",
	"deduction_ratio",
	"allow_deduction",
	"allow_renewal",
	"allow_reset_traffic",
	"reset_cycle",
	"renewal_reset",
	"show_original_price",
}

var userSubscribeDeductionProductColumns = []string{
	"id",
	"unit_price",
	"unit_time",
	"deduction_ratio",
	"allow_deduction",
	"reset_cycle",
}

func preloadUserSubscribeProductList(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeProductListColumns)
}

func preloadUserSubscribeProductSummary(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeProductSummaryColumns)
}

func preloadUserSubscribeAdminListProduct(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeAdminListProductColumns)
}

func preloadUserSubscribeActionProduct(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeActionProductColumns)
}

func preloadUserSubscribeDeductionProduct(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeDeductionProductColumns)
}

func preloadUserSubscribeAdminDetailUser(db *gorm.DB) *gorm.DB {
	return db.Select(userSubscribeAdminDetailUserColumns)
}

func (d *SubscribeDetails) ToSubscribe() *Subscribe {
	if d == nil {
		return nil
	}
	return &Subscribe{
		Id:                 d.Id,
		UserId:             d.UserId,
		OrderId:            d.OrderId,
		SubscribeId:        d.SubscribeId,
		ProductName:        d.ProductName,
		ProductSnapshot:    d.ProductSnapshot,
		StartTime:          d.StartTime,
		ExpireTime:         d.ExpireTime,
		FinishedAt:         d.FinishedAt,
		Traffic:            d.Traffic,
		Download:           d.Download,
		Upload:             d.Upload,
		TrafficResetAt:     d.TrafficResetAt,
		ResetCycle:         d.ResetCycle,
		NextTrafficResetAt: d.NextTrafficResetAt,
		Token:              d.Token,
		UUID:               d.UUID,
		Status:             d.Status,
		Note:               d.Note,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// QueryUserSubscribe returns a list of records that meet the conditions.
func (m *defaultUserModel) QueryUserSubscribe(ctx context.Context, userId int64, status ...int64) ([]*SubscribeDetails, error) {
	return m.queryUserSubscribe(ctx, userId, true, status...)
}

// QueryUserSubscribeRuntime returns subscription runtime fields without preloading product details.
func (m *defaultUserModel) QueryUserSubscribeRuntime(ctx context.Context, userId int64, status ...int64) ([]*SubscribeDetails, error) {
	return m.queryUserSubscribe(ctx, userId, false, status...)
}

func (m *defaultUserModel) queryUserSubscribe(ctx context.Context, userId int64, preloadProduct bool, status ...int64) ([]*SubscribeDetails, error) {
	list := make([]*SubscribeDetails, 0)
	query := func(conn *gorm.DB, v interface{}) error {
		tx := applyUserSubscribeFilter(conn.Model(&Subscribe{}), userId, status...).
			Select(userSubscribeListColumns)
		if preloadProduct {
			tx = tx.Preload("Subscribe", preloadUserSubscribeProductSummary)
		}
		return tx.Find(v).Error
	}
	if !preloadProduct || len(status) > 0 {
		err := m.QueryNoCacheCtx(ctx, &list, query)
		return list, err
	}
	key := cachekeys.Int(cacheUserSubscribeUserPrefix, userId)
	err := m.QueryCtx(ctx, &list, key, query)
	return list, err
}

// QueryUserSubscribePage returns a paginated list of user subscriptions.
func (m *defaultUserModel) QueryUserSubscribePage(ctx context.Context, userId int64, page, size int, status ...int64) ([]*SubscribeDetails, int64, error) {
	list := make([]*SubscribeDetails, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		now := time.Now()
		countQuery := applyUserSubscribeFilterAt(conn.Model(&Subscribe{}), userId, now, status...)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*SubscribeDetails, 0, capacity)
		return applyUserSubscribeFilterAt(conn.Model(&Subscribe{}), userId, now, status...).
			Select(userSubscribeAdminListColumns).
			Order("id DESC").
			Limit(size).
			Offset(offset).
			Preload("Subscribe", preloadUserSubscribeAdminListProduct).
			Find(&list).Error
	})
	return list, total, err
}

func (m *defaultUserModel) CountUserSubscribe(ctx context.Context, userId int64, status ...int64) (int64, error) {
	return m.countUserSubscribe(ctx, userId, 0, status...)
}

func (m *defaultUserModel) CountUserSubscribeByProduct(ctx context.Context, userId, subscribeId int64, status ...int64) (int64, error) {
	if subscribeId <= 0 {
		return 0, nil
	}
	return m.countUserSubscribe(ctx, userId, subscribeId, status...)
}

func (m *defaultUserModel) countUserSubscribe(ctx context.Context, userId, subscribeId int64, status ...int64) (int64, error) {
	var total int64
	err := m.QueryNoCacheCtx(ctx, &total, func(conn *gorm.DB, _ interface{}) error {
		tx := applyUserSubscribeFilter(conn.Model(&Subscribe{}), userId, status...)
		if subscribeId > 0 {
			tx = tx.Where("subscribe_id = ?", subscribeId)
		}
		return tx.Count(&total).Error
	})
	return total, err
}

func applyUserSubscribeFilter(conn *gorm.DB, userId int64, status ...int64) *gorm.DB {
	return applyUserSubscribeFilterAt(conn, userId, time.Now(), status...)
}

func applyUserSubscribeFilterAt(conn *gorm.DB, userId int64, now time.Time, status ...int64) *gorm.DB {
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	conn = conn.Where("user_id = ?", userId)
	if len(status) > 0 {
		conn = conn.Where("status IN ?", status)
	}
	return conn.Where("expire_time > ? OR finished_at >= ? OR expire_time = ?", now, sevenDaysAgo, time.UnixMilli(0))
}

// FindOneUserSubscribeAccess returns only the credential fields needed to render a subscription URL.
func (m *defaultUserModel) FindOneUserSubscribeAccess(ctx context.Context, id int64) (subscribeDetails *SubscribeDetails, err error) {
	var data SubscribeDetails
	err = m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeCredentialColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

// FindOneUserSubscribe  finds a subscribeDetails by id.
func (m *defaultUserModel) FindOneUserSubscribe(ctx context.Context, id int64) (subscribeDetails *SubscribeDetails, err error) {
	//TODO cache
	//key := cachekeys.Int(cacheUserSubscribeUserPrefix, userId)
	var data SubscribeDetails
	err = m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeDetailColumns).
			Preload("Subscribe", preloadUserSubscribeActionProduct).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

// FindOneUserSubscribeForDeduction returns only the fields needed by remaining-amount calculation.
func (m *defaultUserModel) FindOneUserSubscribeForDeduction(ctx context.Context, id int64) (subscribeDetails *SubscribeDetails, err error) {
	var data SubscribeDetails
	err = m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeDeductionColumns).
			Preload("Subscribe", preloadUserSubscribeDeductionProduct).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

// FindOneUserSubscribeForOrderAction returns the user subscription fields needed to create renewal/reset orders.
func (m *defaultUserModel) FindOneUserSubscribeForOrderAction(ctx context.Context, id int64) (subscribeDetails *SubscribeDetails, err error) {
	var data SubscribeDetails
	err = m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeOrderActionColumns).
			Preload("Subscribe", preloadUserSubscribeActionProduct).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

// FindOneSubscribeByToken  finds a record by token.
func (m *defaultUserModel) FindOneSubscribeByToken(ctx context.Context, token string) (*Subscribe, error) {
	var data Subscribe
	key := cachekeys.String(cacheUserSubscribeTokenPrefix, token)
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).Where("token = ?", token).First(&data).Error
	})
	return &data, err
}

func (m *defaultUserModel) FindOneSubscribeByTokenForExpiry(ctx context.Context, token string) (*Subscribe, error) {
	var data Subscribe
	key := cachekeys.String(cacheUserSubscribeExpiryTokenPrefix, token)
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeExpiryColumns).
			Where("token = ?", token).
			First(v).Error
	})
	return &data, err
}

func (m *defaultUserModel) FindOneSubscribeByTokenForRuntime(ctx context.Context, token string) (*Subscribe, error) {
	var data Subscribe
	key := cachekeys.String(cacheUserSubscribeRuntimeTokenPrefix, token)
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeRuntimeColumns).
			Where("token = ?", token).
			First(v).Error
	})
	return &data, err
}

func (m *defaultUserModel) FindOneSubscribeByTokenForActivationUpdate(ctx context.Context, token string, tx ...*gorm.DB) (*Subscribe, error) {
	var data Subscribe
	query := func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeActivationUpdateColumns).
			Where("token = ?", token).
			First(v).Error
	}
	if len(tx) > 0 && tx[0] != nil {
		err := query(tx[0].WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}), &data)
		return &data, err
	}
	err := m.QueryNoCacheCtx(ctx, &data, query)
	return &data, err
}

func (m *defaultUserModel) findOneSubscribeByTokenForCache(ctx context.Context, token string) (*Subscribe, error) {
	var data Subscribe
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeCacheColumns).
			Where("token = ?", token).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateSubscribe updates a record.
func (m *defaultUserModel) UpdateSubscribe(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error {
	old, err := m.findOneSubscribeForCache(ctx, data.Id)
	if err != nil {
		return err
	}

	// 使用 defer 确保更新后清理缓存
	defer func() {
		if clearErr := m.ClearSubscribeCacheByModels(ctx, old, data); clearErr != nil {
			// 记录清理缓存错误
		}
	}()

	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Subscribe{}).Where("id = ?", data.Id).Save(data).Error
	})
}

func (m *defaultUserModel) UpdateSubscribeFields(ctx context.Context, current *Subscribe, update UserSubscribeFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	updated := *current
	if update.SubscribeID != nil {
		updated.SubscribeId = *update.SubscribeID
	}
	if update.ProductName != nil {
		updated.ProductName = *update.ProductName
	}
	if update.ProductSnapshot != nil {
		updated.ProductSnapshot = *update.ProductSnapshot
	}
	if update.ExpireTime != nil {
		updated.ExpireTime = *update.ExpireTime
	}
	if update.ClearFinishedAt {
		updated.FinishedAt = nil
	} else if update.FinishedAt != nil {
		finishedAt := *update.FinishedAt
		updated.FinishedAt = &finishedAt
	}
	if update.Traffic != nil {
		updated.Traffic = *update.Traffic
	}
	if update.Token != nil {
		updated.Token = *update.Token
	}
	if update.UUID != nil {
		updated.UUID = *update.UUID
	}
	if update.Download != nil {
		updated.Download = *update.Download
	}
	if update.Upload != nil {
		updated.Upload = *update.Upload
	}
	if update.TrafficResetAt != nil {
		updated.TrafficResetAt = *update.TrafficResetAt
	}
	if update.ResetCycle != nil {
		updated.ResetCycle = *update.ResetCycle
	}
	if update.ClearNextTrafficResetAt {
		updated.NextTrafficResetAt = nil
	} else if update.NextTrafficResetAt != nil {
		nextResetAt := *update.NextTrafficResetAt
		updated.NextTrafficResetAt = &nextResetAt
	}
	if update.Status != nil {
		updated.Status = *update.Status
	}
	if update.Note != nil {
		updated.Note = *update.Note
	}
	if len(tx) == 0 || tx[0] == nil {
		defer func() {
			if clearErr := m.ClearSubscribeCacheByModels(ctx, current, &updated); clearErr != nil {
				// cache cleanup errors are intentionally non-blocking here
			}
		}()
	}
	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Subscribe{}).Where("id = ?", current.Id).Updates(updates).Error
	})
}

func (m *defaultUserModel) UpdateSubscribeActivationFields(ctx context.Context, current *Subscribe, update UserSubscribeActivationUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	if len(tx) == 0 || tx[0] == nil {
		defer func() {
			if clearErr := m.ClearSubscribeCacheByModels(ctx, current); clearErr != nil {
				// cache cleanup errors are intentionally non-blocking here
			}
		}()
	}
	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Subscribe{}).Where("id = ?", current.Id).Updates(updates).Error
	})
}

// DeleteSubscribe deletes a record.
func (m *defaultUserModel) DeleteSubscribe(ctx context.Context, token string, tx ...*gorm.DB) error {
	data, err := m.findOneSubscribeByTokenForCache(ctx, token)
	if err != nil {
		return err
	}

	// 使用 defer 确保删除后清理缓存
	defer func() {
		if clearErr := m.ClearSubscribeCacheByModels(ctx, data); clearErr != nil {
			// 记录清理缓存错误
		}
	}()

	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Where("token = ?", token).Delete(&Subscribe{}).Error
	})
}

// InsertSubscribe insert Subscribe into the database.
func (m *defaultUserModel) InsertSubscribe(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error {
	if len(tx) == 0 || tx[0] == nil {
		defer func() {
			if clearErr := m.ClearSubscribeCacheByModels(ctx, data); clearErr != nil {
				// cache cleanup errors are intentionally non-blocking here
			}
		}()
	}

	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Create(data).Error
	})
}

func (m *defaultUserModel) DeleteSubscribeById(ctx context.Context, id int64, tx ...*gorm.DB) error {
	data, err := m.findOneSubscribeForCache(ctx, id)
	if err != nil {
		return err
	}

	// 使用 defer 确保删除后清理缓存
	defer func() {
		if clearErr := m.ClearSubscribeCacheByModels(ctx, data); clearErr != nil {
			// 记录清理缓存错误
		}
	}()

	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Where("id = ?", id).Delete(&Subscribe{}).Error
	})
}

func (m *defaultUserModel) ClearSubscribeCache(ctx context.Context, data ...*Subscribe) error {
	return m.ClearSubscribeCacheByModels(ctx, data...)
}
