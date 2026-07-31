package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"

	"github.com/pkg/errors"
)

type PurchaseLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewPurchaseLogic Purchase subscription
func NewPurchaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurchaseLogic {
	return &PurchaseLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

const (
	MaxQuantity    = 1000
	MaxOrderAmount = 2147483647
)

func (l *PurchaseLogic) Purchase(req *types.PortalPurchaseRequest) (resp *types.PortalPurchaseResponse, err error) {
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	if req.Quantity > MaxQuantity {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "quantity exceeds maximum limit of %d", MaxQuantity)
	}
	return nil, errors.Wrapf(xerr.NewErrCode(xerr.MembershipRequired), "membership required")
}
