package server

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

type ServerPushOnlineUsersLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewServerPushOnlineUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ServerPushOnlineUsersLogic {
	return &ServerPushOnlineUsersLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ServerPushOnlineUsersLogic) ServerPushOnlineUsers(req *types.ServerPushOnlineUsersRequest) error {
	if req.ServerId <= 0 {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "server_id is required")
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != "delta" && mode != "full" {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "online user mode must be delta or full")
	}
	exists, err := l.svcCtx.NodeModel.ServerExists(l.ctx, req.ServerId)
	if err != nil || !exists {
		l.Errorw("[ServerPushOnlineUsers] FindOne error", logger.Field("error", err), logger.Field("server_id", req.ServerId))
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "server not found")
	}

	count, err := l.svcCtx.NodeModel.UpdateOnlineUsers(l.ctx, req.ServerId, node.OnlineUsersUpdate{
		Mode:       mode,
		InstanceID: strings.TrimSpace(req.InstanceID),
		Seq:        req.Seq,
		Active:     req.Active,
		Inactive:   req.Inactive,
		TTLSeconds: req.TTLSeconds,
	})
	if err != nil {
		l.Errorw("[ServerPushOnlineUsers] update online users error", logger.Field("error", err), logger.Field("server_id", req.ServerId))
		return xerr.NewErrMsg("update online users failed")
	}
	if l.svcCtx.NodeRealtime != nil {
		l.svcCtx.NodeRealtime.SetOnlineUsers(req.ServerId, count)
	}
	return nil
}
