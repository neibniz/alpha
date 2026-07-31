package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type CreateUserAuthMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create user auth method
func NewCreateUserAuthMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserAuthMethodLogic {
	return &CreateUserAuthMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserAuthMethodLogic) CreateUserAuthMethod(req *types.CreateUserAuthMethodRequest) error {
	if req == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method request is required")
	}
	authType, identifier, err := normalizeAdminAuthMethod(req.UserId, req.AuthType, req.AuthIdentifier)
	if err != nil {
		return err
	}
	req.AuthType = authType
	req.AuthIdentifier = identifier
	if err = ensureAuthIdentifierAvailable(l.ctx, l.svcCtx, req.UserId, authType, identifier); err != nil {
		return err
	}

	err = l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		data, findErr := l.svcCtx.UserModel.FindUserAuthMethodByPlatform(l.ctx, req.UserId, req.AuthType)
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, &user.AuthMethods{
				UserId:         req.UserId,
				AuthType:       req.AuthType,
				AuthIdentifier: req.AuthIdentifier,
			}, db)
		}
		if findErr != nil {
			return findErr
		}
		identifier := req.AuthIdentifier
		return l.svcCtx.UserModel.UpdateUserAuthMethods(l.ctx, data.UserId, data.AuthType, user.UserAuthMethodFieldsUpdate{
			AuthIdentifier: &identifier,
		}, db)
	})
	if err != nil {
		l.Errorw("[CreateUserAuthMethodLogic] Create User Auth Method Error:", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Create User Auth Method Error")
	}
	if err = authsession.RevokeUser(l.ctx, l.svcCtx.Redis, req.UserId); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke user authentication sessions")
	}
	return nil
}
