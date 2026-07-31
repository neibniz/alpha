package user

import "time"

const (
	SubscribePresetModeCustom  = "custom"
	SubscribePresetModeProfile = "profile"
)

type SubscribePreset struct {
	Id              int64     `gorm:"primary_key"`
	UserId          int64     `gorm:"not null;default:0;index:idx_user_subscribe_preset_user;comment:User ID"`
	UserSubscribeId int64     `gorm:"not null;default:0;index:idx_user_subscribe_preset_subscribe;comment:User Subscribe ID"`
	Name            string    `gorm:"type:varchar(100);not null;default:'';comment:Preset Name"`
	PresetKey       string    `gorm:"type:varchar(64);not null;default:'';uniqueIndex:idx_user_subscribe_preset_key;comment:Preset Key"`
	Mode            string    `gorm:"type:varchar(20);not null;default:'custom';comment:Preset Mode"`
	ProfileKey      string    `gorm:"type:varchar(100);not null;default:'';comment:Profile Key"`
	Items           string    `gorm:"type:jsonb;not null;default:'[]';comment:Selected Subscription Items"`
	Sort            int       `gorm:"not null;default:0;comment:Sort"`
	Enabled         *bool     `gorm:"type:boolean;not null;default:true;comment:Enabled"`
	IsDefault       bool      `gorm:"type:boolean;not null;default:false;comment:Default Preset"`
	CreatedAt       time.Time `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt       time.Time `gorm:"comment:Update Time"`
}

func (s *SubscribePreset) TableName() string {
	return "user_subscribe_presets"
}
