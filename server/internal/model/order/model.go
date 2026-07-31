package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/model/payment"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Details struct {
	Id                  int64                `gorm:"primaryKey"`
	ParentId            int64                `gorm:"type:bigint;default:null;comment:Parent Order Id"`
	SubOrders           []*Order             `gorm:"foreignKey:ParentId;references:Id"`
	UserId              int64                `gorm:"type:bigint;not null;default:0;comment:User Id"`
	OrderNo             string               `gorm:"type:varchar(255);not null;default:'';unique;comment:Order No"`
	Source              string               `gorm:"type:varchar(32);not null;default:'customer';comment:Order Source"`
	Type                uint8                `gorm:"type:smallint;not null;default:1;comment:Order Type: 1: Subscribe, 2: Renewal, 3: ResetTraffic, 4: Recharge, 5: Membership"`
	Quantity            int64                `gorm:"type:bigint;not null;default:1;comment:Quantity"`
	Price               int64                `gorm:"type:int;not null;default:0;comment:Original price"`
	Amount              int64                `gorm:"type:int;not null;default:0;comment:Order Amount"`
	Discount            int64                `gorm:"type:int;not null;default:0;comment:Order Discount"`
	Coupon              string               `gorm:"type:varchar(255);default:null;comment:Coupon"`
	CouponDiscount      int64                `gorm:"type:int;not null;default:0;comment:Coupon Discount"`
	PaymentId           int64                `gorm:"type:bigint;not null;default:0;comment:Payment Id"`
	Payment             *payment.Payment     `gorm:"foreignKey:PaymentId;references:Id"`
	Method              string               `gorm:"type:varchar(255);not null;default:'';comment:Payment Method"`
	FeeAmount           int64                `gorm:"type:int;not null;default:0;comment:Fee Amount"`
	TradeNo             string               `gorm:"type:varchar(255);default:null;comment:Trade No"`
	GiftAmount          int64                `gorm:"type:int;not null;default:0;comment:User Gift Amount"`
	CommissionDeduction int64                `gorm:"type:bigint;not null;default:0;comment:Commission balance applied to order"`
	Commission          int64                `gorm:"type:int;not null;default:0;comment:Order Commission"`
	Status              uint8                `gorm:"type:smallint;not null;default:1;comment:Order Status: 1: Pending, 2: Paid, 3: Failed"`
	SubscribeId         int64                `gorm:"type:bigint;not null;default:0;comment:Subscribe Id"`
	SubscribeToken      string               `gorm:"type:varchar(255);default:null;comment:Renewal Subscribe Token"`
	ProductName         string               `gorm:"type:varchar(255);not null;default:'';comment:Product Name Snapshot"`
	ProductSnapshot     string               `gorm:"type:text;not null;default:'{}';comment:Product Snapshot"`
	ClientIP            string               `gorm:"type:varchar(64);not null;default:'';comment:Client IP"`
	CountryShort        string               `gorm:"type:varchar(8);not null;default:'';comment:Client IP Country Code"`
	Region              string               `gorm:"type:varchar(128);not null;default:'';comment:Client IP Region"`
	Location            string               `gorm:"type:varchar(160);not null;default:'';comment:Client IP Location"`
	Subscribe           *subscribe.Subscribe `gorm:"foreignKey:SubscribeId;references:Id"`
	IsNew               bool                 `gorm:"type:boolean;not null;default:false;comment:Is New Order"`
	CreatedAt           time.Time            `gorm:"<-:create;comment:Create Time"`
	UpdatedAt           time.Time            `gorm:"comment:Update Time"`
}

type OrdersTotalWithDate struct {
	Date               string
	AmountTotal        int64
	NewOrderAmount     int64
	RenewalOrderAmount int64
}

const (
	orderTypeSubscribe    uint8 = 1
	orderTypeRenewal      uint8 = 2
	orderTypeResetTraffic uint8 = 3

	smallOrderListSubscribeIndexThreshold = 8
)

