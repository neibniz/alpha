package membership

import "time"

type Plan struct {
	Id                             int64     `gorm:"primaryKey"`
	Name                           string    `gorm:"type:varchar(255);not null;default:'';comment:Membership Plan Name"`
	Description                    string    `gorm:"type:text;not null;default:'';comment:Membership Plan Description"`
	UnitPrice                      int64     `gorm:"type:int;not null;default:0;comment:Membership Unit Price"`
	DurationUnit                   string    `gorm:"type:varchar(32);not null;default:'Year';comment:Membership Duration Unit"`
	DurationValue                  int64     `gorm:"type:bigint;not null;default:1;comment:Membership Duration Value"`
	Enabled                        *bool     `gorm:"type:boolean;not null;default:true;comment:Is Enabled"`
	MonthlyConsumptionLimitEnabled bool      `gorm:"type:boolean;not null;default:false;comment:Enable Monthly Consumption Limit"`
	MonthlyConsumptionLimit        int64     `gorm:"type:bigint;not null;default:0;comment:Monthly Consumption Limit"`
	MonthlyOrderLimitEnabled       bool      `gorm:"type:boolean;not null;default:false;comment:Enable Monthly Order Limit"`
	MonthlyOrderLimit              int64     `gorm:"type:bigint;not null;default:0;comment:Monthly Order Limit"`
	SubscriptionExpiryLimitEnabled bool      `gorm:"type:boolean;not null;default:false;comment:Enable Subscription Expiry Limit"`
	SubscriptionExpiryGraceDays    int64     `gorm:"type:bigint;not null;default:0;comment:Allowed Days Subscription Can Exceed Membership Expiry"`
	Sort                           int64     `gorm:"type:bigint;not null;default:0;comment:Sort"`
	CreatedAt                      time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt                      time.Time `gorm:"comment:Update Time"`
}

func (Plan) TableName() string {
	return "membership_plan"
}
