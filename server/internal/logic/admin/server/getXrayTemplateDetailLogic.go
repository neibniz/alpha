package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetXrayTemplateDetailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetXrayTemplateDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetXrayTemplateDetailLogic {
	return &GetXrayTemplateDetailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetXrayTemplateDetailLogic) GetXrayTemplateDetail(req *types.GetXrayTemplateDetailRequest) (*types.XrayTemplate, error) {
	if req.Id == 0 {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "template id is required"), "template id is required")
	}
	data, err := l.svcCtx.NodeModel.FindOneXrayTemplate(l.ctx, req.Id)
	if err != nil {
		l.Errorf("[GetXrayTemplateDetail] query error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template error: %v", err)
	}
	resp, err := xrayTemplateToTypes(data, true)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
