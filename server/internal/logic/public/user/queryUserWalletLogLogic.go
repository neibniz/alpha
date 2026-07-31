package user

import (
	"context"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryUserWalletLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserWalletLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserWalletLogLogic {
	return &QueryUserWalletLogLogic{Logger: logger.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *QueryUserWalletLogLogic) QueryUserWalletLog(req *types.QueryUserWalletLogListRequest) (*types.QueryUserWalletLogListResponse, error) {
	if req == nil {
		req = &types.QueryUserWalletLogListRequest{}
	}
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not found in context")
	}
	rows, total, err := l.svcCtx.LogModel.FilterSystemLog(l.ctx, &modelLog.FilterParams{
		Page: req.Page, Size: req.Size, ObjectID: u.Id,
		Types: []int{int(modelLog.TypeBalance), int(modelLog.TypeCommission), int(modelLog.TypeGift)},
	})
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query wallet logs")
	}
	return &types.QueryUserWalletLogListResponse{List: walletLogRows(rows), Total: total}, nil
}

func walletLogRows(rows []*modelLog.SystemLog) []types.WalletLog {
	result := make([]types.WalletLog, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		entry := types.WalletLog{Id: row.Id}
		switch modelLog.Type(row.Type) {
		case modelLog.TypeBalance:
			var item modelLog.Balance
			if item.Unmarshal([]byte(row.Content)) != nil {
				continue
			}
			entry.Asset, entry.Type, entry.Amount, entry.OrderNo, entry.Timestamp = "balance", item.Type, signedWalletAmount("balance", item.Type, item.Amount), item.OrderNo, item.Timestamp
			entry.Balance = &item.Balance
		case modelLog.TypeGift:
			var item modelLog.Gift
			if item.Unmarshal([]byte(row.Content)) != nil {
				continue
			}
			entry.Asset, entry.Type, entry.Amount, entry.OrderNo, entry.Timestamp = "gift", item.Type, signedWalletAmount("gift", item.Type, item.Amount), item.OrderNo, item.Timestamp
			entry.Balance = &item.Balance
		case modelLog.TypeCommission:
			var item modelLog.Commission
			if item.Unmarshal([]byte(row.Content)) != nil {
				continue
			}
			entry.Asset, entry.Type, entry.Amount, entry.OrderNo, entry.Timestamp = "commission", item.Type, signedWalletAmount("commission", item.Type, item.Amount), item.OrderNo, item.Timestamp
			if item.Type == modelLog.CommissionTypePayment || item.Type == modelLog.CommissionTypePaymentRefund {
				entry.Balance = &item.Balance
			}
		default:
			continue
		}
		result = append(result, entry)
	}
	return result
}

func signedWalletAmount(asset string, typ uint16, amount int64) int64 {
	negative := (asset == "balance" && (typ == modelLog.BalanceTypeWithdraw || typ == modelLog.BalanceTypePayment)) ||
		(asset == "gift" && typ == modelLog.GiftTypeReduce) ||
		(asset == "commission" && (typ == modelLog.CommissionTypeRefund || typ == modelLog.CommissionTypeWithdraw || typ == modelLog.CommissionTypeConvertBalance || typ == modelLog.CommissionTypePayment))
	if negative && amount > 0 {
		return -amount
	}
	return amount
}
