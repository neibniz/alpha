package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryServerXrayTemplateListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryServerXrayTemplateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerXrayTemplateListLogic {
	return &QueryServerXrayTemplateListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerXrayTemplateListLogic) QueryServerXrayTemplateList(req *types.QueryServerXrayTemplateListRequest) (*types.QueryServerXrayTemplateListResponse, error) {
	if req.ServerId == 0 {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "server id is required"), "server id is required")
	}
	includeTemplate := boolPtrValue(req.IncludeTemplate)
	includeStatus := boolPtrValue(req.IncludeStatus)
	includeDesiredConfig := boolPtrValue(req.IncludeDesiredConfig)
	templatePreload := node.XrayTemplatePreloadNone
	if includeTemplate {
		templatePreload = node.XrayTemplatePreloadSummary
	}
	if includeDesiredConfig {
		templatePreload = node.XrayTemplatePreloadRuntime
	}

	list, err := l.svcCtx.NodeModel.QueryServerXrayTemplatesWithOptions(l.ctx, req.ServerId, node.QueryServerXrayTemplateOptions{
		TemplatePreload: templatePreload,
	})
	if err != nil {
		l.Errorf("[QueryServerXrayTemplateList] query error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server xray templates error: %v", err)
	}
	resp := &types.QueryServerXrayTemplateListResponse{}
	if includeStatus {
		status := l.serverRealtimeStatus(req.ServerId)
		resp.Status = &status
	}
	rows, err := serverXrayTemplateRowsToTypes(list, includeTemplate)
	if err != nil {
		return nil, err
	}
	resp.List = rows
	if includeDesiredConfig {
		srv, err := l.svcCtx.NodeModel.FindOneServerForRuntime(l.ctx, req.ServerId)
		if err != nil {
			l.Errorf("[QueryServerXrayTemplateList] query server error: %v", err.Error())
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server error: %v", err)
		}
		if hash, err := desiredXrayConfigHash(srv, enabledXrayTemplateBindings(list)); err != nil {
			resp.DesiredConfigError = err.Error()
		} else {
			resp.DesiredConfigHash = hash
		}
	}
	return resp, nil
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func enabledXrayTemplateBindings(list []*node.ServerXrayTemplate) []*node.ServerXrayTemplate {
	collector := viewutil.NewRowsCollector[*node.ServerXrayTemplate](len(list))
	for _, item := range list {
		if item == nil || item.Template == nil {
			continue
		}
		if item.Enabled != nil && !*item.Enabled {
			continue
		}
		if item.Template.Enabled != nil && !*item.Template.Enabled {
			continue
		}
		collector.Add(item)
	}
	return collector.Rows()
}

func desiredXrayConfigHash(srv *node.Server, bindings []*node.ServerXrayTemplate) (string, error) {
	raw, err := renderValidatedRawXrayConfigBundle(srv, bindings)
	if err != nil {
		return "", err
	}
	payload := struct {
		Inbounds  []json.RawMessage `json:"inbounds"`
		Outbounds []json.RawMessage `json:"outbounds"`
		DNS       json.RawMessage   `json:"dns,omitempty"`
		Routing   json.RawMessage   `json:"routing,omitempty"`
		Geodata   json.RawMessage   `json:"geodata,omitempty"`
	}{
		Inbounds:  raw.Inbounds,
		Outbounds: raw.Outbounds,
		DNS:       raw.DNS,
		Routing:   raw.Routing,
		Geodata:   raw.Geodata,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (l *QueryServerXrayTemplateListLogic) serverRealtimeStatus(serverID int64) types.ServerStatus {
	status := types.ServerStatus{Status: "offline"}
	if l.svcCtx.NodeRealtime == nil {
		return status
	}
	realtimeStatus, ok := l.svcCtx.NodeRealtime.Get(serverID)
	if !ok {
		return status
	}
	status.Cpu = realtimeStatus.CPU
	status.Mem = realtimeStatus.Mem
	status.Disk = realtimeStatus.Disk
	status.Status = realtimeStatus.Status
	status.XrayRunning = realtimeStatus.XrayRunning
	status.XrayVersion = realtimeStatus.XrayVersion
	status.UpdatedAt = realtimeStatus.UpdatedAt
	status.LastSeen = realtimeStatus.LastSeen
	status.AgentVersion = realtimeStatus.AgentVersion
	status.ConfigVersion = realtimeStatus.ConfigVersion
	status.ConfigSyncStatus = realtimeStatus.ConfigSyncStatus
	status.ConfigApplyStatus = realtimeStatus.ConfigApplyStatus
	status.LastConfigHash = realtimeStatus.LastConfigHash
	status.RunningConfigHash = realtimeStatus.RunningConfigHash
	status.PendingConfigHash = realtimeStatus.PendingConfigHash
	status.LastSyncAt = realtimeStatus.LastSyncAt
	status.LastApplyAt = realtimeStatus.LastApplyAt
	status.LastConfigError = realtimeStatus.LastConfigError
	status.LastApplyError = realtimeStatus.LastApplyError
	return status
}
