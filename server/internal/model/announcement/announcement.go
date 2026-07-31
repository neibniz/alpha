package announcement

import "time"

type Announcement struct {
	Id        int64     `gorm:"primaryKey"`
	Title     string    `gorm:"type:varchar(255);not null;default:'';comment:Title"`
	Content   string    `gorm:"type:text;comment:Content"`
	Show      *bool     `gorm:"type:boolean;not null;default:false;comment:Show"`
	Pinned    *bool     `gorm:"type:boolean;not null;default:false;comment:Pinned"`
	Popup     *bool     `gorm:"type:boolean;not null;default:false;comment:Popup"`
	CreatedAt time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt time.Time `gorm:"comment:Update Time"`
}

func (Announcement) TableName() string {
	return "announcement"
}
