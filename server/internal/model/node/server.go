package node

import (
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"
	"gorm.io/gorm"
)

type Server struct {
	Id      int64  `gorm:"primary_key"`
	Name    string `gorm:"type:varchar(100);not null;default:'';comment:Server Name"`
	Country string `gorm:"type:varchar(128);not null;default:'';comment:Country"`
	City    string `gorm:"type:varchar(128);not null;default:'';comment:City"`
	//Ratio          float32    `gorm:"type:DECIMAL(4,2);not null;default:0;comment:Traffic Ratio"`
	Address        string     `gorm:"type:varchar(100);not null;default:'';comment:Server Address"`
	Sort           int        `gorm:"type:int;not null;default:0;comment:Sort"`
	LastReportedAt *time.Time `gorm:"comment:Last Reported Time"`
	CreatedAt      time.Time  `gorm:"<-:create;comment:Creation Time"`
	UpdatedAt      time.Time  `gorm:"comment:Update Time"`
	NodeCount      int64      `gorm:"->;-:migration"`
	TemplateCount  int64      `gorm:"->;-:migration"`
}

func (*Server) TableName() string {
	return "servers"
}

func (m *Server) BeforeCreate(tx *gorm.DB) error {
	if m.Sort == 0 {
		var maxSort int
		if err := tx.Model(&Server{}).Select("COALESCE(MAX(sort), 0)").Scan(&maxSort).Error; err != nil {
			return err
		}
		m.Sort = maxSort + 1
	}
	return nil
}

func (m *Server) BeforeDelete(tx *gorm.DB) error {
	if err := tx.Exec("UPDATE servers SET sort = sort - 1 WHERE sort > ?", m.Sort).Error; err != nil {
		return err
	}
	return nil
}

func (m *Server) BeforeUpdate(tx *gorm.DB) error {
	var count int64
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Model(&Server{}).
		Where("sort = ? AND id != ?", m.Sort, m.Id).Count(&count).Error; err != nil {
		return err
	}
	if count > 1 {
		// reorder sort
		if err := reorderSortWithServer(tx); err != nil {
			logger.Errorf("[Server] BeforeUpdate reorderSort error: %v", err.Error())
			return err
		}
		// get max sort
		var maxSort int
		if err := tx.Model(&Server{}).Select("MAX(sort)").Scan(&maxSort).Error; err != nil {
			return err
		}
		m.Sort = maxSort + 1
	}
	return nil
}

func reorderSortWithServer(tx *gorm.DB) error {
	var servers []sortRow
	if err := tx.Model(&Server{}).Select("id", "sort").Order("sort, id").Find(&servers).Error; err != nil {
		return err
	}
	for i, server := range servers {
		if server.Sort != i+1 {
			if err := tx.Exec("UPDATE servers SET sort = ? WHERE id = ?", i+1, server.Id).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
