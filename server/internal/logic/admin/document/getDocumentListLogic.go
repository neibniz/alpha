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

type GetDocumentListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get document list
func NewGetDocumentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentListLogic {
	return &GetDocumentListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDocumentListLogic) GetDocumentList(req *types.GetDocumentListRequest) (resp *types.GetDocumentListResponse, err error) {
	total, data, err := l.svcCtx.DocumentModel.QueryDocumentList(l.ctx, int(req.Page), int(req.Size), req.Tag, req.Search)
	if err != nil {
		l.Errorw("[GetDocumentList] Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryDocumentList error: %v", err.Error())
	}
	resp = &types.GetDocumentListResponse{
		Total: total,
		List:  documentview.AdminListRows(data),
	}
	return
}
