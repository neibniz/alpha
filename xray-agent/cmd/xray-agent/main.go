package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/logging"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
	"github.com/neibniz/alpha/xray-agent/internal/supervisor"
	"github.com/neibniz/alpha/xray-agent/internal/syncer"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	closeLog, err := logging.Setup(cfg.Log.AgentPath)
	if err != nil {
		log.Fatalf("setup log: %v", err)
	}
	defer func() {
		if err := closeLog(); err != nil {
			log.Printf("close log file failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := panel.NewClient(cfg.Panel)
	xraySupervisor := supervisor.New(cfg.Xray, cfg.Log)
	runner := syncer.New(cfg, client, xraySupervisor)

	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("agent stopped: %v", err)
	}
}
