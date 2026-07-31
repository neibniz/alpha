package user

import (
	"context"
	"time"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserMembershipLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserMembershipLogic {
	return &UpdateUserMembershipLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserMembershipLogic) UpdateUserMembership(req *types.UpdateUserMembershipRequest) error {
	userInfo, err := l.svcCtx.UserModel.FindOneForContext(l.ctx, req.UserId)
	if err != nil {
		l.Errorw("[UpdateUserMembership] Find User Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Find User Error")
	}

	var expiresAt any
	if req.IsMember {
		if req.MemberExpiredAt <= time.Now().UnixMilli() {
			return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "member expired time must be in the future"), "member expired time must be in the future")
		}
		expiresAt = time.UnixMilli(req.MemberExpiredAt)
	} else {
		expiresAt = nil
	}

	err = l.svcCtx.DB.WithContext(l.ctx).
		Model(&modelUser.User{}).
		Where("id = ?", req.UserId).
		Update("member_expired_at", expiresAt).Error
	if err != nil {
		l.Errorw("[UpdateUserMembership] Update User Membership Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Update User Membership Error")
	}

	if err = l.svcCtx.UserModel.ClearUserCache(l.ctx, userInfo); err != nil {
		l.Errorw("[UpdateUserMembership] Clear User Cache Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Clear User Cache Error")
	}
	return nil
}
