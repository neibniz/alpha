package traffic

import (
	"time"

	"github.com/google/uuid"
)

type EventOutbox struct {
	Id              int64     `gorm:"primaryKey;autoIncrement"`
	EventId         uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	ServerId        int64     `gorm:"not null"`
	UserId          int64     `gorm:"not null"`
	UserSubscribeId int64     `gorm:"not null"`
	Download        int64     `gorm:"not null;default:0"`
	Upload          int64     `gorm:"not null;default:0"`
	EventTime       time.Time `gorm:"not null"`
	CreatedAt       time.Time `gorm:"<-:create;not null"`
}

func (EventOutbox) TableName() string { return "traffic_event_outbox" }