type customOrderLogicModel interface {
	UpdateOrderStatus(ctx context.Context, orderNo string, status uint8, tx ...*gorm.DB) error
	FindOneForStatusUpdate(ctx context.Context, id int64) (*Order, error)
	FindOneForCheckout(ctx context.Context, orderNo string) (*Order, error)
	FindOneForActivation(ctx context.Context, orderNo string) (*Order, error)
	FindOneForPurchaseQuery(ctx context.Context, orderNo string) (*Order, error)
	FindOneForPaymentNotify(ctx context.Context, orderNo string) (*Order, error)
	FindOneForClose(ctx context.Context, orderNo string) (*Order, error)
	FindOneForCloseByUser(ctx context.Context, orderNo string, userID int64) (*Order, error)
	FindOneForUnsubscribeRefund(ctx context.Context, id int64) (*Order, error)
	UpdateFields(ctx context.Context, current *Order, update OrderFieldsUpdate, tx ...*gorm.DB) error
	SetSettlementIfCompatible(ctx context.Context, current *Order, currency string, amount int64) (bool, error)
	UpdateAdminStatusFields(ctx context.Context, current *Order, update OrderFieldsUpdate, tx ...*gorm.DB) error
	UpdateStatusIfCurrent(ctx context.Context, current *Order, fromStatus, toStatus uint8, tx ...*gorm.DB) (bool, error)
	MarkPaid(ctx context.Context, orderNo, tradeNo string) (bool, error)
	QueryAdminOrderListByPage(ctx context.Context, page, size int, status uint8, user, subscribe int64, search string) (int64, []*Details, error)
	QueryOrderListByPage(ctx context.Context, page, size int, status uint8, user, subscribe int64, search string) (int64, []*Details, error)
	FindOneAdminDetails(ctx context.Context, id int64) (*Details, error)
	FindOneDetails(ctx context.Context, id int64) (*Details, error)
	FindOneDetailsByOrderNo(ctx context.Context, orderNo string) (*Details, error)
	FindOneDetailsByOrderNoForUser(ctx context.Context, orderNo string, userID int64) (*Details, error)
	FindOneByRequestID(ctx context.Context, userID int64, orderType uint8, requestID string) (*Order, error)
	QueryMonthlyOrders(ctx context.Context, date time.Time) (OrdersTotal, error)
	QueryDateOrders(ctx context.Context, date time.Time) (OrdersTotal, error)
	QueryTotalOrders(ctx context.Context) (OrdersTotal, error)
	QueryMonthlyUserCounts(ctx context.Context, date time.Time) (int64, int64, error)
	QueryDateUserCounts(ctx context.Context, date time.Time) (int64, int64, error)
	QueryTotalUserCounts(ctx context.Context) (int64, int64, error)
	IsUserEligibleForNewOrder(ctx context.Context, userID int64) (bool, error)
	QueryDailyOrdersList(ctx context.Context, date time.Time) ([]OrdersTotalWithDate, error)
	QueryMonthlyOrdersList(ctx context.Context, date time.Time) ([]OrdersTotalWithDate, error)
}

var orderCacheColumns = []string{
	"id",
	"order_no",
}

type OrderFieldsUpdate struct {
	Status             *uint8
	PaymentID          *int64
	Method             *string
	TradeNo            *string
	SettlementAmount   *int64
	SettlementCurrency *string
}

func UpdateValue[T any](value T) *T {
	return &value
}

func (u OrderFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 4)
	if u.Status != nil {
		updates["status"] = *u.Status
	}
	if u.PaymentID != nil {
		updates["payment_id"] = *u.PaymentID
	}
	if u.Method != nil {
		updates["method"] = *u.Method
	}
	if u.TradeNo != nil {
		updates["trade_no"] = *u.TradeNo
	}
	if u.SettlementAmount != nil {
		updates["settlement_amount"] = *u.SettlementAmount
	}
	if u.SettlementCurrency != nil {
		updates["settlement_currency"] = *u.SettlementCurrency
	}
	return updates
}

