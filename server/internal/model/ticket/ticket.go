package ticket

import (
	"database/sql/driver"
	"time"

	"github.com/lib/pq"
)

const (
	Pending   = 1 // Pending  # Pending follow up
	Waiting   = 2 // Waiting  # Waiting for user response
	Processed = 3 // Processed
	Closed    = 4 // Closed
)

type ImageGallery pq.StringArray

func (gallery ImageGallery) Value() (driver.Value, error) {
	if gallery == nil {
		return "{}", nil
	}
	return pq.StringArray(gallery).Value()
}

func (gallery *ImageGallery) Scan(src any) error {
	return (*pq.StringArray)(gallery).Scan(src)
}

type Ticket struct {
	Id          int64        `gorm:"primaryKey"`
	Title       string       `gorm:"type:varchar(255);not null;default:'';comment:Title"`
	Description string       `gorm:"type:text;comment:Description"`
	Images      ImageGallery `gorm:"type:text[];not null;default:'{}';comment:Image Gallery"`
	UserId      int64        `gorm:"type:bigint;not null;default:0;comment:UserId"`
	Status      uint8        `gorm:"type:smallint;not null;default:1;comment:Status"`
	CreatedAt   time.Time    `gorm:"<-:create;comment:Create Time"`
	UpdatedAt   time.Time    `gorm:"comment:Update Time"`
}

func (Ticket) TableName() string {
	return "ticket"
}

type Follow struct {
	Id        int64        `gorm:"primaryKey"`
	TicketId  int64        `gorm:"type:bigint;not null;default:0;comment:TicketId"`
	From      string       `gorm:"type:varchar(255);not null;default:'';comment:From"`
	Type      uint8        `gorm:"type:smallint;not null;default:1;comment:Type: 1 text, 2 image"`
	Content   string       `gorm:"type:text;comment:Content"`
	Images    ImageGallery `gorm:"type:text[];not null;default:'{}';comment:Image Gallery"`
	CreatedAt time.Time    `gorm:"<-:create;comment:Create Time"`
}

func (Follow) TableName() string {
	return "ticket_follow"
}
