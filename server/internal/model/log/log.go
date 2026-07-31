package log

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
)

type Type uint8

/*

Log Types:
	1X Message Logs
	2X Subscription Logs
	3X User Logs
	4X Traffic Ranking Logs
*/

const (
	TypeEmailMessage       Type = 10 // Message log
	TypeSubscribe          Type = 20 // Subscription log
	TypeSubscribeTraffic   Type = 21 // Subscription traffic log
	TypeServerTraffic      Type = 22 // Server traffic log
	TypeResetSubscribe     Type = 23 // Reset subscription log
	TypeAdjustSubscribe    Type = 24 // Administrator subscription adjustment log
	TypeLogin              Type = 30 // Login log
	TypeRegister           Type = 31 // Registration log
	TypeBalance            Type = 32 // Balance log
	TypeCommission         Type = 33 // Commission log
	TypeGift               Type = 34 // Gift log
	TypeLogout             Type = 35 // Logout log
	TypePrivateDataReveal  Type = 36 // Administrator private identity reveal
	TypeSensitiveLogReveal Type = 37 // Administrator sensitive log reveal
	TypeUserTrafficRank    Type = 40 // Top 10 User traffic rank log
	TypeServerTrafficRank  Type = 41 // Top 10 Server traffic rank log
	TypeTrafficStat        Type = 42 // Daily traffic statistics log
)
const (
	ResetSubscribeTypeAuto       uint16 = 231 // Auto reset
	ResetSubscribeTypeAdvance    uint16 = 232 // Advance reset
	ResetSubscribeTypePaid       uint16 = 233 // Paid reset
	ResetSubscribeTypeQuota      uint16 = 234 // Quota reset
	BalanceTypeRecharge          uint16 = 321 // Recharge
	BalanceTypeWithdraw          uint16 = 322 // Withdraw
	BalanceTypePayment           uint16 = 323 // Payment
	BalanceTypeRefund            uint16 = 324 // Refund
	BalanceTypeAdjust            uint16 = 326 // Admin Adjust
	BalanceTypeReward            uint16 = 325 // Reward
	CommissionTypePurchase       uint16 = 331 // Purchase
	CommissionTypeRenewal        uint16 = 332 // Renewal
	CommissionTypeRefund         uint16 = 333 // Refund
	CommissionTypeWithdraw       uint16 = 334 // withdraw
	CommissionTypeAdjust         uint16 = 335 // Admin Adjust
	CommissionTypeConvertBalance uint16 = 336 // Convert to Balance
	CommissionTypePayment        uint16 = 337 // Order payment deduction
	CommissionTypePaymentRefund  uint16 = 338 // Order payment refund
	GiftTypeIncrease             uint16 = 341 // Increase
	GiftTypeReduce               uint16 = 342 // Reduce
)

// Uint8 converts Type to uint8.
func (t Type) Uint8() uint8 {
	return uint8(t)
}

// SystemLog represents a log entry in the system.
type SystemLog struct {
	Id           int64     `gorm:"primaryKey;autoIncrement"`
	Type         uint8     `gorm:"index:idx_type;type:smallint;not null;default:0;comment:Log type code"`
	Date         string    `gorm:"type:varchar(20);default:null;comment:Log Date"`
	ObjectID     int64     `gorm:"index:idx_object_id;type:bigint;not null;default:0;comment:Object ID"`
	IP           string    `gorm:"type:varchar(64);not null;default:'';comment:Client IP"`
	CountryShort string    `gorm:"type:varchar(8);not null;default:'';comment:Client IP Country Code"`
	Region       string    `gorm:"type:varchar(128);not null;default:'';comment:Client IP Region"`
	Location     string    `gorm:"type:varchar(160);not null;default:'';comment:Client IP Location"`
	Method       string    `gorm:"type:varchar(64);not null;default:'';comment:Login or auth method"`
	Identifier   string    `gorm:"type:varchar(255);not null;default:'';comment:Masked-capable auth identifier source"`
	UserAgent    string    `gorm:"type:varchar(512);not null;default:'';comment:Client user agent"`
	Success      bool      `gorm:"not null;default:false;comment:Whether action succeeded"`
	EventAt      int64     `gorm:"not null;default:0;comment:Client event timestamp in milliseconds"`
	Content      string    `gorm:"type:text;not null;comment:Log Content"`
	PrivacyRef   string    `gorm:"type:varchar(36);not null;default:'';comment:Privacy encryption reference" json:"-"`
	PrivateData  string    `gorm:"type:text;not null;default:'';comment:Encrypted original sensitive fields" json:"-"`
	CreatedAt    time.Time `gorm:"<-:create;comment:Create Time"`
}

