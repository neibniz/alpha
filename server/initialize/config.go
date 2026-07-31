package initialize

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/pkg/logger"
	"gorm.io/driver/postgres"

	"github.com/neibniz/alpha/server/initialize/migrate"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/conf"
	"github.com/neibniz/alpha/server/pkg/orm"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

//go:embed templates/*.html
var templateFS embed.FS

var initStatus = make(chan bool)
var configPath string

func Config(path string) (chan bool, *http.Server) {
	// Set the configuration file path
	configPath = path
	// Create a new Gin instance
	r := gin.Default()
	// get server port
	port := 8080
	host := "127.0.0.1"

	// check gateway mode
	if report.IsGatewayMode() {
		// get free port
		freePort, err := report.ModulePort()
		if err != nil {
			logger.Errorf("get module port error: %s", err.Error())
			panic(err)
		}
		port = freePort
		// register module
		err = report.RegisterModule(port)
		if err != nil {
			logger.Errorf("register module error: %s", err.Error())
			panic(err)
		}
		logger.Infof("module registered on port %d", port)
	}
	// Create a new HTTP server
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	// Load templates
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/init", handleInit)
	r.POST("/init/config", handleInitConfig)
	r.POST("/init/mysql/test", HandlePostgresTest)
	r.POST("/init/redis/test", HandleRedisTest)
	// Handle 404
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/init")
	})

	go func(server *http.Server) {
		// Start the server
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}(server)

	return initStatus, server
}

func handleInit(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}
func handleInitConfig(c *gin.Context) {
	// Load configuration file

	var cfg config.File
	conf.MustLoad(configPath, &cfg)
	var request struct {
		AdminEmail    string `json:"adminEmail"`
		AdminPassword string `json:"adminPassword"`

		MysqlHost     string `json:"mysqlHost"`
		MysqlPort     string `json:"mysqlPort"`
		MysqlDatabase string `json:"mysqlDatabase"`
		MysqlUser     string `json:"mysqlUser"`
		MysqlPassword string `json:"mysqlPassword"`

		RedisHost     string `json:"redisHost"`
		RedisPort     string `json:"redisPort"`
		RedisPassword string `json:"redisPassword"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid request",
			"data": nil,
		})
		c.Abort()
		return
	}
	if err := tool.ValidatePassword(request.AdminPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Administrator password does not meet policy",
			"data": nil,
		})
		c.Abort()
		return
	}
	cfg.Debug = false
	// jwt secret
	cfg.JwtAuth.AccessSecret = uuid.New().String()
	cfg.Postgres.Addr = fmt.Sprintf("%s:%s", request.MysqlHost, request.MysqlPort)
	cfg.Postgres.Dbname = request.MysqlDatabase
	cfg.Postgres.Username = request.MysqlUser
	cfg.Postgres.Password = request.MysqlPassword
	// redis
	cfg.Redis.Host = fmt.Sprintf("%s:%s", request.RedisHost, request.RedisPort)
	cfg.Redis.Pass = request.RedisPassword

	// save config
	fileData, err := yaml.Marshal(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Configuration initialization failed",
			"data": nil,
		})
		c.Abort()
		return
	}

	pg := orm.Postgres{
		Config: orm.Config{
			Addr:          fmt.Sprintf("%s:%s", request.MysqlHost, request.MysqlPort),
			Username:      request.MysqlUser,
			Password:      request.MysqlPassword,
			Dbname:        request.MysqlDatabase,
			Config:        "sslmode=disable&TimeZone=Asia/Shanghai",
			MaxIdleConns:  10,
			MaxOpenConns:  10,
			SlowThreshold: 1000,
		},
	}
	db, err := orm.ConnectPostgres(pg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "PostgreSQL connection failed",
			"data": nil,
		})
		c.Abort()
		return
	}
	// migrate database
	if err = migrate.Migrate(pg.Dsn()).Up(); err != nil {
		if errors.Is(err, migrate.NoChange) {
			logger.Info("[Init Postgres] database not change")
		} else {
			logger.Errorf("[Init Postgres] Migrate failed: %v", err.Error())
			c.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  "Database migration failed",
				"data": nil,
			})
			c.Abort()
			return
		}
	}
	if err = migrate.SyncIdentitySequences(db); err != nil {
		logger.Errorf("[Init Postgres] Sync identity sequences failed: %v", err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "Database sequence sync failed",
			"data": nil,
		})
		c.Abort()
		return
	}

	// create admin user
	if err = migrate.CreateAdminUser(request.AdminEmail, request.AdminPassword, db); err != nil {
		logger.Errorf("[Init Postgres] Create admin user failed: %v", err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "Admin user creation failed",
			"data": nil,
		})
		c.Abort()
		return
	}

	// write to file
	if err = os.WriteFile(configPath, fileData, 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Configuration initialization failed",
			"data": nil,
		})
		c.Abort()
		return
	}
	if err = os.Chmod(configPath, 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Configuration permission update failed",
			"data": nil,
		})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "Configuration initialized",
		"status": true,
	})
	initStatus <- true
}

func HandlePostgresTest(c *gin.Context) {
	var request struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid request",
			"data": nil,
		})
		c.Abort()
		return
	}
	pg := orm.Postgres{Config: orm.Config{
		Addr:     fmt.Sprintf("%s:%s", request.Host, request.Port),
		Username: request.User,
		Password: request.Password,
		Dbname:   request.Database,
		Config:   "sslmode=disable&TimeZone=Asia/Shanghai",
	}}
	dsn := pg.Dsn()
	var status = true
	var message string
	var tx *sql.DB
	var tables []string
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Errorf("connect postgres failed, err: %v\n", err.Error())
		status = false
		message = "PostgreSQL connection failed"
		goto result
	}
	tx, _ = db.DB()
	if err := tx.Ping(); err != nil {
		logger.Errorf("ping postgres failed, err: %v\n", err.Error())
		status = false
		message = "PostgreSQL connection failed"
	}

	tables, err = db.Migrator().GetTables()
	if err != nil {
		logger.Errorf("database table check failed, err: %v\n", err.Error())
		status = false
		message = "Database table check failed"
		goto result
	}
	if len(tables) > 0 {
		status = false
		message = "The database contains existing data. Please clear it before proceeding with the installation."
		goto result
	}

result:
	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    message,
		"status": status,
	})
}

func HandleRedisTest(c *gin.Context) {
	var request struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid request",
			"data": nil,
		})
		c.Abort()
		return
	}
	if err := tool.RedisWritableCheck(fmt.Sprintf("%s:%s", request.Host, request.Port), request.Password, 0); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":   200,
			"msg":    nil,
			"status": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    nil,
		"status": true,
	})
}
