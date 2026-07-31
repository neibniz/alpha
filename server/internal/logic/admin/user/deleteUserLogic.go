package user

import (
	"context"
	"os"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger.Logger
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logger.WithContext(ctx),
	}
}

func (l *DeleteUserLogic) DeleteUser(req *types.GetDetailRequest) error {
	isDemo := strings.ToLower(os.Getenv("ALPHA_MODE")) == "demo"

	if req.Id == 2 && isDemo {
		return errors.Wrapf(xerr.NewErrCodeMsg(503, "Demo mode does not allow deletion of the admin user"), "delete user failed: cannot delete admin user in demo mode")
	}
	current, _ := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if current != nil && current.Id == req.Id {
		return errors.Wrapf(xerr.NewErrCodeMsg(400, "Cannot delete the current admin user"), "delete user failed: cannot delete current admin user")
	}
	target, err := l.svcCtx.UserModel.FindOneForContext(l.ctx, req.Id)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user error: %v", err.Error())
	}
	if target.IsAdmin != nil && *target.IsAdmin {
		var adminCount int64
		if err := l.svcCtx.DB.WithContext(l.ctx).Model(&user.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "count admin user error: %v", err.Error())
		}
		if adminCount <= 1 {
			return errors.Wrapf(xerr.NewErrCodeMsg(400, "Cannot delete the last admin user"), "delete user failed: cannot delete last admin user")
		}
	}
	err = l.svcCtx.UserModel.Delete(l.ctx, req.Id)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete user error: %v", err.Error())
	}
	return nil
}
