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

type QueryDocumentListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get document list
func NewQueryDocumentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryDocumentListLogic {
	return &QueryDocumentListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryDocumentListLogic) QueryDocumentList(req *types.QueryDocumentListRequest) (resp *types.QueryDocumentListResponse, err error) {
	if req == nil {
		req = &types.QueryDocumentListRequest{}
	}
	total, data, err := l.svcCtx.DocumentModel.GetDocumentListByAll(l.ctx, req.Page, req.Size)
	if err != nil {
		l.Errorw("[QueryDocumentList] error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryDocumentList error: %v", err.Error())
	}
	resp = &types.QueryDocumentListResponse{
		Total: total,
		List:  documentview.PublicSummaryRows(data),
	}
	return
}
