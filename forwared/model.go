package main

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
)

// 转发规则
type ForwaredItem struct {
	RuleID     string `json:"rule_id"`
	ListenIP   string `json:"listen_ip,omitempty"`
	LocalPort  uint   `json:"listen_port"`
	TargetHost string `json:"target_host"`
	TargetPort uint   `json:"target_port"`
	NodeID     int64  `json:"node_id,omitempty"`
	NodeName   string `json:"node_name,omitempty"`
	ServerID   int64  `json:"server_id,omitempty"`
	ServerName string `json:"server_name,omitempty"`
	Inbound    string `json:"inbound,omitempty"`
}

// 转发表
type ForwaredTable struct {
	InstanceID    string         `json:"instance_id"`
	IP            string         `json:"public_ip"`
	Version       string         `json:"version"`
	Hash          string         `json:"hash"`
	GeneratedAt   int64          `json:"generated_at"`
	ForwaredItems []ForwaredItem `json:"rules"`
	Warnings      []string       `json:"warnings"`
}

// fetch时的响应
type ForwaredTableResponse struct {
	Code  uint            `json:"code"`
	Error string          `json:"error"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data"`
}

func (r ForwaredTableResponse) Message() string {
	if r.Error != "" {
		return r.Error
	}
	return r.Msg
}

// 转发配置
type ForwarderConfig struct {
	cfg      *Config      // 配置
	publicIP atomic.Value // 公网ip

	mu                         sync.Mutex      // 互斥锁
	forwaredTable              *ForwaredTable  // 转发表ForwaredTable
	fetchIPInterval            uint            // 获取IP的间隔 秒
	fetchForwaredTableInterval uint            // 获取转发配置的间隔 秒
	fm                         *ForwardManager //
	ctx                        context.Context
	cancel                     context.CancelFunc
	agent                      *Agent
	desiredVersion             string
	desiredHash                string
	appliedVersion             string
	appliedHash                string
	applyStatus                string
	applyError                 string
}
