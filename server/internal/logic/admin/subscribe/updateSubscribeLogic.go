package subscribe

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/server/pkg/device"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update subscribe
func NewUpdateSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSubscribeLogic {
	return &UpdateSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSubscribeLogic) UpdateSubscribe(req *types.UpdateSubscribeRequest) error {
	if err := validateSubscribeFields(req.UnitPrice, req.Replacement, req.Inventory, req.Traffic, req.SpeedLimit, req.DeviceLimit, req.Quota, req.DeductionRatio, req.ResetCycle, req.UnitTime, req.Discount); err != nil {
		return err
	}
	minQuantity := productpolicy.NormalizeMinQuantity(req.MinQuantity)
	saleStartsAt := productTimePointer(req.SaleStartsAt)
	saleEndsAt := productTimePointer(req.SaleEndsAt)
	if err := validateMerchandisingFields(minQuantity, req.MaxQuantity, saleStartsAt, saleEndsAt); err != nil {
		return err
	}
	// Query the database to get the subscribe information
	existing, err := l.svcCtx.SubscribeModel.FindOneForUpdateDefaults(l.ctx, req.Id)
	if err != nil {
		l.Logger.Error("[UpdateSubscribe] Database query error", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscribe error: %v", err.Error())
	}
	discount := ""
	if len(req.Discount) > 0 {
		val, _ := json.Marshal(req.Discount)
		discount = string(val)
	}
	sub := &subscribe.Subscribe{
		Id:                req.Id,
		Name:              req.Name,
		Language:          req.Language,
		Description:       req.Description,
		UnitPrice:         req.UnitPrice,
		UnitTime:          req.UnitTime,
		Discount:          discount,
		Replacement:       req.Replacement,
		Inventory:         req.Inventory,
		Traffic:           req.Traffic,
		SpeedLimit:        req.SpeedLimit,
		DeviceLimit:       req.DeviceLimit,
		Quota:             req.Quota,
		Nodes:             subscribeNodeIDsToString(req.Nodes),
		NodeTags:          tool.StringSliceToString(req.NodeTags),
		Show:              boolPtrWithDefault(req.Show, boolPtrValue(existing.Show, false)),
		UserVisible:       boolPtrWithDefault(req.UserVisible, boolPtrValue(existing.UserVisible, false)),
		Sell:              boolPtrWithDefault(req.Sell, boolPtrValue(existing.Sell, false)),
		Sort:              req.Sort,
		DeductionRatio:    req.DeductionRatio,
		AllowDeduction:    boolPtrWithDefault(req.AllowDeduction, boolPtrValue(existing.AllowDeduction, true)),
		AllowRenewal:      boolPtrWithDefault(req.AllowRenewal, boolPtrValue(existing.AllowRenewal, true)),
		AllowResetTraffic: boolPtrWithDefault(req.AllowResetTraffic, boolPtrValue(existing.AllowResetTraffic, true)),
		ResetCycle:        req.ResetCycle,
		RenewalReset:      boolPtrWithDefault(req.RenewalReset, boolPtrValue(existing.RenewalReset, false)),
		ShowOriginalPrice: req.ShowOriginalPrice,
		DisplayTemplate:   productpolicy.NormalizeTemplate(req.DisplayTemplate),
		Badge:             strings.TrimSpace(req.Badge),
		MinQuantity:       minQuantity,
		MaxQuantity:       req.MaxQuantity,
		SaleStartsAt:      saleStartsAt,
		SaleEndsAt:        saleEndsAt,
	}
	err = l.svcCtx.SubscribeModel.Update(l.ctx, sub)
	if err != nil {
		l.Logger.Error("[UpdateSubscribe] update subscribe failed", logger.Field("error", err.Error()), logger.Field("subscribe", sub))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update subscribe error: %v", err.Error())
	}
	l.svcCtx.DeviceManager.Broadcast(device.SubscribeUpdate)
	return nil
}

func boolPtrValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
