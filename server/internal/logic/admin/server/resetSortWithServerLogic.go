package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ResetSortWithServerLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResetSortWithServerLogic Reset server sort
func NewResetSortWithServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetSortWithServerLogic {
	return &ResetSortWithServerLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetSortWithServerLogic) ResetSortWithServer(req *types.ResetSortRequest) error {
	err := l.svcCtx.NodeModel.Transaction(l.ctx, func(db *gorm.DB) error {
		return resetModelSort(l.ctx, db, &node.Server{}, req.Sort, false)
	})
	if err != nil {
		l.Errorw("[NodeSort] Update Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseUpdateError), err.Error())
	}
	return nil
}
