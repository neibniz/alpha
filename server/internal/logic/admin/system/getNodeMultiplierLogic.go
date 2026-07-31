package system

import (
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetNodeMultiplierLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Node Multiplier
func NewGetNodeMultiplierLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNodeMultiplierLogic {
	return &GetNodeMultiplierLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNodeMultiplierLogic) GetNodeMultiplier() (resp *types.GetNodeMultiplierResponse, err error) {
	data, err := l.svcCtx.SystemModel.FindNodeMultiplierConfig(l.ctx)
	if err != nil {
		l.Logger.Error("Get Node Multiplier Config Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get Node Multiplier Config Error: %s", err.Error())
	}
	periods, err := nodeMultiplierPeriodsFromRaw(data.Value)
	if err != nil {
		l.Logger.Error("Unmarshal Node Multiplier Config Error: ", logger.Field("error", err.Error()), logger.Field("value", data.Value))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unmarshal Node Multiplier Config Error: %s", err.Error())
	}

	return &types.GetNodeMultiplierResponse{
		Periods: periods,
	}, nil
}

func nodeMultiplierPeriodsFromRaw(raw string) ([]types.TimePeriod, error) {
	if raw == "" {
		return []types.TimePeriod{}, nil
	}
	var periods []types.TimePeriod
	if err := json.Unmarshal([]byte(raw), &periods); err != nil {
		return nil, err
	}
	return normalizeNodeMultiplierPeriods(periods), nil
}

func normalizeNodeMultiplierPeriods(periods []types.TimePeriod) []types.TimePeriod {
	if len(periods) == 0 {
		return []types.TimePeriod{}
	}
	return periods
}
