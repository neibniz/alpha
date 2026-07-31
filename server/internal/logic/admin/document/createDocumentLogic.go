package document

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/document"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateDocumentLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create document
func NewCreateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDocumentLogic {
	return &CreateDocumentLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateDocumentLogic) CreateDocument(req *types.CreateDocumentRequest) error {
	if err := l.svcCtx.DocumentModel.Insert(l.ctx, &document.Document{
		Title:   req.Title,
		Content: req.Content,
		Tags:    documentTagsToString(req.Tags),
		Show:    req.Show,
	}); err != nil {
		l.Errorw("[CreateDocument] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert document error: %v", err.Error())
	}
	return nil
}

func documentTagsToString(tags []string) string {
	return tool.StringSliceToString(tags)
}
