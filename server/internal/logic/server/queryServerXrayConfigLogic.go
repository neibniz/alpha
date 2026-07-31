package server

import (
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryServerXrayConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryServerXrayConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerXrayConfigLogic {
	return &QueryServerXrayConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerXrayConfigLogic) QueryServerXrayConfig(req *types.QueryServerXrayConfigRequest) (*types.QueryServerXrayConfigResponse, error) {
	if req.ServerID == 0 {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "server id is required"), "server id is required")
	}
	srv, err := l.svcCtx.NodeModel.FindOneServerForRuntime(l.ctx, req.ServerID)
	if err != nil {
		l.Errorf("[QueryServerXrayConfig] query server error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server error: %v", err)
	}
	list, err := l.svcCtx.NodeModel.QueryServerXrayTemplates(l.ctx, req.ServerID, true)
	if err != nil {
		l.Errorf("[QueryServerXrayConfig] query templates error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server xray templates error: %v", err)
	}
	merged, err := xraytemplate.RenderMergedConfigBundle(srv, list)
	if err != nil {
		l.Errorf("[QueryServerXrayConfig] render template config error: %v", err.Error())
		return nil, err
	}
	if err := merged.ValidateRoutingReferences(); err != nil {
		l.Errorf("[QueryServerXrayConfig] validate routing references error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error()), "validate routing references")
	}
	raw, err := merged.Raw()
	if err != nil {
		return nil, err
	}
	return buildRawXrayConfigResponse(raw), nil
}

func buildRawXrayConfigResponse(raw xraytemplate.RawConfigBundle) *types.QueryServerXrayConfigResponse {
	return &types.QueryServerXrayConfigResponse{
		Inbounds:  rawConfigListOrEmpty(raw.Inbounds),
		Outbounds: rawConfigListOrEmpty(raw.Outbounds),
		DNS:       raw.DNS,
		Routing:   raw.Routing,
		Geodata:   raw.Geodata,
		Templates: []types.XrayTemplate{},
	}
}

func rawConfigListOrEmpty(values []json.RawMessage) []json.RawMessage {
	if values == nil {
		return []json.RawMessage{}
	}
	return values
}
