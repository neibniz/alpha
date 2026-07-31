package document

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type BatchDeleteDocumentLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Batch delete document
func NewBatchDeleteDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteDocumentLogic {
	return &BatchDeleteDocumentLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchDeleteDocumentLogic) BatchDeleteDocument(req *types.BatchDeleteDocumentRequest) error {
	if err := l.svcCtx.DocumentModel.BatchDelete(l.ctx, req.Ids); err != nil {
		l.Errorw("[BatchDeleteDocument] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "failed to delete document: %v", err.Error())
	}
	return nil
}
