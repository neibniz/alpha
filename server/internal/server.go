package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/pkg/proc"
	"github.com/neibniz/alpha/server/pkg/trace"

	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/handler"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
)

type Service struct {
	mu         sync.Mutex
	server     *http.Server
	restarting bool
	svc        *svc.ServiceContext
}

func NewService(svc *svc.ServiceContext) *Service {
	service := &Service{
		svc: svc,
	}
	if svc != nil {
		svc.Restart = service.Restart
	}
	return service
}

func initServer(svc *svc.ServiceContext) *gin.Engine {

	// start init system config
	initialize.StartInitSystemConfig(svc)
	if err := setupGinLogWriters(svc.Config.Logger); err != nil {
		panic(err)
	}
	// init gin server
	r := gin.New()
	r.ForwardedByClientIP = false
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(fmt.Sprintf("disable Gin proxy parsing: %v", err))
	}
	clientIPResolver, err := clientip.NewResolver(clientip.Options{
		TrustedProxies: svc.Config.HTTP.TrustedProxies,
	})
	if err != nil {
		panic(fmt.Sprintf("configure client IP resolver: %v", err))
	}
	// use cors middleware
	r.Use(middleware.ClientIPMiddleware(clientIPResolver), middleware.TraceMiddleware(svc), middleware.LoggerMiddleware(svc), middleware.CorsMiddleware(svc), gin.RecoveryWithWriter(gin.DefaultErrorWriter))

	// register handlers
	handler.RegisterHandlers(r, svc)
	// register subscribe handler
	handler.RegisterSubscribeHandlers(r, svc)
	// register telegram handler
	handler.RegisterTelegramHandlers(r, svc)
	// register notify handler
	handler.RegisterNotifyHandlers(r, svc)
	return r
}

func setupGinLogWriters(c logger.LogConf) error {
	if c.Mode != "file" && c.Mode != "volume" {
		return nil
	}
	if c.Path == "" {
		return fmt.Errorf("gin log path is empty")
	}
	if err := os.MkdirAll(c.Path, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(c.Path, "gin.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	gin.DefaultWriter = file
	gin.DefaultErrorWriter = file
	return nil
}

func (m *Service) Start() {
	if m.svc == nil {
		panic("config file path is nil")
	}

	// init service
	r := initServer(m.svc)
	// get server port
	port := m.svc.Config.Port
	host := m.svc.Config.Host
	// check gateway mode
	if report.IsGatewayMode() {
		// get free port
		freePort, err := report.ModulePort()
		if err != nil {
			logger.Errorf("get module port error: %s", err.Error())
			panic(err)
		}
		port = freePort
		host = "127.0.0.1"
		// register module
		err = report.RegisterModule(port)
		if err != nil {
			logger.Errorf("register module error: %s", err.Error())
			os.Exit(1)
		}
		logger.Infof("module registered on port %d", port)
	}

	serverAddr := fmt.Sprintf("%v:%d", host, port)
	server := &http.Server{
		Addr:              serverAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	m.mu.Lock()
	m.server = server
	m.restarting = false
	m.mu.Unlock()
	trace.StartAgent(trace.Config{
		Name:    "alpha",
		Sampler: 1.0,
		Batcher: "",
	})
	proc.AddShutdownListener(func() {
		trace.StopAgent()
	})
	logger.Infof("server start at %v", serverAddr)
	if m.svc.Config.TLS.Enable {
		if err := server.ListenAndServeTLS(m.svc.Config.TLS.CertFile, m.svc.Config.TLS.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("server start error: %s", err.Error())
		}
	} else {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("server start error: %s", err.Error())
		}
	}
	m.mu.Lock()
	if m.server == server {
		m.server = nil
	}
	m.mu.Unlock()
}

func (m *Service) Stop() {
	m.mu.Lock()
	server := m.server
	m.mu.Unlock()
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("server shutdown error: %s", err.Error())
	}
	logger.Info("server shutdown")
}

func (m *Service) Restart() error {
	m.mu.Lock()
	if m.restarting {
		m.mu.Unlock()
		return errors.New("server restart is already in progress")
	}
	server := m.server
	if server == nil {
		m.mu.Unlock()
		return errors.New("server is nil")
	}
	m.restarting = true
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		m.mu.Lock()
		m.restarting = false
		m.mu.Unlock()
		logger.Errorf("server shutdown error: %v", err.Error())
		return err
	}
	logger.Info("server shutdown")
	go m.Start()
	return nil
}
