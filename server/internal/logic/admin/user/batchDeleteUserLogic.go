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
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type BatchDeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger.Logger
}

func NewBatchDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteUserLogic {
	return &BatchDeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logger.WithContext(ctx),
	}
}

func (l *BatchDeleteUserLogic) BatchDeleteUser(req *types.BatchDeleteUserRequest) error {
	isDemo := strings.ToLower(os.Getenv("ALPHA_MODE")) == "demo"
	ids := tool.NormalizePositiveInt64Slice(req.Ids)
	if len(ids) == 0 {
		return nil
	}

	if tool.Contains(ids, 2) && isDemo {
		return errors.Wrapf(xerr.NewErrCodeMsg(503, "Demo mode does not allow deletion of the admin user"), "BatchDeleteUser failed: cannot delete admin user in demo mode")
	}
	current, _ := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if current != nil && tool.Contains(ids, current.Id) {
		return errors.Wrapf(xerr.NewErrCodeMsg(400, "Cannot delete the current admin user"), "BatchDeleteUser failed: cannot delete current admin user")
	}
	var deletingAdminCount int64
	if err := l.svcCtx.DB.WithContext(l.ctx).Model(&user.User{}).Where("id IN ? AND is_admin = ?", ids, true).Count(&deletingAdminCount).Error; err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "BatchDeleteUser failed: count deleting admin users: %v", err.Error())
	}
	if deletingAdminCount > 0 {
		var adminCount int64
		if err := l.svcCtx.DB.WithContext(l.ctx).Model(&user.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "BatchDeleteUser failed: count admin users: %v", err.Error())
		}
		if adminCount-deletingAdminCount <= 0 {
			return errors.Wrapf(xerr.NewErrCodeMsg(400, "Cannot delete the last admin user"), "BatchDeleteUser failed: cannot delete last admin user")
		}
	}

	err := l.svcCtx.UserModel.BatchDeleteUser(l.ctx, ids)
	if err != nil {
		l.Logger.Error("[BatchDeleteUserLogic] BatchDeleteUser failed: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "BatchDeleteUser failed: %v", err.Error())
	}
	return nil
}
