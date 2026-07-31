package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type QueryUserInfoLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query User Info
func NewQueryUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserInfoLogic {
	return &QueryUserInfoLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserInfoLogic) QueryUserInfo() (resp *types.User, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	userInfo := publicUserFromModel(u, time.Now())
	return &userInfo, nil
}

// getAuthTypePriority keeps email first and OAuth methods after it.
func getAuthTypePriority(authType string) int {
	switch authType {
	case "email":
		return 1
	default:
		return 100
	}
}

// maskOpenID 脱敏 OpenID，只保留前 3 和后 3 位
func maskOpenID(openID string) string {
	length := len(openID)
	if length <= 6 {
		return "***" // 如果 ID 太短，直接返回 "***"
	}

	// 计算中间需要被替换的 `*` 数量
	maskLength := length - 6
	mask := make([]byte, maskLength)
	for i := range mask {
		mask[i] = '*'
	}

	// 组合脱敏后的 OpenID
	return openID[:3] + string(mask) + openID[length-3:]
}
