package coupon

import (
	couponModel "github.com/neibniz/alpha/server/internal/model/coupon"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func couponFromCreateRequest(req *types.CreateCouponRequest) *couponModel.Coupon {
	if req == nil {
		return &couponModel.Coupon{}
	}
	return &couponModel.Coupon{
		Name:       req.Name,
		Code:       req.Code,
		Count:      req.Count,
		Type:       req.Type,
		Discount:   req.Discount,
		StartTime:  req.StartTime,
		ExpireTime: req.ExpireTime,
		UserLimit:  req.UserLimit,
		Subscribe:  tool.Int64SliceToString(tool.NormalizePositiveInt64Slice(req.Subscribe)),
		UsedCount:  req.UsedCount,
		Enable:     req.Enable,
	}
}

func couponFromUpdateRequest(req *types.UpdateCouponRequest) *couponModel.Coupon {
	if req == nil {
		return &couponModel.Coupon{}
	}
	return &couponModel.Coupon{
		Id:         req.Id,
		Name:       req.Name,
		Code:       req.Code,
		Count:      req.Count,
		Type:       req.Type,
		Discount:   req.Discount,
		StartTime:  req.StartTime,
		ExpireTime: req.ExpireTime,
		UserLimit:  req.UserLimit,
		Subscribe:  tool.Int64SliceToString(tool.NormalizePositiveInt64Slice(req.Subscribe)),
		UsedCount:  req.UsedCount,
		Enable:     req.Enable,
	}
}

func couponResponseFromModel(item *couponModel.Coupon) types.Coupon {
	if item == nil {
		return types.Coupon{}
	}
	enable := false
	if item.Enable != nil {
		enable = *item.Enable
	}
	return types.Coupon{
		Id:         item.Id,
		Name:       item.Name,
		Code:       item.Code,
		Count:      item.Count,
		Type:       item.Type,
		Discount:   item.Discount,
		StartTime:  item.StartTime,
		ExpireTime: item.ExpireTime,
		UserLimit:  item.UserLimit,
		Subscribe:  tool.StringToPositiveUniqueInt64Slice(item.Subscribe),
		UsedCount:  item.UsedCount,
		Enable:     enable,
		CreatedAt:  timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:  timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func couponListItemFromModel(item *couponModel.Coupon) types.CouponListItem {
	if item == nil {
		return types.CouponListItem{}
	}
	enable := false
	if item.Enable != nil {
		enable = *item.Enable
	}
	return types.CouponListItem{
		Id:         item.Id,
		Name:       item.Name,
		Code:       item.Code,
		Count:      item.Count,
		Type:       item.Type,
		Discount:   item.Discount,
		StartTime:  item.StartTime,
		ExpireTime: item.ExpireTime,
		UserLimit:  item.UserLimit,
		UsedCount:  item.UsedCount,
		Enable:     enable,
	}
}

func couponRowsFromModels(rows []*couponModel.Coupon) []types.Coupon {
	if len(rows) == 0 {
		return []types.Coupon{}
	}
	collector := viewutil.NewRowsCollector[types.Coupon](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(couponResponseFromModel(row))
	}
	return collector.Rows()
}

func couponListRowsFromModels(rows []*couponModel.Coupon) []types.CouponListItem {
	if len(rows) == 0 {
		return []types.CouponListItem{}
	}
	collector := viewutil.NewRowsCollector[types.CouponListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(couponListItemFromModel(row))
	}
	return collector.Rows()
}
