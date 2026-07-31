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

type QueryXrayTemplateCatalogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryXrayTemplateCatalogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryXrayTemplateCatalogLogic {
	return &QueryXrayTemplateCatalogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryXrayTemplateCatalogLogic) QueryXrayTemplateCatalog(req *types.QueryXrayTemplateCatalogRequest) (*types.QueryXrayTemplateCatalogResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 100
	}
	total, list, err := l.svcCtx.NodeModel.QueryXrayTemplateCatalog(l.ctx, &node.FilterXrayTemplateParams{
		Page:    req.Page,
		Size:    req.Size,
		Type:    req.Type,
		Search:  req.Search,
		Enabled: req.Enabled,
	})
	if err != nil {
		l.Errorf("[QueryXrayTemplateCatalog] query error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template catalog error: %v", err)
	}
	rows, err := xrayTemplateCatalogRowsToTypes(list)
	if err != nil {
		return nil, err
	}
	return &types.QueryXrayTemplateCatalogResponse{
		Total: total,
		List:  rows,
	}, nil
}