// UserCounts  User counts for new and renewal users
type UserCounts struct {
	NewUsers     int64 `gorm:"column:new_users"`
	RenewalUsers int64 `gorm:"column:renewal_users"`
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customOrderModel{
		defaultOrderModel: newOrderModel(conn, c),
	}
}

func (m *customOrderModel) FindOneByRequestID(ctx context.Context, userID int64, orderType uint8, requestID string) (*Order, error) {
	var data Order
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select("id", "order_no", "user_id", "type", "request_id", "status").
			Where("user_id = ? AND type = ? AND request_id = ?", userID, orderType, requestID).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// QueryAdminOrderListByPage returns the lightweight admin order list shape.
func (m *customOrderModel) QueryAdminOrderListByPage(ctx context.Context, page, size int, status uint8, userID, subscribeID int64, search string) (int64, []*Details, error) {
	return m.queryOrderListByPage(ctx, page, size, status, userID, subscribeID, search, orderListQueryOptions{
		columns:        adminOrderListColumns,
		preloadPayment: preloadAdminOrderListPayment,
	})
}

// QueryOrderListByPage Query order list by page
func (m *customOrderModel) QueryOrderListByPage(ctx context.Context, page, size int, status uint8, userID, subscribeID int64, search string) (int64, []*Details, error) {
	return m.queryOrderListByPage(ctx, page, size, status, userID, subscribeID, search, orderListQueryOptions{
		columns: publicOrderListColumns,
		hydrate: hydratePublicOrderListSubscribes,
	})
}

type orderListQueryOptions struct {
	columns        []string
	preloadPayment func(*gorm.DB) *gorm.DB
	hydrate        func(context.Context, *gorm.DB, []*Details) error
}

func (m *customOrderModel) queryOrderListByPage(ctx context.Context, page, size int, status uint8, userID, subscribeID int64, search string, opts orderListQueryOptions) (int64, []*Details, error) {
	page, size = pagination.Normalize(page, size)
	list := []*Details{}
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		countQuery := applyOrderListFilter(conn.Model(&Order{}), status, userID, subscribeID, search)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Details, 0, capacity)
		preloadPayment := opts.preloadPayment
		if preloadPayment == nil {
			preloadPayment = preloadOrderListPayment
		}
		if err := applyOrderListFilter(conn.Model(&Order{}), status, userID, subscribeID, search).
			Select(opts.columns).
			Order("id desc").
			Preload("Payment", preloadPayment).
			Offset(offset).
			Limit(size).
			Find(&list).Error; err != nil {
			return err
		}
		if opts.hydrate == nil {
			return nil
		}
		return opts.hydrate(ctx, conn, list)
	})
	return total, list, err
}

var adminOrderListColumns = []string{
	"id",
	"user_id",
	"order_no",
	"source",
	"type",
	"quantity",
	"price",
	"amount",
	"gift_amount",
	"commission_deduction",
	"discount",
	"coupon_discount",
	"payment_id",
	"fee_amount",
	"trade_no",
	"status",
	"subscribe_id",
	"product_name",
	"updated_at",
}

var publicOrderListColumns = []string{
	"id",
	"user_id",
	"order_no",
	"type",
	"quantity",
	"price",
	"amount",
	"gift_amount",
	"commission_deduction",
	"discount",
	"coupon",
	"coupon_discount",
	"payment_id",
	"method",
	"fee_amount",
	"trade_no",
	"status",
	"subscribe_id",
	"product_name",
	"created_at",
	"updated_at",
}

var publicOrderDetailColumns = append(append([]string{}, publicOrderListColumns...), "product_snapshot")

var adminOrderDetailColumns = append(append([]string{}, publicOrderDetailColumns...), "source", "commission")

var internalOrderDetailColumns = []string{
	"id",
	"order_no",
	"quantity",
	"amount",
	"gift_amount",
	"commission_deduction",
}

var internalOrderSubOrderColumns = []string{
	"id",
	"parent_id",
	"quantity",
	"amount",
	"gift_amount",
	"commission_deduction",
	"status",
}

var orderStatusUpdateColumns = []string{
	"id",
	"order_no",
	"status",
}

