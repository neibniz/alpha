package subscribe

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateSubscribeLogic Create subscribe
func NewCreateSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSubscribeLogic {
	return &CreateSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSubscribeLogic) CreateSubscribe(req *types.CreateSubscribeRequest) error {
	if err := validateSubscribeFields(req.UnitPrice, req.Replacement, req.Inventory, req.Traffic, req.SpeedLimit, req.DeviceLimit, req.Quota, req.DeductionRatio, req.ResetCycle, req.UnitTime, req.Discount); err != nil {
		return err
	}
	minQuantity := productpolicy.NormalizeMinQuantity(req.MinQuantity)
	saleStartsAt := productTimePointer(req.SaleStartsAt)
	saleEndsAt := productTimePointer(req.SaleEndsAt)
	if err := validateMerchandisingFields(minQuantity, req.MaxQuantity, saleStartsAt, saleEndsAt); err != nil {
		return err
	}
	discount := ""
	if len(req.Discount) > 0 {
		val, _ := json.Marshal(req.Discount)
		discount = string(val)
	}
	sub := &subscribe.Subscribe{
		Id:                0,
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
		Show:              boolPtrWithDefault(req.Show, false),
		UserVisible:       boolPtrWithDefault(req.UserVisible, false),
		Sell:              boolPtrWithDefault(req.Sell, false),
		Sort:              0,
		DeductionRatio:    req.DeductionRatio,
		AllowDeduction:    boolPtrWithDefault(req.AllowDeduction, true),
		AllowRenewal:      boolPtrWithDefault(req.AllowRenewal, true),
		AllowResetTraffic: boolPtrWithDefault(req.AllowResetTraffic, true),
		ResetCycle:        req.ResetCycle,
		RenewalReset:      boolPtrWithDefault(req.RenewalReset, false),
		ShowOriginalPrice: req.ShowOriginalPrice,
		DisplayTemplate:   productpolicy.NormalizeTemplate(req.DisplayTemplate),
		Badge:             strings.TrimSpace(req.Badge),
		MinQuantity:       minQuantity,
		MaxQuantity:       req.MaxQuantity,
		SaleStartsAt:      saleStartsAt,
		SaleEndsAt:        saleEndsAt,
	}
	err := l.svcCtx.SubscribeModel.Insert(l.ctx, sub)
	if err != nil {
		l.Logger.Error("[CreateSubscribeLogic] create subscribe error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create subscribe error: %v", err.Error())
	}

	return nil
}

func boolPtrWithDefault(value *bool, fallback bool) *bool {
	if value != nil {
		return value
	}
	return &fallback
}

func subscribeNodeIDsToString(nodes []int64) string {
	return tool.Int64SliceToString(tool.NormalizePositiveInt64Slice(nodes))
}

func validateSubscribeFields(unitPrice, replacement, inventory, traffic, speedLimit, deviceLimit, quota, deductionRatio, resetCycle int64, unitTime string, discounts []types.SubscribeDiscount) error {
	if unitPrice < 0 || replacement < 0 || traffic < 0 || speedLimit < 0 || deviceLimit < 0 || quota < 0 || deductionRatio < 0 || resetCycle < 0 {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscribe numeric fields must not be negative")
	}
	if inventory < -1 {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "inventory must be -1 or greater")
	}
	switch unitTime {
	case "Minute", "Hour", "Day", "Month", "Year", "NoLimit":
	default:
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "invalid unit time")
	}
	for _, item := range discounts {
		if item.Quantity <= 0 {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "discount quantity must be greater than zero")
		}
		if item.Discount < 0 || item.Discount > 100 {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "discount must be between 0 and 100")
		}
	}
	return nil
}

func productTimePointer(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.UnixMilli(value)
	return &t
}

func validateMerchandisingFields(minQuantity, maxQuantity int64, saleStartsAt, saleEndsAt *time.Time) error {
	if minQuantity < 1 || minQuantity > 1000 {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "minimum quantity must be between 1 and 1000")
	}
	if maxQuantity < 0 || maxQuantity > 1000 {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "maximum quantity must be between 0 and 1000")
	}
	if maxQuantity > 0 && maxQuantity < minQuantity {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "maximum quantity must be greater than or equal to minimum quantity")
	}
	if err := productpolicy.ValidateSaleWindow(saleStartsAt, saleEndsAt); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), err.Error())
	}
	return nil
}
