package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/neibniz/alpha/forwared/logger"
	"os"
	"os/signal"
	"syscall"
)

var geoIPDatabase *GeoIPDatabase

func main() {

	configPath := flag.String("config", "", "optional config path; environment variables override YAML")
	flag.Parse()
	// 加载配置文件
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	filepath := cfg.LogFile
	// 初始化日志记录器
	if err := logger.Init(filepath); err != nil {
		fmt.Println("Failed to initialize logger:", err)
		os.Exit(2)
	}

	// 使用 logger 记录日志
	logger.Logger.Info("Application started")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	geoIPDatabase = NewGeoIPDatabase(cfg)
	if err := geoIPDatabase.Prepare(ctx); err != nil {
		logger.Logger.Error("IP2Location database is unavailable", "error", err)
		os.Exit(3)
	}
	defer geoIPDatabase.Close()
	updatesDone := make(chan struct{})
	go func() { defer close(updatesDone); geoIPDatabase.Run(ctx) }()

	agent, err := NewAgent(cfg)
	if err != nil {
		logger.Logger.Error("Initialize agent failed", "error", err)
		os.Exit(4)
	}
	if err := agent.Start(); err != nil {
		logger.Logger.Error("Register forwarder failed", "error", err)
		os.Exit(5)
	}
	fc := NewForwarderConfig(cfg, agent)
	fc.fetch()
	// 阻塞主goroutine，等待信号处理完成
	<-ctx.Done()
	cancel()
	fc.stop()
	agent.Stop()
	<-updatesDone
}