var orderCheckoutColumns = []string{
	"id",
	"user_id",
	"order_no",
	"type",
	"quantity",
	"amount",
	"settlement_amount",
	"settlement_currency",
	"payment_id",
	"method",
	"status",
	"subscribe_id",
	"subscribe_token",
	"product_snapshot",
}

var orderActivationColumns = []string{
	"id",
	"user_id",
	"order_no",
	"source",
	"type",
	"quantity",
	"price",
	"amount",
	"fee_amount",
	"coupon",
	"commission",
	"method",
	"trade_no",
	"status",
	"subscribe_id",
	"subscribe_token",
	"product_name",
	"product_snapshot",
	"is_new",
	"created_at",
}

var orderPurchaseQueryColumns = []string{
	"id",
	"order_no",
	"quantity",
	"price",
	"amount",
	"discount",
	"coupon",
	"coupon_discount",
	"payment_id",
	"fee_amount",
	"status",
	"subscribe_id",
	"product_snapshot",
	"created_at",
}

var orderBalanceCheckoutColumns = []string{
	"id",
	"user_id",
	"order_no",
	"amount",
	"status",
}

var orderPaymentNotifyColumns = []string{
	"id",
	"order_no",
	"amount",
	"settlement_amount",
	"settlement_currency",
	"status",
}

var orderCloseColumns = []string{
	"id",
	"user_id",
	"order_no",
	"type",
	"payment_id",
	"method",
	"trade_no",
	"gift_amount",
	"commission_deduction",
	"status",
	"subscribe_id",
}

var orderUnsubscribeRefundColumns = []string{
	"id",
	"order_no",
	"method",
	"gift_amount",
	"commission_deduction",
}

var orderListPaymentColumns = []string{
	"id",
	"name",
	"platform",
	"icon",
	"description",
	"fee_mode",
	"fee_percent",
	"fee_amount",
}

var adminOrderListPaymentColumns = []string{
	"id",
	"name",
	"platform",
}

var publicOrderListSubscribeColumns = []string{
	"id",
	"name",
}

var orderDetailSubscribeColumns = []string{
	"id",
	"name",
	"unit_price",
	"unit_time",
	"traffic",
	"speed_limit",
	"device_limit",
	"show_original_price",
}

func preloadOrderListPayment(db *gorm.DB) *gorm.DB {
	return db.Select(orderListPaymentColumns)
}

func preloadAdminOrderListPayment(db *gorm.DB) *gorm.DB {
	return db.Select(adminOrderListPaymentColumns)
}

func preloadPublicOrderListSubscribe(db *gorm.DB) *gorm.DB {
	return db.Select(publicOrderListSubscribeColumns)
}

func preloadOrderDetailSubscribe(db *gorm.DB) *gorm.DB {
	return db.Select(orderDetailSubscribeColumns)
}

func preloadInternalOrderSubOrders(db *gorm.DB) *gorm.DB {
	return db.Select(internalOrderSubOrderColumns)
}

func hydratePublicOrderListSubscribes(ctx context.Context, conn *gorm.DB, list []*Details) error {
	subscribeIDs := collectPublicOrderListSubscribeIDs(list)
	if len(subscribeIDs) == 0 {
		return nil
	}

	subscribes := make([]subscribe.Subscribe, 0, len(subscribeIDs))
	if err := conn.WithContext(ctx).
		Model(&subscribe.Subscribe{}).
		Select(publicOrderListSubscribeColumns).
		Where("id IN ?", subscribeIDs).
		Find(&subscribes).Error; err != nil {
		return err
	}
	subscribeIndex := newOrderListSubscribeIndex(subscribes)
	for _, item := range list {
		if item == nil || !orderTypeUsesSubscribeSummary(item.Type) {
			continue
		}
		item.Subscribe = subscribeIndex.Find(item.SubscribeId)
	}
	return nil
}

