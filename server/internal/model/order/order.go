package order

import (
	"time"

	"github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
)

type Order struct {
	Id                  int64     `gorm:"primaryKey"`
	ParentId            int64     `gorm:"type:bigint;default:null;comment:Parent Order Id"`
	UserId              int64     `gorm:"type:bigint;not null;default:0;comment:User Id"`
	OrderNo             string    `gorm:"type:varchar(255);not null;default:'';unique;comment:Order No"`
	RequestId           string    `gorm:"type:varchar(64);not null;default:'';comment:Client idempotency key"`
	Source              string    `gorm:"type:varchar(32);not null;default:'customer';comment:Order Source"`
	Type                uint8     `gorm:"type:smallint;not null;default:1;comment:Order Type: 1: Subscribe, 2: Renewal, 3: ResetTraffic, 4: Recharge, 5: Membership"`
	Quantity            int64     `gorm:"type:bigint;not null;default:1;comment:Quantity"`
	Price               int64     `gorm:"type:int;not null;default:0;comment:Original price"`
	Amount              int64     `gorm:"type:int;not null;default:0;comment:Order Amount"`
	GiftAmount          int64     `gorm:"type:int;not null;default:0;comment:User Gift Amount"`
	CommissionDeduction int64     `gorm:"type:bigint;not null;default:0;comment:Commission balance applied to order"`
	Discount            int64     `gorm:"type:int;not null;default:0;comment:Discount Amount"`
	Coupon              string    `gorm:"type:varchar(255);default:null;comment:Coupon"`
	CouponDiscount      int64     `gorm:"type:int;not null;default:0;comment:Coupon Discount Amount"`
	Commission          int64     `gorm:"type:int;not null;default:0;comment:Order Commission"`
	PaymentId           int64     `gorm:"type:bigint;not null;default:0;comment:Payment Method Id"`
	Method              string    `gorm:"type:varchar(255);not null;default:'';comment:Payment Method"`
	FeeAmount           int64     `gorm:"type:int;not null;default:0;comment:Fee Amount"`
	SettlementAmount    int64     `gorm:"type:bigint;not null;default:0;comment:External payment settlement amount in minor units"`
	SettlementCurrency  string    `gorm:"type:varchar(8);not null;default:'';comment:External payment settlement currency"`
	TradeNo             string    `gorm:"type:varchar(255);default:null;comment:Trade No"`
	Status              uint8     `gorm:"type:smallint;not null;default:1;comment:Order Status: 1: Pending, 2: Paid, 3:Close, 4: Failed, 5:Finished;"`
	SubscribeId         int64     `gorm:"type:bigint;not null;default:0;comment:Subscribe Id"`
	SubscribeToken      string    `gorm:"type:varchar(255);default:null;comment:Renewal Subscribe Token"`
	ProductName         string    `gorm:"type:varchar(255);not null;default:'';comment:Product Name Snapshot"`
	ProductSnapshot     string    `gorm:"type:text;not null;default:'{}';comment:Product Snapshot"`
	ClientIP            string    `gorm:"type:varchar(64);not null;default:'';comment:Client IP"`
	CountryShort        string    `gorm:"type:varchar(8);not null;default:'';comment:Client IP Country Code"`
	Region              string    `gorm:"type:varchar(128);not null;default:'';comment:Client IP Region"`
	Location            string    `gorm:"type:varchar(160);not null;default:'';comment:Client IP Location"`
	IsNew               bool      `gorm:"type:boolean;not null;default:false;comment:Is New Order"`
	CreatedAt           time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt           time.Time `gorm:"comment:Update Time"`
}

type OrdersTotal struct {
	AmountTotal        int64
	NewOrderAmount     int64
	RenewalOrderAmount int64
}

func (Order) TableName() string {
	return "order"
}

func (o *Order) BeforeSave(*gorm.DB) error {
	if o != nil {
		o.ClientIP = privacy.MaskIP(o.ClientIP)
	}
	return nil
}
