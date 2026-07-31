package user

import (
	"time"

	"github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
)

type User struct {
	Id                    int64          `gorm:"primaryKey"`
	Password              string         `gorm:"type:varchar(100);not null;comment:User Password"`
	Algo                  string         `gorm:"type:varchar(20);default:'default';comment:Encryption Algorithm"`
	Salt                  string         `gorm:"type:varchar(20);default:null;comment:Password Salt"`
	Avatar                string         `gorm:"type:text;comment:User Avatar"`
	Balance               int64          `gorm:"default:0;comment:User Balance"` // User Balance Amount
	ReferCode             string         `gorm:"type:varchar(20);index:idx_user_refer_code;default:'';comment:Referral Code"`
	RefererId             int64          `gorm:"index:idx_referer;comment:Referrer ID"`
	Commission            int64          `gorm:"default:0;comment:Commission"`                      // Commission Amount
	ReferralPercentage    uint8          `gorm:"default:0;comment:Referral"`                        // Referral Percentage
	OnlyFirstPurchase     *bool          `gorm:"default:true;not null;comment:Only First Purchase"` // Only First Purchase Referral
	GiftAmount            int64          `gorm:"default:0;comment:User Gift Amount"`
	MemberExpiredAt       time.Time      `gorm:"default:NULL;comment:Membership Expire Time"`
	Enable                *bool          `gorm:"default:true;not null;comment:Is Account Enabled"`
	IsAdmin               *bool          `gorm:"default:false;not null;comment:Is Admin"`
	EnableBalanceNotify   *bool          `gorm:"default:false;not null;comment:Enable Balance Change Notifications"`
	EnableLoginNotify     *bool          `gorm:"default:false;not null;comment:Enable Login Notifications"`
	EnableSubscribeNotify *bool          `gorm:"default:false;not null;comment:Enable Subscription Notifications"`
	EnableTradeNotify     *bool          `gorm:"default:false;not null;comment:Enable Trade Notifications"`
	AuthMethods           []AuthMethods  `gorm:"foreignKey:UserId;references:Id"`
	UserDevices           []Device       `gorm:"foreignKey:UserId;references:Id"`
	Rules                 string         `gorm:"type:TEXT;comment:User Rules"`
	CreatedAt             time.Time      `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt             time.Time      `gorm:"comment:Update Time"`
	DeletedAt             gorm.DeletedAt `gorm:"index;comment:Deletion Time"`
}

func (*User) TableName() string {
	return "user"
}

type Subscribe struct {
	Id                 int64      `gorm:"primaryKey"`
	UserId             int64      `gorm:"index:idx_user_id;not null;comment:User ID"`
	User               User       `gorm:"foreignKey:UserId;references:Id"`
	OrderId            int64      `gorm:"index:idx_order_id;not null;comment:Order ID"`
	SubscribeId        int64      `gorm:"index:idx_subscribe_id;not null;comment:Subscription ID"`
	ProductName        string     `gorm:"type:varchar(255);not null;default:'';comment:Product Name Snapshot"`
	ProductSnapshot    string     `gorm:"type:text;not null;default:'{}';comment:Product Snapshot"`
	StartTime          time.Time  `gorm:"default:CURRENT_TIMESTAMP;not null;comment:Subscription Start Time"`
	ExpireTime         time.Time  `gorm:"default:'1970-01-01 00:00:00+00:00';not null;comment:Subscription Expire Time"`
	FinishedAt         *time.Time `gorm:"default:NULL;comment:Finished Time"`
	Traffic            int64      `gorm:"default:0;comment:Traffic"`
	Download           int64      `gorm:"default:0;comment:Download Traffic"`
	Upload             int64      `gorm:"default:0;comment:Upload Traffic"`
	TrafficResetAt     time.Time  `gorm:"default:'1970-01-01 00:00:00+00:00';not null;comment:Traffic Reset Barrier"`
	ResetCycle         uint8      `gorm:"type:smallint;default:0;not null;comment:Traffic Reset Cycle"`
	NextTrafficResetAt *time.Time `gorm:"default:NULL;comment:Next Traffic Reset Time"`
	Token              string     `gorm:"index:idx_token;unique;type:varchar(255);default:'';comment:Token"`
	UUID               string     `gorm:"type:varchar(255);unique;index:idx_uuid;default:'';comment:UUID"`
	Status             uint8      `gorm:"type:smallint;default:0;comment:Subscription Status: 0: Pending 1: Active 2: Exhausted 3: Expired 4: Deducted 5: Stopped"`
	Note               string     `gorm:"type:varchar(500);default:'';comment:User note for subscription"`
	CreatedAt          time.Time  `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt          time.Time  `gorm:"comment:Update Time"`
}