func collectPublicOrderListSubscribeIDs(list []*Details) []int64 {
	ids := newOrderListInt64Collector(len(list))
	for _, item := range list {
		if item == nil || !orderTypeUsesSubscribeSummary(item.Type) || item.SubscribeId <= 0 {
			continue
		}
		ids.Add(item.SubscribeId)
	}
	return ids.Values()
}

type orderListInt64Collector struct {
	values  []int64
	seen    map[int64]struct{}
	capHint int
}

func newOrderListInt64Collector(capHint int) orderListInt64Collector {
	return orderListInt64Collector{capHint: capHint}
}

func (c *orderListInt64Collector) Add(value int64) bool {
	if value <= 0 {
		return false
	}
	if c.seen != nil {
		if _, ok := c.seen[value]; ok {
			return false
		}
		c.seen[value] = struct{}{}
		c.values = append(c.values, value)
		return true
	}
	for _, existing := range c.values {
		if existing == value {
			return false
		}
	}
	if c.values == nil {
		c.values = make([]int64, 0, orderListSubscribeIndexInitialCap(c.capHint))
	}
	c.values = append(c.values, value)
	if len(c.values) > smallOrderListSubscribeIndexThreshold {
		c.upgradeLookup()
	}
	return true
}

func (c orderListInt64Collector) Values() []int64 {
	if len(c.values) == 0 {
		return []int64{}
	}
	return c.values
}

func (c *orderListInt64Collector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := orderListSubscribeLookupCap(len(c.values), c.capHint)
	if nextCap > cap(c.values) {
		values := make([]int64, len(c.values), nextCap)
		copy(values, c.values)
		c.values = values
	}
	c.seen = make(map[int64]struct{}, nextCap)
	for _, value := range c.values {
		c.seen[value] = struct{}{}
	}
}

type orderListSubscribeIndex struct {
	rows    []subscribe.Subscribe
	byID    map[int64]int
	capHint int
}

func newOrderListSubscribeIndex(subscribes []subscribe.Subscribe) orderListSubscribeIndex {
	index := orderListSubscribeIndex{capHint: len(subscribes)}
	for pos := range subscribes {
		index.Add(subscribes[pos])
	}
	return index
}

func (i *orderListSubscribeIndex) Add(item subscribe.Subscribe) {
	if item.Id <= 0 {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[item.Id]; ok {
			return
		}
		i.byID[item.Id] = len(i.rows)
		i.rows = append(i.rows, item)
		return
	}
	for pos := range i.rows {
		if i.rows[pos].Id == item.Id {
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, item)
	if len(i.rows) > smallOrderListSubscribeIndexThreshold {
		i.upgradeLookup()
	}
}

func (i orderListSubscribeIndex) Find(id int64) *subscribe.Subscribe {
	if id <= 0 {
		return nil
	}
	if i.byID != nil {
		pos, ok := i.byID[id]
		if !ok {
			return nil
		}
		return &i.rows[pos]
	}
	for pos := range i.rows {
		if i.rows[pos].Id == id {
			return &i.rows[pos]
		}
	}
	return nil
}

func (i *orderListSubscribeIndex) ensureRows() {
	if i.rows != nil || i.capHint <= 0 {
		return
	}
	i.rows = make([]subscribe.Subscribe, 0, orderListSubscribeIndexInitialCap(i.capHint))
}

func (i *orderListSubscribeIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := orderListSubscribeLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]subscribe.Subscribe, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byID = make(map[int64]int, nextCap)
	for pos := range i.rows {
		i.byID[i.rows[pos].Id] = pos
	}
}

func orderListSubscribeIndexInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallOrderListSubscribeIndexThreshold+1 {
		return smallOrderListSubscribeIndexThreshold + 1
	}
	return capHint
}

func orderListSubscribeLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func orderTypeUsesSubscribeSummary(orderType uint8) bool {
	switch orderType {
	case orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic:
		return true
	default:
		return false
	}
}

