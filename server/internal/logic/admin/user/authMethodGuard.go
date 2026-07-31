package user

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func normalizeAdminAuthMethod(userID int64, authType, identifier string) (string, string, error) {
	authType = strings.ToLower(strings.TrimSpace(authType))
	identifier = privacy.Normalize(authType, identifier)
	if userID <= 0 || authType == "" || identifier == "" {
		return "", "", errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid authentication method")
	}
	if authType == "mobile" || authType == "device" {
		return "", "", errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method is no longer supported")
	}
	return authType, identifier, nil
}

func ensureAuthIdentifierAvailable(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, authType, identifier string) error {
	existing, err := svcCtx.UserModel.FindUserAuthMethodByOpenID(ctx, authType, identifier)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query authentication identifier: %v", err)
	}
	if existing.UserId != userID {
		return errors.Wrap(xerr.NewErrCode(xerr.UserExist), "authentication identifier already belongs to another user")
	}
	return nil
}
