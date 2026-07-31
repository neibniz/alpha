package user

import (
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserRulesLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserRulesLogic Update User Rules
func NewUpdateUserRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserRulesLogic {
	return &UpdateUserRulesLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserRulesLogic) UpdateUserRules(req *types.UpdateUserRulesRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	rules := normalizeUserRules(req.Rules)
	rulesJSON, err := marshalUserRules(rules)
	if err != nil {
		l.Logger.Errorf("UpdateUserRulesLogic json marshal rules error: %v", err)
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "json marshal rules failed: %v", err.Error())
	}
	if err = l.svcCtx.UserModel.UpdateUserRules(l.ctx, u.Id, rulesJSON); err != nil {
		l.Logger.Errorf("UpdateUserRulesLogic UpdateUserRules error: %v", err)
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update user rules failed: %v", err.Error())
	}
	return nil
}

func normalizeUserRules(rules []string) []string {
	return tool.NormalizeStringSlice(rules)
}

func marshalUserRules(rules []string) (string, error) {
	bytes, err := json.Marshal(rules)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