func (*Subscribe) TableName() string {
	return "user_subscribe"
}

type AuthMethods struct {
	Id                       int64     `gorm:"primaryKey"`
	UserId                   int64     `gorm:"index:idx_user_id;not null;comment:User ID"`
	AuthType                 string    `gorm:"type:varchar(255);not null;comment:Authentication method"`
	AuthIdentifier           string    `gorm:"type:text;not null;comment:Encrypted auth identifier"`
	AuthIdentifierBlindIndex string    `gorm:"column:auth_identifier_bidx;type:varchar(64);index:idx_user_auth_type_bidx;default:'';not null;comment:Auth identifier blind index"`
	AuthIdentifierDisplay    string    `gorm:"column:auth_identifier_display;type:varchar(255);default:'';not null;comment:Masked auth identifier"`
	Verified                 bool      `gorm:"default:false;not null;comment:Is Verified"`
	CreatedAt                time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt                time.Time `gorm:"comment:Update Time"`
}

func (*AuthMethods) TableName() string {
	return "user_auth_methods"
}

type Device struct {
	Id                   int64     `gorm:"primaryKey"`
	Ip                   string    `gorm:"type:text;not null;comment:Encrypted device IP"`
	IPDisplay            string    `gorm:"column:ip_display;type:varchar(64);not null;default:'';comment:Masked device IP"`
	UserId               int64     `gorm:"index:idx_user_id;not null;comment:User ID"`
	UserAgent            string    `gorm:"type:text;default:null;comment:Encrypted UserAgent"`
	UserAgentDisplay     string    `gorm:"column:user_agent_display;type:varchar(255);not null;default:'';comment:Masked UserAgent"`
	Identifier           string    `gorm:"column:identifier;type:text;default:'';comment:Encrypted device identifier"`
	IdentifierBlindIndex string    `gorm:"column:identifier_bidx;type:varchar(64);index:idx_user_device_identifier_bidx;not null;default:'';comment:Device identifier blind index"`
	IdentifierDisplay    string    `gorm:"column:identifier_display;type:varchar(255);not null;default:'';comment:Masked device identifier"`
	Online               bool      `gorm:"default:false;not null;comment:Online"`
	Enabled              bool      `gorm:"default:true;not null;comment:Enabled"`
	CreatedAt            time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt            time.Time `gorm:"comment:Update Time"`
}

func (*Device) TableName() string {
	return "user_device"
}

type DeviceOnlineRecord struct {
	Id            int64     `gorm:"primaryKey"`
	UserId        int64     `gorm:"type:bigint;not null;comment:User ID"`
	Identifier    string    `gorm:"type:varchar(255);not null;comment:Device Identifier"`
	OnlineTime    time.Time `gorm:"comment:Online Time"` // The time when the device goes online
	OfflineTime   time.Time `gorm:"comment:Offline Time"`
	OnlineSeconds int64     `gorm:"comment:Offline Seconds"`
	DurationDays  int64     `gorm:"comment:Duration Days"`
	CreatedAt     time.Time `gorm:"<-:create;comment:Creation Time"`
}

func (DeviceOnlineRecord) TableName() string {
	return "user_device_online_record"
}

func (r *DeviceOnlineRecord) BeforeSave(*gorm.DB) error {
	if r != nil {
		r.Identifier = privacy.MaskIdentifier(r.Identifier)
	}
	return nil
}

type Withdrawal struct {
	Id        int64     `gorm:"primaryKey"`
	UserId    int64     `gorm:"index:idx_user_id;not null;comment:User ID"`
	Amount    int64     `gorm:"not null;comment:Withdrawal Amount"`
	Content   string    `gorm:"type:text;comment:Withdrawal Content"`
	Status    uint8     `gorm:"type:smallint;default:0;comment:Withdrawal Status: 0: Pending 1: Approved 2: Rejected"`
	Reason    string    `gorm:"type:varchar(500);default:'';comment:Rejection Reason"`
	CreatedAt time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt time.Time `gorm:"comment:Update Time"`
}

func (*Withdrawal) TableName() string {
	return "user_withdrawal"
}
