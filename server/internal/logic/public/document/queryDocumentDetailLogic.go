package document

import (
	"context"

	"github.com/neibniz/alpha/server/internal/documentview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryDocumentDetailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get document detail
func NewQueryDocumentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryDocumentDetailLogic {
	return &QueryDocumentDetailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryDocumentDetailLogic) QueryDocumentDetail(req *types.QueryDocumentDetailRequest) (resp *types.Document, err error) {
	// find document
	data, err := l.svcCtx.DocumentModel.QueryDocumentDetail(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[QueryDocumentDetailLogic] QueryDocumentDetail error", logger.Field("id", req.Id), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryDocumentDetail error: %s", err.Error())
	}
	doc := documentview.PublicDetail(data)
	resp = &doc
	return
}