func applyOrderListFilter(conn *gorm.DB, status uint8, user, subscribe int64, search string) *gorm.DB {
	if status > 0 {
		conn = conn.Where("status = ?", status)
	}
	if user > 0 {
		conn = conn.Where("user_id = ?", user)
	}
	if subscribe > 0 {
		conn = conn.Where("subscribe_id = ? AND type IN ?", subscribe, []uint8{1, 2, 3})
	}
	if search != "" {
		conn = conn.Where("order_no like ? or trade_no like ? or coupon like ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	return conn
}

// UpdateOrderStatus Update order status
func (m *customOrderModel) UpdateOrderStatus(ctx context.Context, orderNo string, status uint8, tx ...*gorm.DB) error {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderStatusUpdateColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Order{}).Where("order_no = ?", orderNo).Update("status", status).Error
	}, m.getCacheKeys(&orderInfo)...)
}

func (m *customOrderModel) FindOneForStatusUpdate(ctx context.Context, id int64) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderStatusUpdateColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForCheckout(ctx context.Context, orderNo string) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderCheckoutColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForActivation(ctx context.Context, orderNo string) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderActivationColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForPurchaseQuery(ctx context.Context, orderNo string) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderPurchaseQueryColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForPaymentNotify(ctx context.Context, orderNo string) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderPaymentNotifyColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForClose(ctx context.Context, orderNo string) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderCloseColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForCloseByUser(ctx context.Context, orderNo string, userID int64) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderCloseColumns).
			Where("order_no = ? AND user_id = ?", orderNo, userID).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneForUnsubscribeRefund(ctx context.Context, id int64) (*Order, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderUnsubscribeRefundColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) UpdateFields(ctx context.Context, current *Order, update OrderFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Order{}).
			Where("id = ?", current.Id).
			Updates(updates).Error
	}, m.getCacheKeys(current)...)
}

func (m *customOrderModel) SetSettlementIfCompatible(ctx context.Context, current *Order, currency string, amount int64) (bool, error) {
	if current == nil || current.Id <= 0 || currency == "" || amount <= 0 {
		return false, nil
	}
	var rowsAffected int64
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		result := conn.Model(&Order{}).
			Where("id = ? AND status = ? AND (settlement_amount = 0 OR (settlement_currency = ? AND settlement_amount = ?))", current.Id, uint8(1), currency, amount).
			Updates(OrderFieldsUpdate{
				SettlementCurrency: UpdateValue(currency),
				SettlementAmount:   UpdateValue(amount),
			}.updates())
		rowsAffected = result.RowsAffected
		return result.Error
	}, m.getCacheKeys(current)...)
	return rowsAffected > 0, err
}

func (m *customOrderModel) UpdateAdminStatusFields(ctx context.Context, current *Order, update OrderFieldsUpdate, tx ...*gorm.DB) error {
	return m.UpdateFields(ctx, current, update, tx...)
}

func (m *customOrderModel) UpdateStatusIfCurrent(ctx context.Context, current *Order, fromStatus, toStatus uint8, tx ...*gorm.DB) (bool, error) {
	if current == nil {
		return false, nil
	}
	var rowsAffected int64
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		result := conn.Model(&Order{}).
			Where("id = ? AND status = ?", current.Id, fromStatus).
			Update("status", toStatus)
		rowsAffected = result.RowsAffected
		return result.Error
	}, m.getCacheKeys(current)...)
	return rowsAffected > 0, err
}

func (m *customOrderModel) MarkPaid(ctx context.Context, orderNo, tradeNo string) (bool, error) {
	var orderInfo Order
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderStatusUpdateColumns).
			Where("order_no = ?", orderNo).
			First(v).Error
	})
	if err != nil {
		return false, err
	}
	var rowsAffected int64
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		updates := OrderFieldsUpdate{Status: UpdateValue(uint8(2))}
		if tradeNo != "" {
			updates.TradeNo = &tradeNo
		}
		result := conn.Model(&Order{}).Where("order_no = ? AND status = ?", orderNo, uint8(1)).Updates(updates.updates())
		rowsAffected = result.RowsAffected
		return result.Error
	}, m.getCacheKeys(&orderInfo)...)
	return rowsAffected > 0, err
}

