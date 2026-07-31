package user

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type DeleteUserAuthMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Delete user auth method
func NewDeleteUserAuthMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserAuthMethodLogic {
	return &DeleteUserAuthMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserAuthMethodLogic) DeleteUserAuthMethod(req *types.DeleteUserAuthMethodRequest) error {
	if req == nil || req.UserId <= 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method request is required")
	}
	authType := strings.ToLower(strings.TrimSpace(req.AuthType))
	if authType == "" {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method type is required")
	}

	methods, err := l.svcCtx.UserModel.FindUserAuthMethodSummaries(l.ctx, req.UserId)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query user authentication methods: %v", err)
	}
	found := false
	remaining := 0
	for _, method := range methods {
		if strings.EqualFold(method.AuthType, authType) {
			found = true
			continue
		}
		remaining++
	}
	if !found {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method is not bound")
	}
	if remaining == 0 {
		return errors.Wrap(xerr.NewErrCodeMsg(400, "Cannot remove the last sign-in method"), "user must retain at least one sign-in method")
	}

	err = l.svcCtx.UserModel.DeleteUserAuthMethods(l.ctx, req.UserId, authType)
	if err != nil {
		l.Errorw("[DeleteUserAuthMethodLogic] Delete User Auth Method Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId), logger.Field("authType", authType))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "Delete User Auth Method Error")
	}
	if err = authsession.RevokeUser(l.ctx, l.svcCtx.Redis, req.UserId); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke user authentication sessions")
	}
	return nil
}
