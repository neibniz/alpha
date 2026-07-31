package authMethod

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetAuthMethodListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetAuthMethodListLogic Get auth method list
func NewGetAuthMethodListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthMethodListLogic {
	return &GetAuthMethodListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAuthMethodListLogic) GetAuthMethodList() (resp *types.GetAuthMethodListResponse, err error) {
	methods, err := l.svcCtx.AuthModel.FindAllSummaries(l.ctx)
	if err != nil {
		l.Errorw("find all failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find all failed: %v", err.Error())
	}
	return &types.GetAuthMethodListResponse{List: authMethodSummaryRows(methods)}, nil
}
