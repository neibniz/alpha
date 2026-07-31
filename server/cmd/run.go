package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/conf"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/orm"
	"github.com/neibniz/alpha/server/pkg/service"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/queue"
	"github.com/neibniz/alpha/server/scheduler"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	startCmd.Flags().StringVar(&startConfigPath, "config", "etc/alpha.yaml", "alpha.yaml directory to read from")
}

var (
	startConfigPath string
)

var startCmd = &cobra.Command{
	Use:   "run",
	Short: "start Alpha",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Infof("[Alpha version] v%s(%s)", constant.Version, constant.BuildTime)
		run()
	},
}

func run() {
	services := getServers()
	defer services.Stop()
	go services.Start()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
}
func getServers() *service.Group {
	var c config.Config

	// check config file is exist
	if _, err := os.Stat(startConfigPath); os.IsNotExist(err) {
		// check directory is existed
		if _, err := os.Stat("etc"); os.IsNotExist(err) {
			logger.Errorf("Directory %s does not exist. Creating it...\n", "etc")
			if err = os.MkdirAll("etc", 0o700); err != nil {
				logger.Errorf("Please create the directory %s and place the configuration file %s in it.\n", "etc", startConfigPath)
				os.Exit(1)
			}
		}
		// create new config file
		file, err := os.OpenFile(startConfigPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- path is an explicit operator CLI setting
		if err != nil {
			logger.Errorf("Please create the configuration file %s in the directory %s.\n", startConfigPath, "etc")
			panic(fmt.Sprintf("Please create the configuration file %s in the directory %s.\n", startConfigPath, "etc"))
		}
		if err := file.Close(); err != nil {
			panic(fmt.Sprintf("Failed to close configuration file %s: %v", startConfigPath, err))
		}
	}
	// check config file is empty, if empty, start init web server
	if initConfig(&c) {
		status, server := initialize.Config(startConfigPath)
		<-status
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Errorf("Init Server Shutdown: %s", err.Error())
		}
	}
	mustLoadRuntimeConfig(startConfigPath, &c)
	if err := config.ValidateRuntimeSecurity(c); err != nil {
		panic(fmt.Sprintf("validate runtime security: %v", err))
	}
	if !c.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	// init logger
	if err := logger.SetUp(c.Logger); err != nil {
		logger.Errorf("Logger setup failed: %v", err.Error())
	}

	// init service context
	ctx := svc.NewServiceContext(c)
	services := service.NewServiceGroup()
	services.Add(internal.NewService(ctx))
	services.Add(queue.NewService(ctx))
	services.Add(scheduler.NewService(ctx))
	return services
}

func initConfig(c *config.Config) bool {
	// load config
	mustLoadRuntimeConfig(startConfigPath, c)
	//  check custom config
	if startConfigPath != "etc/alpha.yaml" && c.Postgres.Addr == "" {
		return true
	}
	// check access secret
	if c.JwtAuth.AccessSecret == "" && startConfigPath == "etc/alpha.yaml" {
		c.JwtAuth.AccessSecret = uuid.New().String()
		// Get environment variables
		dsn := os.Getenv("ALPHA_DB")
		if dsn == "" {
			return true
		}
		cfg := orm.ParseDSN(dsn)
		if cfg == nil {
			return true
		} else {
			c.Postgres = *cfg
		}

		// Get environment variables
		uri := os.Getenv("ALPHA_REDIS")
		if uri == "" {
			return true
		}
		addr, pass, db, err := tool.ParseRedisURI(uri)
		if err != nil {
			return true
		} else {
			c.Redis.Host = addr
			c.Redis.Pass = pass
			c.Redis.DB = db
		}
		// save yaml file
		newConfig := config.File{
			Host:     c.Host,
			Port:     c.Port,
			Debug:    c.Debug,
			JwtAuth:  c.JwtAuth,
			Logger:   c.Logger,
			Postgres: c.Postgres,
			Redis:    c.Redis,
		}
		fileData, err := yaml.Marshal(newConfig)
		if err != nil {
			panic(err.Error())
		}
		// write to file
		if err := os.WriteFile(startConfigPath, fileData, 0o600); err != nil {
			panic(err.Error())
		}
		if err := os.Chmod(startConfigPath, 0o600); err != nil {
			panic(err.Error())
		}
	}
	return false
}

func mustLoadRuntimeConfig(path string, cfg *config.Config) {
	conf.MustLoad(path, cfg)
	if err := config.ApplyEnvironment(cfg); err != nil {
		panic(fmt.Sprintf("load runtime secrets: %v", err))
	}
}