// FindOneDetailsByOrderNo Find order details by order number
func (m *customOrderModel) FindOneDetailsByOrderNo(ctx context.Context, orderNo string) (*Details, error) {
	var orderInfo Details
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(publicOrderDetailColumns).
			Where("order_no = ?", orderNo).
			Preload("Subscribe", preloadOrderDetailSubscribe).
			Preload("Payment", preloadOrderListPayment).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneDetailsByOrderNoForUser(ctx context.Context, orderNo string, userID int64) (*Details, error) {
	var orderInfo Details
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(publicOrderDetailColumns).
			Where("order_no = ? AND user_id = ?", orderNo, userID).
			Preload("Subscribe", preloadOrderDetailSubscribe).
			Preload("Payment", preloadOrderListPayment).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneAdminDetails(ctx context.Context, id int64) (*Details, error) {
	var orderInfo Details
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(adminOrderDetailColumns).
			Where("id = ?", id).
			Preload("Subscribe", preloadOrderDetailSubscribe).
			Preload("Payment", preloadOrderListPayment).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) FindOneDetails(ctx context.Context, id int64) (*Details, error) {
	var orderInfo Details
	err := m.QueryNoCacheCtx(ctx, &orderInfo, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(internalOrderDetailColumns).
			Where("id = ?", id).
			Preload("SubOrders", preloadInternalOrderSubOrders).
			First(v).Error
	})
	return &orderInfo, err
}

func (m *customOrderModel) QueryMonthlyOrders(ctx context.Context, date time.Time) (OrdersTotal, error) {
	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
	var result OrdersTotal
	err := m.QueryNoCacheCtx(ctx, &result, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Where("status IN ? AND created_at BETWEEN ? AND ? AND method != ?", []int64{2, 5}, firstDay, lastDay, "balance").
			Select(
				"COALESCE(SUM(amount), 0) as amount_total, "+
					"COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) as new_order_amount, "+
					"COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) as renewal_order_amount",
				true,
				false,
			).
			Scan(v).Error
	})
	return result, err
}

// QueryDateOrders Query orders by date
func (m *customOrderModel) QueryDateOrders(ctx context.Context, date time.Time) (OrdersTotal, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour).Add(-time.Nanosecond)
	var result OrdersTotal
	err := m.QueryNoCacheCtx(ctx, &result, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Where("status IN ? AND created_at BETWEEN ? AND ? AND method != ?", []int64{2, 5}, start, end, "balance").
			Select(
				"COALESCE(SUM(amount), 0) as amount_total, "+
					"COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) as new_order_amount, "+
					"COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) as renewal_order_amount",
				true,
				false,
			).
			Scan(v).Error
	})
	return result, err
}

func (m *customOrderModel) QueryTotalOrders(ctx context.Context) (OrdersTotal, error) {
	var result OrdersTotal

	err := m.QueryNoCacheCtx(ctx, &result, func(conn *gorm.DB, _ interface{}) error {
		return conn.Model(&Order{}).
			Select(`
				COALESCE(SUM(amount), 0) AS amount_total,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS new_order_amount,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS renewal_order_amount
			`, true, false).
			Where("status IN ? AND method != ?", []int64{2, 5}, "balance").
			Scan(&result).Error
	})

	return result, err
}

