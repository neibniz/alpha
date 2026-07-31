package main

import (
	"flag"
	"fmt"
	"github.com/neibniz/alpha/forwared/logger"
	"os"
	"os/signal"
	"syscall"

	ip2location "github.com/ip2location/ip2location-go/v9"
)

var IP2Location_db *ip2location.DB

func main() {

	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	// 注册退出信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	configPath := flag.String("config", "forwared.yaml", "config path")
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

	// 读取ip数据库
	IP2Location_db, err = ip2location.OpenDB(cfg.GeoIPFile)
	if err != nil {
		fmt.Println("Failed to load IP2Location database:", err)
		os.Exit(3)
	}

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
	<-sigChan
	fc.stop()
	agent.Stop()
}
