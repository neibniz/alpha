package server

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type ServerPushStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewServerPushStatusLogic Push server status
func NewServerPushStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ServerPushStatusLogic {
	return &ServerPushStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ServerPushStatusLogic) ServerPushStatus(req *types.ServerPushStatusRequest) error {
	exists, err := l.svcCtx.NodeModel.ServerExists(l.ctx, req.ServerId)
	if err != nil || !exists {
		l.Errorw("[ServerPushStatus] FindOne error", logger.Field("error", err))
		return errors.New("server not found")
	}
	if l.svcCtx.NodeRealtime != nil {
		l.svcCtx.NodeRealtime.Update(realtime.NodeStatus{
			ServerID:                  req.ServerId,
			CPU:                       req.Cpu,
			Mem:                       req.Mem,
			Disk:                      req.Disk,
			Load1:                     req.Load1,
			Load5:                     req.Load5,
			Load15:                    req.Load15,
			Uptime:                    req.Uptime,
			XrayRunning:               req.XrayRunning,
			XrayVersion:               req.XrayVersion,
			Connections:               req.Connections,
			SystemConnections:         req.SystemConnections,
			SystemInboundConnections:  req.SystemInboundConnections,
			SystemOutboundConnections: req.SystemOutboundConnections,
			XrayConnections:           req.XrayConnections,
			XrayInboundConnections:    req.XrayInboundConnections,
			XrayOutboundConnections:   req.XrayOutboundConnections,
			OnlineUsers:               req.OnlineUsers,
			NetRxBps:                  req.NetRxBps,
			NetTxBps:                  req.NetTxBps,
			NetRxBytes:                req.NetRxBytes,
			NetTxBytes:                req.NetTxBytes,
			XrayRxBps:                 req.XrayRxBps,
			XrayTxBps:                 req.XrayTxBps,
			XrayRxBytes:               req.XrayRxBytes,
			XrayTxBytes:               req.XrayTxBytes,
			XrayStatsError:            req.XrayStatsError,
			UpdatedAt:                 req.UpdatedAt,
			MemoryUsed:                req.MemoryUsed,
			MemoryTotal:               req.MemoryTotal,
			DiskUsed:                  req.DiskUsed,
			DiskTotal:                 req.DiskTotal,
			AgentVersion:              req.AgentVersion,
			ConfigVersion:             req.ConfigVersion,
			ConfigSyncStatus:          req.ConfigSyncStatus,
			ConfigApplyStatus:         req.ConfigApplyStatus,
			LastConfigHash:            req.LastConfigHash,
			RunningConfigHash:         req.RunningConfigHash,
			PendingConfigHash:         req.PendingConfigHash,
			LastSyncAt:                req.LastSyncAt,
			LastApplyAt:               req.LastApplyAt,
			LastConfigError:           req.LastConfigError,
			LastApplyError:            req.LastApplyError,
		})
	}

	return nil
}
