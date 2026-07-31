package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type FilterXrayTemplateListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFilterXrayTemplateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterXrayTemplateListLogic {
	return &FilterXrayTemplateListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterXrayTemplateListLogic) FilterXrayTemplateList(req *types.FilterXrayTemplateListRequest) (*types.FilterXrayTemplateListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	total, list, err := l.svcCtx.NodeModel.FilterXrayTemplateList(l.ctx, &node.FilterXrayTemplateParams{
		Page:    req.Page,
		Size:    req.Size,
		Type:    req.Type,
		Search:  req.Search,
		Enabled: req.Enabled,
	})
	if err != nil {
		l.Errorf("[FilterXrayTemplateList] query error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template list error: %v", err)
	}
	return &types.FilterXrayTemplateListResponse{
		Total: total,
		List:  xrayTemplateListRowsToTypes(list),
	}, nil
}