func (m *customOrderModel) QueryMonthlyUserCounts(ctx context.Context, date time.Time) (int64, int64, error) {
	// 获取当月第一天零点
	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	// 获取下个月第一天零点（避免漏掉最后一天的订单）
	nextMonth := firstDay.AddDate(0, 1, 0)

	var counts UserCounts

	// 执行查询
	err := m.QueryNoCacheCtx(ctx, nil, func(conn *gorm.DB, _ interface{}) error {
		return conn.Model(&Order{}).
			Select(`
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS new_users,
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS renewal_users
			`, true, false).
			Where("status IN ? AND created_at >= ? AND created_at < ? AND method != ?",
				[]int64{2, 5}, firstDay, nextMonth, "balance").
			Scan(&counts).Error
	})

	return counts.NewUsers, counts.RenewalUsers, err
}
func (m *customOrderModel) QueryDateUserCounts(ctx context.Context, date time.Time) (int64, int64, error) {
	// 当天 00:00:00
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	// 下一天 00:00:00
	nextDay := start.Add(24 * time.Hour)

	var counts UserCounts

	err := m.QueryNoCacheCtx(ctx, nil, func(conn *gorm.DB, _ interface{}) error {
		return conn.Model(&Order{}).
			Select(`
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS new_users,
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS renewal_users
			`, true, false).
			Where("status IN ? AND created_at >= ? AND created_at < ? AND method != ?",
				[]int64{2, 5}, start, nextDay, "balance").
			Scan(&counts).Error
	})

	return counts.NewUsers, counts.RenewalUsers, err
}
func (m *customOrderModel) QueryTotalUserCounts(ctx context.Context) (int64, int64, error) {
	var counts UserCounts

	err := m.QueryNoCacheCtx(ctx, nil, func(conn *gorm.DB, _ interface{}) error {
		return conn.Model(&Order{}).
			Where("status IN ? AND method != ?", []int64{2, 5}, "balance").
			Select(`
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS new_users,
				COUNT(DISTINCT CASE WHEN is_new = ? THEN user_id END) AS renewal_users
			`, true, false).
			Scan(&counts).Error
	})

	return counts.NewUsers, counts.RenewalUsers, err
}

func (m *customOrderModel) IsUserEligibleForNewOrder(ctx context.Context, userID int64) (bool, error) {
	var matched struct{ Id int64 }
	err := m.QueryNoCacheCtx(ctx, &matched, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select("id").
			Where("user_id = ? AND status IN ?", userID, []int64{2, 5}).
			Limit(1).
			Scan(v).Error
	})
	return matched.Id == 0, err
}

// QueryDailyOrdersList 查询当月每日订单统计
func (m *customOrderModel) QueryDailyOrdersList(ctx context.Context, date time.Time) ([]OrdersTotalWithDate, error) {
	var results []OrdersTotalWithDate

	err := m.QueryNoCacheCtx(ctx, &results, func(conn *gorm.DB, v interface{}) error {
		// 当月 1 号 00:00:00
		firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		// 第二天 00:00:00
		nextDay := date.AddDate(0, 0, 1).Truncate(24 * time.Hour)

		return conn.Model(&Order{}).
			Select(`
				TO_CHAR(created_at, 'YYYY-MM-DD') AS date,
				COALESCE(SUM(amount), 0) AS amount_total,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS new_order_amount,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS renewal_order_amount
			`, true, false).
			Where("status IN ? AND created_at >= ? AND created_at < ? AND method != ?",
				[]int64{2, 5}, firstDay, nextDay, "balance").
			Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
			Order("date ASC").
			Scan(v).Error
	})
	return results, err
}

// QueryMonthlyOrdersList 查询过去 6 个月订单统计（包含当前月）
func (m *customOrderModel) QueryMonthlyOrdersList(ctx context.Context, date time.Time) ([]OrdersTotalWithDate, error) {
	var results []OrdersTotalWithDate

	err := m.QueryNoCacheCtx(ctx, &results, func(conn *gorm.DB, v interface{}) error {
		// 六个月前（取月初）
		start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location()).AddDate(0, -5, 0)
		// 下个月月初
		end := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location()).AddDate(0, 1, 0)

		return conn.Model(&Order{}).
			Select(`
				TO_CHAR(created_at, 'YYYY-MM') AS date,
				COALESCE(SUM(amount), 0) AS amount_total,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS new_order_amount,
				COALESCE(SUM(CASE WHEN is_new = ? THEN amount ELSE 0 END), 0) AS renewal_order_amount
			`, true, false).
			Where("status IN ? AND created_at >= ? AND created_at < ? AND method != ?",
				[]int64{2, 5}, start, end, "balance").
			Group("TO_CHAR(created_at, 'YYYY-MM')").
			Order("date ASC").
			Scan(v).Error
	})
	return results, err
}
