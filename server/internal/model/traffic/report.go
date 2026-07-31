package traffic

import (
	"time"

	"github.com/google/uuid"
)

type ReportReceipt struct {
	ReportId        uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	ServerId        int64     `gorm:"primaryKey;not null"`
	SampledAt       time.Time `gorm:"not null"`
	IntervalSeconds int32     `gorm:"not null"`
	Sequence        int64     `gorm:"not null;default:0"`
	ProcessedAt     time.Time `gorm:"<-:create;not null"`
}

func (ReportReceipt) TableName() string { return "traffic_report_receipt" }
