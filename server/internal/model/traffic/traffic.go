package traffic

import "time"

//goland:noinspection GoNameStartsWithPackageName
type TrafficLog struct {
	Id          int64     `gorm:"primaryKey"`
	ServerId    int64     `gorm:"index:idx_server_id;not null;comment:Server ID"`
	UserId      int64     `gorm:"index:idx_user_id;not null;comment:User ID"`
	SubscribeId int64     `gorm:"index:idx_subscribe_id;not null;comment:Subscription ID"`
	Download    int64     `gorm:"default:0;comment:Download Traffic"`
	Upload      int64     `gorm:"default:0;comment:Upload Traffic"`
	Timestamp   time.Time `gorm:"default:CURRENT_TIMESTAMP;not null;comment:Traffic Log Time"`
}

type DailyUserTraffic struct {
	Id          int64     `gorm:"primaryKey;autoIncrement"`
	Date        string    `gorm:"type:varchar(20);index:idx_daily_user_traffic_date_total,priority:1;index:idx_daily_user_traffic_user_sub_date,priority:3;not null;comment:Traffic Date"`
	UserId      int64     `gorm:"index:idx_daily_user_traffic_user_sub_date,priority:1;not null;comment:User ID"`
	SubscribeId int64     `gorm:"index:idx_daily_user_traffic_user_sub_date,priority:2;not null;comment:Subscription ID"`
	Download    int64     `gorm:"default:0;comment:Download Traffic"`
	Upload      int64     `gorm:"default:0;comment:Upload Traffic"`
	Total       int64     `gorm:"index:idx_daily_user_traffic_date_total,priority:2;default:0;comment:Total Traffic"`
	CreatedAt   time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt   time.Time `gorm:"comment:Update Time"`
}

type DailyServerTraffic struct {
	Id        int64     `gorm:"primaryKey;autoIncrement"`
	Date      string    `gorm:"type:varchar(20);index:idx_daily_server_traffic_date_total,priority:1;index:idx_daily_server_traffic_server_date,priority:2;not null;comment:Traffic Date"`
	ServerId  int64     `gorm:"index:idx_daily_server_traffic_server_date,priority:1;not null;comment:Server ID"`
	Download  int64     `gorm:"default:0;comment:Download Traffic"`
	Upload    int64     `gorm:"default:0;comment:Upload Traffic"`
	Total     int64     `gorm:"index:idx_daily_server_traffic_date_total,priority:2;default:0;comment:Total Traffic"`
	CreatedAt time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt time.Time `gorm:"comment:Update Time"`
}

type DailyTrafficStat struct {
	Id        int64     `gorm:"primaryKey;autoIncrement"`
	Date      string    `gorm:"uniqueIndex:idx_daily_traffic_stat_date;type:varchar(20);not null;comment:Traffic Date"`
	Download  int64     `gorm:"default:0;comment:Download Traffic"`
	Upload    int64     `gorm:"default:0;comment:Upload Traffic"`
	Total     int64     `gorm:"default:0;comment:Total Traffic"`
	CreatedAt time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt time.Time `gorm:"comment:Update Time"`
}

type TotalTraffic struct {
	Download int64
	Upload   int64
}

type ServerTrafficRanking struct {
	ServerId int64
	Download int64
	Upload   int64
	Total    int64
}

type UserTrafficRanking struct {
	UserId      int64
	SubscribeId int64
	Download    int64
	Upload      int64
	Total       int64
}

func (TrafficLog) TableName() string {
	return "traffic_log"
}

func (DailyUserTraffic) TableName() string {
	return "daily_user_traffic"
}

func (DailyServerTraffic) TableName() string {
	return "daily_server_traffic"
}

func (DailyTrafficStat) TableName() string {
	return "daily_traffic_stat"
}
