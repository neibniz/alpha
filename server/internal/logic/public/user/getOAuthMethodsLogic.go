package user

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetOAuthMethodsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get OAuth Methods
func NewGetOAuthMethodsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOAuthMethodsLogic {
	return &GetOAuthMethodsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOAuthMethodsLogic) GetOAuthMethods() (resp *types.GetOAuthMethodsResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	methods, err := l.svcCtx.UserModel.FindUserAuthMethodSummaries(l.ctx, u.Id)
	if err != nil {
		l.Errorw("find user auth methods failed:", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user auth methods failed: %v", err.Error())
	}
	return &types.GetOAuthMethodsResponse{
		Methods: publicAuthMethodsFromPointers(methods, false),
	}, nil
}
