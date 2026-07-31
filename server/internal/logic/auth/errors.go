package auth

import (
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

func authFailedError() error {
	return errors.Wrap(xerr.NewErrCode(xerr.UserAuthFailed), "invalid account or password")
}
