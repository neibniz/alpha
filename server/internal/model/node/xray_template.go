package node

import "time"

const (
	XrayTemplateInbound  = "inbound"
	XrayTemplateOutbound = "outbound"
	XrayTemplateDNS      = "dns"
	XrayTemplateRouting  = "routing"
	XrayTemplateGeodata  = "geodata"
)

type XrayTemplate struct {
	Id                       int64     `gorm:"primary_key"`
	Name                     string    `gorm:"type:varchar(100);not null;default:'';comment:Template Name"`
	Type                     string    `gorm:"type:varchar(20);not null;default:'';index:idx_xray_template_type;comment:Template Type"`
	Config                   string    `gorm:"type:jsonb;not null;default:'{}';comment:Xray JSON Config"`
	ConfigTemplate           string    `gorm:"type:text;not null;default:'';comment:Xray JSON Config Template"`
	VariablesSchema          string    `gorm:"type:jsonb;not null;default:'{}';comment:Template Variables Schema"`
	DefaultVariables         string    `gorm:"type:jsonb;not null;default:'{}';comment:Default Variables"`
	SubscriptionMeta         string    `gorm:"type:jsonb;not null;default:'{}';comment:Subscription Mapping Metadata"`
	Description              string    `gorm:"type:varchar(255);not null;default:'';comment:Description"`
	Enabled                  *bool     `gorm:"type:boolean;not null;default:true;comment:Enabled"`
	CreatedAt                time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt                time.Time `gorm:"comment:Update Time"`
	BindingCount             int64     `gorm:"->;-:migration"`
	SubscriptionBindingCount int64     `gorm:"->;-:migration"`
}

func (x *XrayTemplate) TableName() string {
	return "xray_templates"
}

type ServerXrayTemplate struct {
	Id                    int64         `gorm:"primary_key"`
	ServerId              int64         `gorm:"not null;default:0;uniqueIndex:idx_server_xray_template;comment:Server ID"`
	Server                *Server       `gorm:"foreignKey:ServerId;references:Id"`
	TemplateId            int64         `gorm:"not null;default:0;uniqueIndex:idx_server_xray_template;comment:Template ID"`
	Template              *XrayTemplate `gorm:"foreignKey:TemplateId;references:Id"`
	Alias                 string        `gorm:"type:varchar(64);not null;default:'';comment:Binding Alias"`
	Variables             string        `gorm:"type:jsonb;not null;default:'{}';comment:Binding Variables"`
	SubscriptionEnabled   *bool         `gorm:"type:boolean;not null;default:true;comment:Include In Subscription"`
	SubscriptionName      string        `gorm:"type:varchar(255);not null;default:'';comment:Subscription Node Name Template"`
	SubscriptionVariables string        `gorm:"type:jsonb;not null;default:'{}';comment:Subscription Variables"`
	Sort                  int           `gorm:"not null;default:0;comment:Sort"`
	Enabled               *bool         `gorm:"type:boolean;not null;default:true;comment:Enabled"`
	CreatedAt             time.Time     `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt             time.Time     `gorm:"comment:Update Time"`
}

func (s *ServerXrayTemplate) TableName() string {
	return "server_xray_templates"
}