type SubscribeAccessLog struct {
	Id              int64     `gorm:"primaryKey;autoIncrement"`
	UserId          int64     `gorm:"index:idx_subscribe_access_user_created,priority:1;not null;default:0;comment:User ID"`
	UserSubscribeId int64     `gorm:"index:idx_subscribe_access_user_sub_created,priority:1;not null;default:0;comment:User Subscribe ID"`
	TokenHash       string    `gorm:"type:varchar(64);not null;default:'';comment:Subscribe Token SHA256"`
	TokenPrefix     string    `gorm:"type:varchar(12);not null;default:'';comment:Subscribe Token Prefix"`
	UserAgent       string    `gorm:"type:varchar(512);not null;default:'';comment:User Agent"`
	ClientIP        string    `gorm:"type:varchar(64);not null;default:'';comment:Client IP"`
	CountryShort    string    `gorm:"type:varchar(8);not null;default:'';comment:Client IP Country Code"`
	Region          string    `gorm:"type:varchar(128);not null;default:'';comment:Client IP Region"`
	Location        string    `gorm:"type:varchar(160);not null;default:'';comment:Client IP Location"`
	Success         bool      `gorm:"not null;default:true;comment:Whether Subscribe Request Succeeded"`
	PrivacyRef      string    `gorm:"type:varchar(36);not null;default:'';comment:Privacy encryption reference" json:"-"`
	PrivateData     string    `gorm:"type:text;not null;default:'';comment:Encrypted original sensitive fields" json:"-"`
	Revealed        bool      `gorm:"-"`
	CreatedAt       time.Time `gorm:"<-:create;index:idx_subscribe_access_created;index:idx_subscribe_access_user_created,priority:2;index:idx_subscribe_access_user_sub_created,priority:2;comment:Create Time"`
}

func (SubscribeAccessLog) TableName() string {
	return "subscribe_access_logs"
}

func HashSubscribeToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TokenPrefix(token string) string {
	if len(token) <= 8 {
		return strings.Clone(token)
	}
	return strings.Clone(token[:8])
}

func marshalLogJSON[T any](value *T) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(value)
}

// TableName returns the name of the table for SystemLogs.
func (SystemLog) TableName() string {
	return "system_logs"
}

func (s *SystemLog) BeforeSave(*gorm.DB) error {
	if s == nil {
		return nil
	}
	s.IP = privacy.MaskIP(s.IP)
	s.UserAgent = privacy.MaskUserAgent(s.UserAgent)
	s.Identifier = privacy.MaskIdentifier(s.Identifier)
	s.Content = string(privacy.MaskJSON([]byte(s.Content)))
	return nil
}

func (s *SubscribeAccessLog) BeforeSave(*gorm.DB) error {
	if s == nil {
		return nil
	}
	s.ClientIP = privacy.MaskIP(s.ClientIP)
	s.UserAgent = privacy.MaskUserAgent(s.UserAgent)
	return nil
}

// Message represents a message log entry.
type Message struct {
	To       string          `json:"to"`
	Subject  string          `json:"subject,omitempty"`
	Content  json.RawMessage `json:"content"`
	Platform string          `json:"platform"`
	Template string          `json:"template"`
	Status   uint8           `json:"status"` // 1: Sent, 2: Failed
}

// Marshal implements the json.Marshaler interface for Message.
func (m *Message) Marshal() ([]byte, error) {
	type Alias Message
	return marshalLogJSON((*Alias)(m))
}

