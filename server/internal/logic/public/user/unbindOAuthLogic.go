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

type UnbindOAuthLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Unbind OAuth
func NewUnbindOAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnbindOAuthLogic {
	return &UnbindOAuthLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnbindOAuthLogic) UnbindOAuth(req *types.UnbindOAuthRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if !l.validator(req) {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "invalid parameter")
	}
	methods, err := l.svcCtx.UserModel.FindUserAuthMethodSummaries(l.ctx, u.Id)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query user auth methods failed: %v", err)
	}
	found := false
	for _, method := range methods {
		if method != nil && method.AuthType == req.Method {
			found = true
			break
		}
	}
	if !found {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "oauth method is not bound")
	}
	if len(methods) <= 1 {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "cannot remove the last sign-in method")
	}
	err = l.svcCtx.UserModel.DeleteUserAuthMethods(l.ctx, u.Id, req.Method)
	if err != nil {
		l.Errorw("delete user auth methods failed:", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete user auth methods failed: %v", err.Error())
	}

	return nil
}
func (l *UnbindOAuthLogic) validator(req *types.UnbindOAuthRequest) bool {
	return req.Method == "github"
}
