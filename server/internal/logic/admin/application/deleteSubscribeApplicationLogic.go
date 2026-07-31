package application

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type DeleteSubscribeApplicationLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeleteSubscribeApplicationLogic Delete subscribe application
func NewDeleteSubscribeApplicationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSubscribeApplicationLogic {
	return &DeleteSubscribeApplicationLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSubscribeApplicationLogic) DeleteSubscribeApplication(req *types.DeleteSubscribeApplicationRequest) error {
	application, err := l.svcCtx.ClientModel.FindOneForUpdate(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to find subscribe application with ID %d: %v", req.Id, err)
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), err.Error())
	}
	if application.IsDefault {
		return errors.Wrap(
			xerr.NewErrCodeMsg(xerr.InvalidParams, "default subscription client cannot be deleted"),
			"set another subscription client as default before deleting this one",
		)
	}
	err = l.svcCtx.ClientModel.Delete(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to delete subscribe application with ID %d: %v", req.Id, err)
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseDeletedError), err.Error())
	}
	return nil
}
