package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/pkg/orm"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

func TestQueryCtxCoalescesConcurrentCacheMisses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	conn := NewConn(db, rds, WithExpiry(time.Minute))

	const callers = 32
	start := make(chan struct{})
	queryStarted := make(chan struct{})
	releaseQuery := make(chan struct{})
	var queryStartedOnce sync.Once
	var queryCount atomic.Int32
	query := func(_ *gorm.DB, v interface{}) error {
		queryCount.Add(1)
		queryStartedOnce.Do(func() { close(queryStarted) })
		<-releaseQuery
		*v.(*cacheTestItem) = cacheTestItem{Id: 1, Name: "shared"}
		return nil
	}

	results := make(chan cacheTestItem, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			<-start
			var got cacheTestItem
			if err := conn.QueryCtx(context.Background(), &got, "cache:test:shared", query); err != nil {
				errs <- err
				return
			}
			results <- got
		}()
	}
	close(start)
	<-queryStarted
	time.Sleep(50 * time.Millisecond)
	if got := queryCount.Load(); got != 1 {
		close(releaseQuery)
		wg.Wait()
		t.Fatalf("QueryCtx() concurrent query count = %d, want 1", got)
	}
	close(releaseQuery)
	wg.Wait()
	close(errs)
	close(results)

	for err := range errs {
		t.Errorf("QueryCtx() error = %v", err)
	}
	resultCount := 0
	for got := range results {
		resultCount++
		if got != (cacheTestItem{Id: 1, Name: "shared"}) {
			t.Errorf("QueryCtx() item = %+v, want shared item", got)
		}
	}
	if resultCount != callers {
		t.Fatalf("QueryCtx() result count = %d, want %d", resultCount, callers)
	}
}

type User struct {
	Id                    int64                 `gorm:"primarykey"`
	Email                 string                `gorm:"index:idx_email;type:varchar(100);unique;not null;comment:电子邮箱"`
	Password              string                `gorm:"type:varchar(100);comment:用户密码;not null"`
	Avatar                string                `gorm:"type:varchar(200);default:'';comment:用户头像"`
	Balance               int64                 `gorm:"default:0;comment:用户余额"`
	Telegram              int64                 `gorm:"default:null;comment:Telegram账号"`
	ReferCode             string                `gorm:"type:varchar(20);default:'';comment:推荐码"`
	RefererId             int64                 `gorm:"comment:推荐人ID"`
	Enable                bool                  `gorm:"default:true;not null;comment:账户是否可用"`
	IsAdmin               bool                  `gorm:"default:false;not null;comment:是否管理员"`
	ValidEmail            bool                  `gorm:"default:false;not null;comment:是否验证邮箱"`
	EnableEmailNotify     bool                  `gorm:"default:false;not null;comment:是否启用邮件通知"`
	EnableTelegramNotify  bool                  `gorm:"default:false;not null;comment:是否启用Telegram通知"`
	EnableBalanceNotify   bool                  `gorm:"default:false;not null;comment:是否启用余额变动通知"`
	EnableLoginNotify     bool                  `gorm:"default:false;not null;comment:是否启用登录通知"`
	EnableSubscribeNotify bool                  `gorm:"default:false;not null;comment:是否启用订阅通知"`
	EnableTradeNotify     bool                  `gorm:"default:false;not null;comment:是否启用交易通知"`
	CreatedAt             time.Time             `gorm:"<-:create;comment:创建时间"`
	UpdatedAt             time.Time             `gorm:"comment:更新时间"`
	DeletedAt             gorm.DeletedAt        `gorm:"default:null;comment:删除时间"`
	IsDel                 soft_delete.DeletedAt `gorm:"softDelete:flag,DeletedAtField:DeletedAt;comment:1:正常 0:删除"` // Use `1` `0` to identify
}

type cacheTestItem struct {
	Id   int64 `gorm:"primarykey"`
	Name string
}

func TestQueryCtxDropsCorruptCacheAndRefreshes(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&cacheTestItem{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&cacheTestItem{Id: 1, Name: "fresh"}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	conn := NewConn(db, rds, WithExpiry(time.Minute))
	key := "cache:test:item:1"
	if err := rds.Set(ctx, key, "{bad-json", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	var got cacheTestItem
	queryCount := 0
	if err := conn.QueryCtx(ctx, &got, key, func(conn *gorm.DB, v interface{}) error {
		queryCount++
		return conn.First(v, 1).Error
	}); err != nil {
		t.Fatal(err)
	}
	if got.Name != "fresh" {
		t.Fatalf("QueryCtx() item name = %q, want fresh", got.Name)
	}
	if queryCount != 1 {
		t.Fatalf("QueryCtx() query count = %d, want 1", queryCount)
	}

	var cached cacheTestItem
	if err := conn.GetCacheCtx(ctx, key, &cached); err != nil {
		t.Fatal(err)
	}
	if cached.Name != "fresh" {
		t.Fatalf("cached item name = %q, want fresh", cached.Name)
	}
}

func TestGormCacheCtx(t *testing.T) {
	t.Skipf("skip TestGormCacheCtx test")
	db, err := orm.ConnectPostgres(orm.Postgres{
		Config: orm.Config{
			Addr:     "localhost:5432",
			Config:   "sslmode=disable&TimeZone=Asia/Shanghai",
			Dbname:   "alpha",
			Username: "postgres",
			Password: "postgres",
		},
	})
	if err != nil {
		t.Error(err)
	}
	rds := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	conn := NewConn(db, rds)
	var u User
	key := "user:id"
	err = conn.QueryCtx(context.Background(), &u, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Where("id = ?", 1).First(v).Error
	})
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("get cache success %+v", u)
}
