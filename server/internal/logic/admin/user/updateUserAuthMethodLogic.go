package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authsession"
	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserAuthMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update user auth method
func NewUpdateUserAuthMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserAuthMethodLogic {
	return &UpdateUserAuthMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserAuthMethodLogic) UpdateUserAuthMethod(req *types.UpdateUserAuthMethodRequest) error {
	if req == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method request is required")
	}
	authType, identifier, err := normalizeAdminAuthMethod(req.UserId, req.AuthType, req.AuthIdentifier)
	if err != nil {
		return err
	}
	req.AuthType = authType
	req.AuthIdentifier = identifier

	method, err := l.svcCtx.UserModel.FindUserAuthMethodByPlatform(l.ctx, req.UserId, authType)
	if err != nil {
		l.Errorw("Get user auth method error", logger.Field("error", err.Error()), logger.Field("userId", req.UserId), logger.Field("authType", authType))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get user auth method error: %v", err.Error())
	}
	if err = ensureAuthIdentifierAvailable(l.ctx, l.svcCtx, req.UserId, authType, identifier); err != nil {
		return err
	}

	if err = l.svcCtx.UserModel.UpdateUserAuthMethods(l.ctx, method.UserId, authType, userModel.UserAuthMethodFieldsUpdate{
		AuthIdentifier: userModel.UpdateValue(identifier),
	}); err != nil {
		l.Errorw("Update user auth method error", logger.Field("error", err.Error()), logger.Field("userId", req.UserId), logger.Field("authType", authType))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Update user auth method error: %v", err.Error())
	}
	if err = authsession.RevokeUser(l.ctx, l.svcCtx.Redis, req.UserId); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke user authentication sessions")
	}
	return nil
}