// Unmarshal implements the json.Unmarshaler interface for Message.
func (m *Message) Unmarshal(data []byte) error {
	type Alias Message
	return json.Unmarshal(data, (*Alias)(m))
}

// Traffic represents a subscription traffic log entry.
type Traffic struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

// Marshal implements the json.Marshaler interface for SubscribeTraffic.
func (s *Traffic) Marshal() ([]byte, error) {
	type Alias Traffic
	return marshalLogJSON((*Alias)(s))
}

// Unmarshal implements the json.Unmarshaler interface for SubscribeTraffic.
func (s *Traffic) Unmarshal(data []byte) error {
	type Alias Traffic
	return json.Unmarshal(data, (*Alias)(s))
}

// Login represents a login log entry.
type Login struct {
	Method    string `json:"method"`
	LoginIP   string `json:"login_ip"`
	UserAgent string `json:"user_agent"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Login.
func (l *Login) Marshal() ([]byte, error) {
	type Alias Login
	return marshalLogJSON((*Alias)(l))
}

// Unmarshal implements the json.Unmarshaler interface for Login.
func (l *Login) Unmarshal(data []byte) error {
	type Alias Login
	return json.Unmarshal(data, (*Alias)(l))
}

// Logout represents a logout log entry.
type Logout struct {
	LogoutIP  string `json:"logout_ip"`
	UserAgent string `json:"user_agent"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Logout.
func (l *Logout) Marshal() ([]byte, error) {
	type Alias Logout
	return marshalLogJSON((*Alias)(l))
}

// Unmarshal implements the json.Unmarshaler interface for Logout.
func (l *Logout) Unmarshal(data []byte) error {
	type Alias Logout
	return json.Unmarshal(data, (*Alias)(l))
}

// Register represents a registration log entry.
type Register struct {
	AuthMethod string `json:"auth_method"`
	Identifier string `json:"identifier"`
	RegisterIP string `json:"register_ip"`
	UserAgent  string `json:"user_agent"`
	Timestamp  int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Register.
func (r *Register) Marshal() ([]byte, error) {
	type Alias Register
	return marshalLogJSON((*Alias)(r))
}

// Unmarshal implements the json.Unmarshaler interface for Register.

func (r *Register) Unmarshal(data []byte) error {
	type Alias Register
	return json.Unmarshal(data, (*Alias)(r))
}

// Subscribe represents a subscription log entry.
type Subscribe struct {
	Token           string `json:"token"`
	UserAgent       string `json:"user_agent"`
	ClientIP        string `json:"client_ip"`
	UserSubscribeId int64  `json:"user_subscribe_id"`
}

// Marshal implements the json.Marshaler interface for Subscribe.
func (s *Subscribe) Marshal() ([]byte, error) {
	type Alias Subscribe
	return marshalLogJSON((*Alias)(s))
}

// Unmarshal implements the json.Unmarshaler interface for Subscribe.
func (s *Subscribe) Unmarshal(data []byte) error {
	type Alias Subscribe
	return json.Unmarshal(data, (*Alias)(s))
}

// ResetSubscribe represents a reset subscription log entry.
type ResetSubscribe struct {
	Type           uint16 `json:"type"`
	UserId         int64  `json:"user_id"`
	OrderNo        string `json:"order_no,omitempty"`
	BatchId        string `json:"batch_id,omitempty"`
	Scope          string `json:"scope,omitempty"`
	OperatorUserId int64  `json:"operator_user_id,omitempty"`
	Timestamp      int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for ResetSubscribe.
func (r *ResetSubscribe) Marshal() ([]byte, error) {
	type Alias ResetSubscribe
	return marshalLogJSON((*Alias)(r))
}

// Unmarshal implements the json.Unmarshaler interface for ResetSubscribe.
func (r *ResetSubscribe) Unmarshal(data []byte) error {
	type Alias ResetSubscribe
	return json.Unmarshal(data, (*Alias)(r))
}

type SubscribeAdjustmentState struct {
	SubscribeId int64 `json:"subscribe_id"`
	Traffic     int64 `json:"traffic"`
	Download    int64 `json:"download"`
	Upload      int64 `json:"upload"`
	ExpireAt    int64 `json:"expire_at"`
	Status      uint8 `json:"status"`
}

// SubscribeAdjustment records an administrator entitlement change without secrets or PII.
type SubscribeAdjustment struct {
	UserId         int64                    `json:"user_id"`
	OperatorUserId int64                    `json:"operator_user_id"`
	Before         SubscribeAdjustmentState `json:"before"`
	After          SubscribeAdjustmentState `json:"after"`
	Timestamp      int64                    `json:"timestamp"`
}

// PrivateDataReveal records access to decrypted identity data without storing the data itself.
type PrivateDataReveal struct {
	OperatorUserId int64    `json:"operator_user_id"`
	TargetUserId   int64    `json:"target_user_id"`
	Reason         string   `json:"reason"`
	AuthTypes      []string `json:"auth_types,omitempty"`
	Timestamp      int64    `json:"timestamp"`
}

func (r *PrivateDataReveal) Marshal() ([]byte, error) {
	type Alias PrivateDataReveal
	return marshalLogJSON((*Alias)(r))
}

func (r *PrivateDataReveal) Unmarshal(data []byte) error {
	type Alias PrivateDataReveal
	return json.Unmarshal(data, (*Alias)(r))
}

// SensitiveLogReveal records audited access to encrypted log fields.
type SensitiveLogReveal struct {
	OperatorUserId int64  `json:"operator_user_id"`
	Scope          string `json:"scope"`
	Reason         string `json:"reason"`
	RecordCount    int    `json:"record_count"`
	Timestamp      int64  `json:"timestamp"`
}

func (r *SensitiveLogReveal) Marshal() ([]byte, error) {
	type Alias SensitiveLogReveal
	return marshalLogJSON((*Alias)(r))
}

func (r *SensitiveLogReveal) Unmarshal(data []byte) error {
	type Alias SensitiveLogReveal
	return json.Unmarshal(data, (*Alias)(r))
}

func (a *SubscribeAdjustment) Marshal() ([]byte, error) {
	type Alias SubscribeAdjustment
	return marshalLogJSON((*Alias)(a))
}

func (a *SubscribeAdjustment) Unmarshal(data []byte) error {
	type Alias SubscribeAdjustment
	return json.Unmarshal(data, (*Alias)(a))
}

// Balance represents a balance log entry.
type Balance struct {
	Type      uint16 `json:"type"`
	Amount    int64  `json:"amount"`
	OrderNo   string `json:"order_no,omitempty"`
	Balance   int64  `json:"balance"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Balance.
func (b *Balance) Marshal() ([]byte, error) {
	type Alias Balance
	return marshalLogJSON((*Alias)(b))
}

// Unmarshal implements the json.Unmarshaler interface for Balance.
func (b *Balance) Unmarshal(data []byte) error {
	type Alias Balance
	return json.Unmarshal(data, (*Alias)(b))
}

// Commission represents a commission log entry.
type Commission struct {
	Type      uint16 `json:"type"`
	Amount    int64  `json:"amount"`
	OrderNo   string `json:"order_no"`
	Balance   int64  `json:"balance,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Commission.
func (c *Commission) Marshal() ([]byte, error) {
	type Alias Commission
	return marshalLogJSON((*Alias)(c))
}

// Unmarshal implements the json.Unmarshaler interface for Commission.
func (c *Commission) Unmarshal(data []byte) error {
	type Alias Commission
	return json.Unmarshal(data, (*Alias)(c))
}

// Gift represents a gift log entry.
type Gift struct {
	Type        uint16 `json:"type"`
	OrderNo     string `json:"order_no"`
	SubscribeId int64  `json:"subscribe_id"`
	Amount      int64  `json:"amount"`
	Balance     int64  `json:"balance"`
	Remark      string `json:"remark,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Gift.
func (g *Gift) Marshal() ([]byte, error) {
	type Alias Gift
	return marshalLogJSON((*Alias)(g))
}

// Unmarshal implements the json.Unmarshaler interface for Gift.
func (g *Gift) Unmarshal(data []byte) error {
	type Alias Gift
	return json.Unmarshal(data, (*Alias)(g))
}

// UserTraffic represents a user traffic log entry.
type UserTraffic struct {
	SubscribeId int64 `json:"subscribe_id"` // Subscribe ID
	UserId      int64 `json:"user_id"`      // User ID
	Upload      int64 `json:"upload"`       // Upload traffic in bytes
	Download    int64 `json:"download"`     // Download traffic in bytes
	Total       int64 `json:"total"`        // Total traffic in bytes (Upload + Download)
}

// Marshal implements the json.Marshaler interface for UserTraffic.
func (u *UserTraffic) Marshal() ([]byte, error) {
	type Alias UserTraffic
	return marshalLogJSON((*Alias)(u))
}

// Unmarshal implements the json.Unmarshaler interface for UserTraffic.
func (u *UserTraffic) Unmarshal(data []byte) error {
	type Alias UserTraffic
	return json.Unmarshal(data, (*Alias)(u))
}

// UserTrafficRank represents a user traffic rank entry.
type UserTrafficRank struct {
	Rank map[uint8]UserTraffic `json:"rank"` // Key is rank ,type is UserTraffic
}

// Marshal implements the json.Marshaler interface for UserTrafficRank.
func (u *UserTrafficRank) Marshal() ([]byte, error) {
	type Alias UserTrafficRank
	return marshalLogJSON((*Alias)(u))
}

// Unmarshal implements the json.Unmarshaler interface for UserTrafficRank.
func (u *UserTrafficRank) Unmarshal(data []byte) error {
	type Alias UserTrafficRank
	return json.Unmarshal(data, (*Alias)(u))
}

// ServerTraffic represents a server traffic log entry.
type ServerTraffic struct {
	ServerId int64 `json:"server_id"` // Server ID
	Upload   int64 `json:"upload"`    // Upload traffic in bytes
	Download int64 `json:"download"`  // Download traffic in bytes
	Total    int64 `json:"total"`     // Total traffic in bytes (Upload + Download)
}

// Marshal implements the json.Marshaler interface for ServerTraffic.
func (s *ServerTraffic) Marshal() ([]byte, error) {
	type Alias ServerTraffic
	return marshalLogJSON((*Alias)(s))
}

// Unmarshal implements the json.Unmarshaler interface for ServerTraffic.
func (s *ServerTraffic) Unmarshal(data []byte) error {
	type Alias ServerTraffic
	return json.Unmarshal(data, (*Alias)(s))
}

// ServerTrafficRank represents a server traffic rank entry.
type ServerTrafficRank struct {
	Rank map[uint8]ServerTraffic `json:"rank"` // Key is rank ,type is ServerTraffic
}

// Marshal implements the json.Marshaler interface for ServerTrafficRank.
func (s *ServerTrafficRank) Marshal() ([]byte, error) {
	type Alias ServerTrafficRank
	return marshalLogJSON((*Alias)(s))
}

// Unmarshal implements the json.Unmarshaler interface for ServerTrafficRank.
func (s *ServerTrafficRank) Unmarshal(data []byte) error {
	type Alias ServerTrafficRank
	return json.Unmarshal(data, (*Alias)(s))
}

// TrafficStat represents a daily traffic statistics log entry.
type TrafficStat struct {
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
	Total    int64 `json:"total"`
}

// Marshal implements the json.Marshaler interface for TrafficStat.
func (t *TrafficStat) Marshal() ([]byte, error) {
	type Alias TrafficStat
	return marshalLogJSON((*Alias)(t))
}

// Unmarshal implements the json.Unmarshaler interface for TrafficStat.
func (t *TrafficStat) Unmarshal(data []byte) error {
	type Alias TrafficStat
	return json.Unmarshal(data, (*Alias)(t))
}
