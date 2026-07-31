package user

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/viewutil"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	cacheUserSubscribeTokenPrefix        = "cache:user:subscribe:token:"         // #nosec G101 -- Redis key namespace, not a token
	cacheUserSubscribeRuntimeTokenPrefix = "cache:user:subscribe:runtime-token:" // #nosec G101 -- Redis key namespace, not a token
	cacheUserSubscribeExpiryTokenPrefix  = "cache:user:subscribe:expiry-token:"  // #nosec G101 -- Redis key namespace, not a token
	cacheUserSubscribeUserPrefix         = "cache:user:subscribe:user:"
	cacheUserSubscribeIdPrefix           = "cache:user:subscribe:id:"
	cacheUserDeviceNumberPrefix          = "cache:user:device:number:"
	cacheUserDeviceIdPrefix              = "cache:user:device:id:"
)

type SubscribeDetails struct {
	Id                 int64                `gorm:"primarykey"`
	UserId             int64                `gorm:"index:idx_user_id;not null;comment:User ID"`
	User               *User                `gorm:"foreignKey:UserId;references:Id"`
	OrderId            int64                `gorm:"index:idx_order_id;not null;comment:Order ID"`
	SubscribeId        int64                `gorm:"index:idx_subscribe_id;not null;comment:Subscription ID"`
	Subscribe          *subscribe.Subscribe `gorm:"foreignKey:SubscribeId;references:Id"`
	ProductName        string               `gorm:"type:varchar(255);not null;default:'';comment:Product Name Snapshot"`
	ProductSnapshot    string               `gorm:"type:text;not null;default:'{}';comment:Product Snapshot"`
	StartTime          time.Time            `gorm:"default:CURRENT_TIMESTAMP;not null;comment:Subscription Start Time"`
	ExpireTime         time.Time            `gorm:"default:'1970-01-01 00:00:00+00:00';not null;comment:Subscription Expire Time"`
	FinishedAt         *time.Time           `gorm:"default:NULL;comment:Finished Time"`
	Traffic            int64                `gorm:"default:0;comment:Traffic"`
	Download           int64                `gorm:"default:0;comment:Download Traffic"`
	Upload             int64                `gorm:"default:0;comment:Upload Traffic"`
	TrafficResetAt     time.Time            `gorm:"default:'1970-01-01 00:00:00+00:00';not null;comment:Traffic Reset Barrier"`
	ResetCycle         uint8                `gorm:"type:smallint;default:0;not null;comment:Traffic Reset Cycle"`
	NextTrafficResetAt *time.Time           `gorm:"default:NULL;comment:Next Traffic Reset Time"`
	Token              string               `gorm:"index:idx_token;unique;type:varchar(255);default:'';comment:Token"`
	UUID               string               `gorm:"type:varchar(255);unique;index:idx_uuid;default:'';comment:UUID"`
	Status             uint8                `gorm:"type:smallint;default:0;comment:Subscription Status: 0: Pending 1: Active 2: Exhausted 3: Expired 4: Deducted 5: Stopped"`
	Note               string               `gorm:"type:varchar(500);default:'';comment:User note for subscription"`
	CreatedAt          time.Time            `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt          time.Time            `gorm:"comment:Update Time"`
}

type SubscribeLogFilterParams struct {
	IP              string
	UserAgent       string
	UserId          int64
	Token           string
	UserSubscribeId int64
}

type LoginLogFilterParams struct {
	IP        string
	UserId    int64
	UserAgent string
	Success   *bool
}

type UserFilterParams struct {
	Search          string
	Status          string
	UserId          *int64
	SubscribeId     *int64
	UserSubscribeId *int64
	Order           string // Order by id, e.g., "desc"
	Unscoped        bool   // Whether to include soft-deleted records
}

type customUserLogicModel interface {
	QueryPageList(ctx context.Context, page, size int, filter *UserFilterParams) ([]*User, int64, error)
	FindOneByReferCode(ctx context.Context, referCode string) (*User, error)
	BatchDeleteUser(ctx context.Context, ids []int64, tx ...*gorm.DB) error
	InsertSubscribe(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error
	FindOneSubscribeByToken(ctx context.Context, token string) (*Subscribe, error)
	FindOneSubscribeByTokenForExpiry(ctx context.Context, token string) (*Subscribe, error)
	FindOneSubscribeByTokenForRuntime(ctx context.Context, token string) (*Subscribe, error)
	FindOneSubscribeByTokenForActivationUpdate(ctx context.Context, token string, tx ...*gorm.DB) (*Subscribe, error)
	FindOneSubscribeByOrderId(ctx context.Context, orderId int64) (*Subscribe, error)
	FindOneSubscribe(ctx context.Context, id int64) (*Subscribe, error)
	FindOneSubscribeForMutation(ctx context.Context, id int64, tx ...*gorm.DB) (*Subscribe, error)
	FindOneSubscribeForPreset(ctx context.Context, id int64) (*Subscribe, error)
	UpdateSubscribe(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error
	UpdateSubscribeFields(ctx context.Context, current *Subscribe, update UserSubscribeFieldsUpdate, tx ...*gorm.DB) error
	UpdateSubscribeActivationFields(ctx context.Context, current *Subscribe, update UserSubscribeActivationUpdate, tx ...*gorm.DB) error
	DeleteSubscribe(ctx context.Context, token string, tx ...*gorm.DB) error
	DeleteSubscribeById(ctx context.Context, id int64, tx ...*gorm.DB) error
	QueryUserSubscribe(ctx context.Context, userId int64, status ...int64) ([]*SubscribeDetails, error)
	QueryUserSubscribeRuntime(ctx context.Context, userId int64, status ...int64) ([]*SubscribeDetails, error)
	QueryUserSubscribePage(ctx context.Context, userId int64, page, size int, status ...int64) ([]*SubscribeDetails, int64, error)
	CountUserSubscribe(ctx context.Context, userId int64, status ...int64) (int64, error)
	CountUserSubscribeByProduct(ctx context.Context, userId, subscribeId int64, status ...int64) (int64, error)
	FindOneUserSubscribeAccess(ctx context.Context, id int64) (*SubscribeDetails, error)
	FindOneSubscribeDetailsById(ctx context.Context, id int64) (*SubscribeDetails, error)
	FindOneUserSubscribe(ctx context.Context, id int64) (*SubscribeDetails, error)
	FindOneUserSubscribeForDeduction(ctx context.Context, id int64) (*SubscribeDetails, error)
	FindOneUserSubscribeForOrderAction(ctx context.Context, id int64) (*SubscribeDetails, error)
	FindUsersSubscribeBySubscribeId(ctx context.Context, subscribeId int64) ([]*Subscribe, error)
	UpdateUserSubscribeWithTraffic(ctx context.Context, id, download, upload int64, tx ...*gorm.DB) error
	QueryResisterUserTotalByDate(ctx context.Context, date time.Time) (int64, error)
	QueryResisterUserTotalByMonthly(ctx context.Context, date time.Time) (int64, error)
	QueryResisterUserTotal(ctx context.Context) (int64, error)
	QueryAdminUsers(ctx context.Context) ([]*User, error)
	UpdateUserCache(ctx context.Context, data *User) error
	UpdateUserSubscribeCache(ctx context.Context, data *Subscribe) error
	QueryActiveSubscriptions(ctx context.Context, subscribeId ...int64) (map[int64]int64, error)
	FindUserAuthMethods(ctx context.Context, userId int64) ([]*AuthMethods, error)
	FindUserAuthMethodSummaries(ctx context.Context, userId int64) ([]*AuthMethods, error)
	RevealUserAuthMethods(ctx context.Context, userId int64) ([]*AuthMethods, error)
	InsertUserAuthMethods(ctx context.Context, data *AuthMethods, tx ...*gorm.DB) error
	UpdateUserAuthMethods(ctx context.Context, userId int64, authType string, update UserAuthMethodFieldsUpdate, tx ...*gorm.DB) error
	DeleteUserAuthMethods(ctx context.Context, userId int64, platform string, tx ...*gorm.DB) error
	FindUserAuthMethodByOpenID(ctx context.Context, method, openID string) (*AuthMethods, error)
	FindUserAuthMethodByUserId(ctx context.Context, method string, userId int64) (*AuthMethods, error)
	FindUserAuthMethodByPlatform(ctx context.Context, userId int64, platform string) (*AuthMethods, error)
	FindUserAuthMethodsByType(ctx context.Context, authType string, userIds []int64, reveal bool) ([]*AuthMethods, error)
	FindOneByEmail(ctx context.Context, email string) (*User, error)
	FindOneForLogin(ctx context.Context, id int64) (*User, error)
	FindOneForLoginByEmail(ctx context.Context, email string) (*User, error)
	FindOneForSubscribeAccess(ctx context.Context, id int64) (*User, error)
	FindOneForAdminCheck(ctx context.Context, id int64) (*User, error)
	FindOneForOrderActivation(ctx context.Context, id int64) (*User, error)
	FindOneForReferralPolicy(ctx context.Context, id int64) (*User, error)
	FindOneForBasicUpdate(ctx context.Context, id int64) (*User, error)
	FindOneDevice(ctx context.Context, id int64) (*Device, error)
	FindOneDeviceForStatusUpdate(ctx context.Context, id int64) (*Device, error)
	FindOneDeviceByIdentifierForStatusUpdate(ctx context.Context, id string) (*Device, error)
	QueryDeviceList(ctx context.Context, userid int64, page, size int) ([]*Device, int64, error)
	QueryDevicePageList(ctx context.Context, userid, subscribeId int64, page, size int) ([]*Device, int64, error)
	UpdateDevice(ctx context.Context, data *Device, tx ...*gorm.DB) error
	UpdateDeviceFields(ctx context.Context, current *Device, update DeviceFieldsUpdate, tx ...*gorm.DB) error
	FindOneDeviceByIdentifier(ctx context.Context, id string) (*Device, error)
	DeleteDevice(ctx context.Context, id int64, tx ...*gorm.DB) error
	DeleteDeviceByModel(ctx context.Context, data *Device, tx ...*gorm.DB) error
	InsertDevice(ctx context.Context, data *Device, tx ...*gorm.DB) error

	ClearSubscribeCache(ctx context.Context, data ...*Subscribe) error
	ClearUserCache(ctx context.Context, data ...*User) error

	QueryDailyUserStatisticsList(ctx context.Context, date time.Time) ([]UserStatisticsWithDate, error)
	QueryMonthlyUserStatisticsList(ctx context.Context, date time.Time) ([]UserStatisticsWithDate, error)
	UpdateUserPassword(ctx context.Context, userId int64, password string) error
	UpdateUserBalance(ctx context.Context, userId int64, balance int64, tx ...*gorm.DB) error
	UpdateUserGiftAmount(ctx context.Context, userId int64, giftAmount int64, tx ...*gorm.DB) error
	UpdateUserCommission(ctx context.Context, userId int64, commission int64, tx ...*gorm.DB) error
	UpdateUserReferCode(ctx context.Context, userId int64, referCode string, tx ...*gorm.DB) error
	UpdateUserMemberExpiredAt(ctx context.Context, userId int64, expiredAt time.Time, tx ...*gorm.DB) error
	UpdateUserRules(ctx context.Context, userId int64, rules string) error
	UpdateUserNotifySettings(ctx context.Context, userId int64, update UserNotifySettingsUpdate) error
	UpdateUserBasicFields(ctx context.Context, userId int64, update UserBasicFieldsUpdate, tx ...*gorm.DB) error
	MigratePrivacy(ctx context.Context, batchSize int) (int64, error)
	MigrateDevicePrivacy(ctx context.Context, batchSize int) (int64, error)
}

type UserStatisticsWithDate struct {
	Date              string
	Register          int64
	NewOrderUsers     int64
	RenewalOrderUsers int64
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client, codecs ...*privacycodec.Codec) Model {
	codec := privacycodec.DisabledCodec()
	if len(codecs) > 0 && codecs[0] != nil {
		codec = codecs[0]
	}
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, codec),
	}
}

// QueryPageList returns a list of records that meet the conditions.
func (m *customUserModel) QueryPageList(ctx context.Context, page, size int, filter *UserFilterParams) ([]*User, int64, error) {
	page, size = pagination.Normalize(page, size)
	list := []*User{}
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		var err error
		total, err = m.countUserPageIDs(conn, filter)
		if err != nil {
			return err
		}
		if total == 0 {
			return nil
		}
		ids, err := m.queryUserPageIDs(conn, filter, page, size, total)
		if err != nil {
			return err
		}
		list, err = m.queryUsersByIDs(conn, filter, ids)
		return err
	})
	return list, total, err
}

func (m *customUserModel) countUserPageIDs(conn *gorm.DB, filter *UserFilterParams) (int64, error) {
	var total int64
	idQuery := m.applyUserFilter(conn.Model(&User{}), filter).Select(`DISTINCT "user".id`)
	if err := conn.Table("(?) AS filtered_users", idQuery).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

type userIDRow struct {
	Id int64 `gorm:"column:id"`
}

func (m *customUserModel) queryUserPageIDs(conn *gorm.DB, filter *UserFilterParams, page, size int, total int64) ([]int64, error) {
	offset, capacity, ok := pagination.PageWindow(total, page, size)
	if !ok {
		return []int64{}, nil
	}
	rows := make([]userIDRow, 0, capacity)
	query := applyUserOrder(m.applyUserFilter(conn.Model(&User{}), filter), filter)
	if err := query.
		Select(`DISTINCT "user".id AS id`).
		Limit(size).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	return ids, nil
}

func (m *customUserModel) queryUsersByIDs(conn *gorm.DB, filter *UserFilterParams, ids []int64) ([]*User, error) {
	if len(ids) == 0 {
		return []*User{}, nil
	}
	users := make([]*User, 0, len(ids))
	query := conn.Model(&User{}).
		Select(userListSelectColumns).
		Where(`"user".id IN ?`, ids)
	if shouldUnscopeUserQuery(filter) {
		query = query.Unscoped()
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	authMethods, err := m.queryPrimaryAuthMethodsByUserIDs(conn, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range users {
		if item == nil {
			continue
		}
		if authMethod, ok := authMethods[item.Id]; ok {
			item.AuthMethods = []AuthMethods{authMethod}
		}
	}
	return orderUsersByIDs(users, ids), nil
}

func (m *customUserModel) queryPrimaryAuthMethodsByUserIDs(conn *gorm.DB, userIDs []int64) (map[int64]AuthMethods, error) {
	if len(userIDs) == 0 {
		return map[int64]AuthMethods{}, nil
	}
	ranked := conn.Model(&AuthMethods{}).
		Select(m.authMethodDisplayColumns(userPrimaryAuthMethodListColumns)).
		Where("user_id IN ?", userIDs).
		Where("auth_type NOT IN ?", removedUserSignInMethods)
	rows := make([]AuthMethods, 0, len(userIDs))
	if err := conn.Table("(?) AS ranked_auth_methods", ranked).
		Where("rn = 1").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	methods := make(map[int64]AuthMethods, len(rows))
	for _, row := range rows {
		methods[row.UserId] = row
	}
	return methods, nil
}

func orderUsersByIDs(users []*User, ids []int64) []*User {
	ordered := viewutil.NewRowsCollector[*User](len(ids))
	if len(users) <= smallUserOrderLookupLimit {
		for _, id := range ids {
			for _, user := range users {
				if user != nil && user.Id == id {
					ordered.Add(user)
					break
				}
			}
		}
		return ordered.Rows()
	}
	byID := make(map[int64]*User, len(users))
	for _, user := range users {
		if user != nil {
			byID[user.Id] = user
		}
	}
	for _, id := range ids {
		if user, ok := byID[id]; ok {
			ordered.Add(user)
		}
	}
	return ordered.Rows()
}

const smallUserOrderLookupLimit = 8

var userListColumns = []string{
	"id",
	"balance",
	"refer_code",
	"referer_id",
	"commission",
	"gift_amount",
	"member_expired_at",
	"enable",
	"created_at",
	"deleted_at",
}

var userListSelectColumns = qualifyTableColumns((&User{}).TableName(), userListColumns)

var userContextColumns = []string{
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
	"rules",
	"created_at",
	"updated_at",
	"deleted_at",
}

var userContextSelectColumns = qualifyTableColumns((&User{}).TableName(), userContextColumns)

var userBasicUpdateColumns = []string{
	"id",
	"balance",
	"commission",
	"gift_amount",
	"enable",
	"is_admin",
}

var userBasicUpdateSelectColumns = qualifyTableColumns((&User{}).TableName(), userBasicUpdateColumns)

var userLoginColumns = []string{
	"id",
	"password",
	"algo",
	"salt",
	"enable",
	"deleted_at",
}

var userLoginSelectColumns = qualifyTableColumns((&User{}).TableName(), userLoginColumns)

var userSubscribeAccessColumns = []string{
	"id",
	"enable",
	"deleted_at",
}

var userSubscribeAccessSelectColumns = qualifyTableColumns((&User{}).TableName(), userSubscribeAccessColumns)

var userAdminCheckColumns = []string{
	"id",
	"enable",
	"is_admin",
	"deleted_at",
}

var userAdminCheckSelectColumns = qualifyTableColumns((&User{}).TableName(), userAdminCheckColumns)

var userOrderActivationColumns = []string{
	"id",
	"referer_id",
}

var userOrderActivationSelectColumns = qualifyTableColumns((&User{}).TableName(), userOrderActivationColumns)

var userReferralPolicyColumns = []string{
	"id",
	"referral_percentage",
	"only_first_purchase",
}

var userReferralPolicySelectColumns = qualifyTableColumns((&User{}).TableName(), userReferralPolicyColumns)

var userEmailLookupColumns = []string{
	"id",
	"deleted_at",
}

var userEmailLookupSelectColumns = qualifyTableColumns((&User{}).TableName(), userEmailLookupColumns)

var userReferCodeLookupColumns = []string{
	"id",
}

var userReferCodeLookupSelectColumns = qualifyTableColumns((&User{}).TableName(), userReferCodeLookupColumns)

var userCacheColumns = []string{
	"id",
}

var userCacheSelectColumns = qualifyTableColumns((&User{}).TableName(), userCacheColumns)

var userAuthMethodListColumns = []string{
	"id",
	"user_id",
	"auth_type",
	"auth_identifier",
	"verified",
}

var removedUserSignInMethods = []string{"mobile", "device"}

var userPrimaryAuthMethodListColumns = []string{
	"user_id",
	"auth_type",
	"auth_identifier",
	"verified",
	"ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY CASE WHEN auth_type = 'email' THEN 0 ELSE 1 END, id ASC) AS rn",
}

var userAuthMethodCacheColumns = []string{
	"user_id",
	"auth_type",
	"auth_identifier",
}

var userAuthMethodLookupColumns = []string{
	"id",
	"user_id",
	"auth_type",
	"auth_identifier",
	"verified",
}

var userAuthMethodContextColumns = []string{
	"user_id",
	"auth_type",
	"auth_identifier",
	"verified",
}

var userAuthMethodSummaryColumns = []string{
	"auth_type",
	"auth_identifier",
	"verified",
}

var userAdminNotificationColumns = []string{
	"id",
}

var userAdminNotificationAuthColumns = []string{
	"user_id",
	"auth_type",
	"auth_identifier",
}

func qualifyTableColumns(table string, columns []string) []string {
	qualified := make([]string, len(columns))
	for i, column := range columns {
		qualified[i] = `"` + table + `".` + column
	}
	return qualified
}

func (m *customUserModel) applyUserFilter(conn *gorm.DB, filter *UserFilterParams) *gorm.DB {
	if filter == nil {
		return conn
	}
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Unscoped || status == "deleted" {
		conn = conn.Unscoped()
	}
	switch status {
	case "enabled":
		conn = conn.Where(`"user".enable = ?`, true).Where(`"user".deleted_at IS NULL`)
	case "disabled":
		conn = conn.Where(`"user".enable = ?`, false).Where(`"user".deleted_at IS NULL`)
	case "deleted":
		conn = conn.Where(`"user".deleted_at IS NOT NULL`)
	}
	if filter.UserId != nil {
		conn = conn.Where(`"user".id = ?`, *filter.UserId)
	}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		conn = conn.Joins(
			`LEFT JOIN user_auth_methods ON "user".id = user_auth_methods.user_id AND user_auth_methods.auth_type NOT IN ?`,
			removedUserSignInMethods,
		)
		likeSearch := "%" + search + "%"
		userID, userIDError := strconv.ParseInt(search, 10, 64)
		matchesUserID := userIDError == nil && userID > 0
		if m.privacyCodec.Enabled() && looksLikeIdentity(search) {
			if matchesUserID {
				conn = conn.Where(`(user_auth_methods.auth_identifier_bidx IN ? OR "user".refer_code LIKE ? OR "user".id = ?)`, m.identitySearchIndexes(search), likeSearch, userID)
			} else {
				conn = conn.Where(`(user_auth_methods.auth_identifier_bidx IN ? OR "user".refer_code LIKE ?)`, m.identitySearchIndexes(search), likeSearch)
			}
		} else if m.privacyCodec.Enabled() {
			if matchesUserID {
				conn = conn.Where(`("user".refer_code LIKE ? OR "user".id = ?)`, likeSearch, userID)
			} else {
				conn = conn.Where(`"user".refer_code LIKE ?`, likeSearch)
			}
		} else {
			if matchesUserID {
				conn = conn.Where(`(user_auth_methods.auth_identifier LIKE ? OR "user".refer_code LIKE ? OR "user".id = ?)`, likeSearch, likeSearch, userID)
			} else {
				conn = conn.Where(`(user_auth_methods.auth_identifier LIKE ? OR "user".refer_code LIKE ?)`, likeSearch, likeSearch)
			}
		}
	}
	if filter.UserSubscribeId != nil || filter.SubscribeId != nil {
		conn = conn.Joins(`LEFT JOIN user_subscribe ON "user".id = user_subscribe.user_id`).
			Where("user_subscribe.status = ?", 1)
		if filter.UserSubscribeId != nil {
			conn = conn.Where("user_subscribe.id = ?", *filter.UserSubscribeId)
		}
		if filter.SubscribeId != nil {
			conn = conn.Where("user_subscribe.subscribe_id = ?", *filter.SubscribeId)
		}
	}
	return conn
}

func shouldUnscopeUserQuery(filter *UserFilterParams) bool {
	if filter == nil {
		return false
	}
	return filter.Unscoped || strings.EqualFold(strings.TrimSpace(filter.Status), "deleted")
}

func applyUserOrder(conn *gorm.DB, filter *UserFilterParams) *gorm.DB {
	if filter == nil || filter.Order == "" {
		return conn
	}
	switch strings.ToUpper(filter.Order) {
	case "ASC":
		return conn.Order(`"user".id ASC`)
	case "DESC":
		return conn.Order(`"user".id DESC`)
	default:
		return conn
	}
}

// BatchDeleteUser deletes multiple records by primary key.
func (m *customUserModel) BatchDeleteUser(ctx context.Context, ids []int64, tx ...*gorm.DB) error {
	ids = normalizeUserCacheIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	users, err := m.queryUsersForCache(ctx, ids, false, tx...)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Where("id in ?", ids).Delete(&User{}).Error
	}, m.batchGetCacheKeys(users...)...)
}

func (m *customUserModel) UpdateUserSubscribeWithTraffic(ctx context.Context, id, download, upload int64, tx ...*gorm.DB) error {
	sub, err := m.findOneSubscribeForCache(ctx, id)
	if err != nil {
		return err
	}

	// 使用 defer 确保更新后清理缓存
	defer func() {
		if clearErr := m.ClearSubscribeCacheByModels(ctx, sub); clearErr != nil {
			// 记录清理缓存错误
		}
	}()

	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Subscribe{}).Where("id = ?", id).Updates(SubscribeTrafficDelta{
			Download: download,
			Upload:   upload,
		}.Updates()).Error
	})
}

func (m *customUserModel) QueryResisterUserTotalByDate(ctx context.Context, date time.Time) (int64, error) {
	var total int64
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour).Add(-time.Second)
	err := m.QueryNoCacheCtx(ctx, &total, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).Where("created_at > ? and created_at < ?", start, end).Count(&total).Error
	})
	return total, err
}

func (m *customUserModel) QueryResisterUserTotalByMonthly(ctx context.Context, date time.Time) (int64, error) {
	var total int64
	start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	err := m.QueryNoCacheCtx(ctx, &total, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).Where("created_at > ? and created_at < ?", start, end).Count(&total).Error
	})
	return total, err
}

func (m *customUserModel) QueryResisterUserTotal(ctx context.Context) (int64, error) {
	var total int64
	err := m.QueryNoCacheCtx(ctx, &total, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).Count(&total).Error
	})
	return total, err
}

func (m *customUserModel) QueryAdminUsers(ctx context.Context) ([]*User, error) {
	data := make([]*User, 0)
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Select(userAdminNotificationColumns).
			Preload("AuthMethods", func(db *gorm.DB) *gorm.DB {
				return db.Select(userAdminNotificationAuthColumns).
					Where("auth_type = ?", "telegram")
			}).
			Where("is_admin = ?", true).
			Find(&data).Error
	})
	if err == nil {
		for _, item := range data {
			for index := range item.AuthMethods {
				if err = m.openAuthMethod(&item.AuthMethods[index]); err != nil {
					return nil, err
				}
			}
		}
	}
	return data, err
}

func (m *customUserModel) UpdateUserCache(ctx context.Context, data *User) error {
	return m.ClearUserCache(ctx, data)
}

func (m *customUserModel) FindOneByReferCode(ctx context.Context, referCode string) (*User, error) {
	var data User
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Select(userReferCodeLookupSelectColumns).
			Where("refer_code = ?", referCode).
			First(&data).Error
	})
	return &data, err
}

func (m *customUserModel) FindOneSubscribeDetailsById(ctx context.Context, id int64) (*SubscribeDetails, error) {
	var data SubscribeDetails
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(userSubscribeAdminDetailColumns).
			Preload("Subscribe", preloadUserSubscribeProductList).
			Preload("User", preloadUserSubscribeAdminDetailUser).
			Where("id = ?", id).
			First(&data).Error
	})
	return &data, err
}

// QueryDailyUserStatisticsList Query daily user statistics list for the current month (from 1st to current date)
func (m *customUserModel) QueryDailyUserStatisticsList(ctx context.Context, date time.Time) ([]UserStatisticsWithDate, error) {
	var results []UserStatisticsWithDate

	err := m.QueryNoCacheCtx(ctx, &results, func(conn *gorm.DB, v interface{}) error {
		firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())

		// 子查询：统计每天的新用户订单数量
		newOrderSub := conn.Model(&order.Order{}).
			Select("TO_CHAR(created_at, 'YYYY-MM-DD') AS date, COUNT(DISTINCT user_id) AS new_order_users").
			Where("is_new = ? AND created_at BETWEEN ? AND ? AND status IN ?", true, firstDay, date, []int64{2, 5}).
			Group("TO_CHAR(created_at, 'YYYY-MM-DD')")

		// 子查询：统计每天的续费订单数量
		renewalOrderSub := conn.Model(&order.Order{}).
			Select("TO_CHAR(created_at, 'YYYY-MM-DD') AS date, COUNT(DISTINCT user_id) AS renewal_order_users").
			Where("is_new = ? AND created_at BETWEEN ? AND ? AND status IN ?", false, firstDay, date, []int64{2, 5}).
			Group("TO_CHAR(created_at, 'YYYY-MM-DD')")

		return conn.Model(&User{}).
			Select(`
                TO_CHAR("user".created_at, 'YYYY-MM-DD') AS date,
                COUNT(*) AS register,
                COALESCE(MAX(n.new_order_users), 0) AS new_order_users,
                COALESCE(MAX(r.renewal_order_users), 0) AS renewal_order_users
            `).
			Joins(`LEFT JOIN (?) AS n ON TO_CHAR("user".created_at, 'YYYY-MM-DD') = n.date`, newOrderSub).
			Joins(`LEFT JOIN (?) AS r ON TO_CHAR("user".created_at, 'YYYY-MM-DD') = r.date`, renewalOrderSub).
			Where(`"user".created_at BETWEEN ? AND ?`, firstDay, date).
			Group(`TO_CHAR("user".created_at, 'YYYY-MM-DD')`).
			Order("date ASC").
			Scan(v).Error
	})

	return results, err
}

// QueryMonthlyUserStatisticsList Query monthly user statistics list for the past 6 months
func (m *customUserModel) QueryMonthlyUserStatisticsList(ctx context.Context, date time.Time) ([]UserStatisticsWithDate, error) {
	var results []UserStatisticsWithDate

	err := m.QueryNoCacheCtx(ctx, &results, func(conn *gorm.DB, v interface{}) error {
		// 获取 6 个月前的日期
		sixMonthsAgo := date.AddDate(0, -5, 0)

		// 子查询：每月新订单用户数量
		newOrderSub := conn.Model(&order.Order{}).
			Select("TO_CHAR(created_at, 'YYYY-MM') AS date, COUNT(DISTINCT user_id) AS new_order_users").
			Where("is_new = ? AND created_at >= ? AND status IN ?", true, sixMonthsAgo, []int64{2, 5}).
			Group("TO_CHAR(created_at, 'YYYY-MM')")

		// 子查询：每月续费订单用户数量
		renewalOrderSub := conn.Model(&order.Order{}).
			Select("TO_CHAR(created_at, 'YYYY-MM') AS date, COUNT(DISTINCT user_id) AS renewal_order_users").
			Where("is_new = ? AND created_at >= ? AND status IN ?", false, sixMonthsAgo, []int64{2, 5}).
			Group("TO_CHAR(created_at, 'YYYY-MM')")

		return conn.Model(&User{}).
			Select(`
				TO_CHAR("user".created_at, 'YYYY-MM') AS date,
				COUNT(*) AS register,
				COALESCE(MAX(n.new_order_users), 0) AS new_order_users,
				COALESCE(MAX(r.renewal_order_users), 0) AS renewal_order_users
			`).
			Joins(`LEFT JOIN (?) AS n ON TO_CHAR("user".created_at, 'YYYY-MM') = n.date`, newOrderSub).
			Joins(`LEFT JOIN (?) AS r ON TO_CHAR("user".created_at, 'YYYY-MM') = r.date`, renewalOrderSub).
			Where(`"user".created_at >= ?`, sixMonthsAgo).
			Group(`TO_CHAR("user".created_at, 'YYYY-MM')`).
			Order("date ASC").
			Scan(v).Error
	})

	return results, err
}
