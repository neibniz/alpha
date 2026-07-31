package user

import (
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/userview"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
)

const smallAdminUserAuthMethodCapacityThreshold = 8

func userListItemFromModel(item *userModel.User, now time.Time) types.UserListItem {
	if item == nil {
		return types.UserListItem{}
	}
	resp := types.UserListItem{
		Id:              item.Id,
		Balance:         item.Balance,
		Commission:      item.Commission,
		GiftAmount:      item.GiftAmount,
		MemberExpiredAt: timex.UnixMilliOrZero(item.MemberExpiredAt),
		IsMember:        !item.MemberExpiredAt.IsZero() && item.MemberExpiredAt.After(now),
		ReferCode:       item.ReferCode,
		RefererId:       item.RefererId,
		Enable:          userBoolValue(item.Enable),
		CreatedAt:       timex.UnixMilliOrZero(item.CreatedAt),
	}
	if item.DeletedAt.Valid {
		resp.DeletedAt = timex.UnixMilliOrZero(item.DeletedAt.Time)
	}
	for _, method := range item.AuthMethods {
		if isVisibleSignInMethod(method.AuthType) {
			resp.PrimaryAuthMethod = userAuthMethodFromModel(method, false)
			break
		}
	}
	return resp
}

func userListRowsFromModels(rows []*userModel.User, now time.Time) []types.UserListItem {
	if len(rows) == 0 {
		return []types.UserListItem{}
	}
	collector := viewutil.NewRowsCollector[types.UserListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(userListItemFromModel(row, now))
	}
	return collector.Rows()
}

func userDetailFromModel(item *userModel.User, now time.Time) types.User {
	return userFromModelWithRulesCache(item, now, false, nil)
}

func userFromModel(item *userModel.User, now time.Time, formatMobile bool) types.User {
	return userFromModelWithRulesCache(item, now, formatMobile, nil)
}

func userFromModelWithRulesCache(item *userModel.User, now time.Time, formatMobile bool, rulesCache *userview.RulesCache) types.User {
	if item == nil {
		return types.User{}
	}
	resp := types.User{
		Id:                    item.Id,
		Avatar:                item.Avatar,
		Balance:               item.Balance,
		Commission:            item.Commission,
		ReferralPercentage:    item.ReferralPercentage,
		OnlyFirstPurchase:     userBoolValue(item.OnlyFirstPurchase),
		GiftAmount:            item.GiftAmount,
		MemberExpiredAt:       timex.UnixMilliOrZero(item.MemberExpiredAt),
		IsMember:              !item.MemberExpiredAt.IsZero() && item.MemberExpiredAt.After(now),
		ReferCode:             item.ReferCode,
		RefererId:             item.RefererId,
		Enable:                userBoolValue(item.Enable),
		IsAdmin:               userBoolValue(item.IsAdmin),
		EnableBalanceNotify:   userBoolValue(item.EnableBalanceNotify),
		EnableLoginNotify:     userBoolValue(item.EnableLoginNotify),
		EnableSubscribeNotify: userBoolValue(item.EnableSubscribeNotify),
		EnableTradeNotify:     userBoolValue(item.EnableTradeNotify),
		AuthMethods:           []types.UserAuthMethod{},
		Rules:                 rulesCache.Rules(item.Rules),
		CreatedAt:             timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:             timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if item.DeletedAt.Valid {
		resp.DeletedAt = timex.UnixMilliOrZero(item.DeletedAt.Time)
	}
	if len(item.AuthMethods) > 0 {
		resp.AuthMethods = userAuthMethodsFromValues(item.AuthMethods, formatMobile)
	}
	return resp
}

func userAuthMethodsFromValues(methods []userModel.AuthMethods, formatMobile bool) []types.UserAuthMethod {
	if len(methods) == 0 {
		return []types.UserAuthMethod{}
	}
	collector := newAdminUserAuthMethodCollector(len(methods))
	for _, method := range methods {
		if !isVisibleSignInMethod(method.AuthType) {
			continue
		}
		collector.Add(userAuthMethodFromModel(method, formatMobile))
	}
	return collector.Methods()
}

func userAuthMethodsFromModels(methods []*userModel.AuthMethods, formatMobile bool) []types.UserAuthMethod {
	if len(methods) == 0 {
		return []types.UserAuthMethod{}
	}
	collector := newAdminUserAuthMethodCollector(len(methods))
	for _, method := range methods {
		if method == nil {
			continue
		}
		if !isVisibleSignInMethod(method.AuthType) {
			continue
		}
		collector.Add(userAuthMethodFromModel(*method, formatMobile))
	}
	return collector.Methods()
}

type adminUserAuthMethodCollector struct {
	methods []types.UserAuthMethod
	capHint int
}

func newAdminUserAuthMethodCollector(capHint int) adminUserAuthMethodCollector {
	return adminUserAuthMethodCollector{capHint: capHint}
}

func (c *adminUserAuthMethodCollector) Add(method types.UserAuthMethod) {
	if c.methods == nil {
		c.methods = make([]types.UserAuthMethod, 0, adminUserAuthMethodInitialCap(c.capHint))
	}
	c.methods = append(c.methods, method)
	if len(c.methods) > smallAdminUserAuthMethodCapacityThreshold {
		nextCap := adminUserAuthMethodLookupCap(len(c.methods), c.capHint)
		if nextCap <= cap(c.methods) {
			return
		}
		methods := make([]types.UserAuthMethod, len(c.methods), nextCap)
		copy(methods, c.methods)
		c.methods = methods
	}
}

func (c adminUserAuthMethodCollector) Methods() []types.UserAuthMethod {
	if len(c.methods) == 0 {
		return []types.UserAuthMethod{}
	}
	return c.methods
}

func adminUserAuthMethodInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallAdminUserAuthMethodCapacityThreshold+1 {
		return smallAdminUserAuthMethodCapacityThreshold + 1
	}
	return capHint
}

func adminUserAuthMethodLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func userAuthMethodFromModel(method userModel.AuthMethods, _ bool) types.UserAuthMethod {
	identifier := method.AuthIdentifier
	if method.AuthIdentifierDisplay != "" {
		identifier = method.AuthIdentifierDisplay
	}
	return types.UserAuthMethod{
		AuthType:       method.AuthType,
		AuthIdentifier: identifier,
		Verified:       method.Verified,
	}
}

func isVisibleSignInMethod(authType string) bool {
	return authType != "mobile" && authType != "device"
}

func userBoolValue(value *bool) bool {
	return value != nil && *value
}

func userSubscribeFromDetails(item *userModel.SubscribeDetails) types.UserSubscribe {
	return userSubscribeFromDetailsWithSnapshotCache(item, nil)
}

func userSubscribeProductSummaryFromModel(item *modelSubscribe.Subscribe) types.UserSubscribeProductSummary {
	if item == nil {
		return types.UserSubscribeProductSummary{}
	}
	return types.UserSubscribeProductSummary{
		Id:          item.Id,
		Name:        item.Name,
		SpeedLimit:  item.SpeedLimit,
		DeviceLimit: item.DeviceLimit,
	}
}

func userSubscribeProductSummaryFromSnapshot(snapshot ordersnapshot.Snapshot) types.UserSubscribeProductSummary {
	return types.UserSubscribeProductSummary{
		Id:          snapshot.ID,
		Name:        snapshot.Name,
		SpeedLimit:  snapshot.SpeedLimit,
		DeviceLimit: snapshot.DeviceLimit,
	}
}

func userSubscribeFromDetailsWithSnapshotCache(item *userModel.SubscribeDetails, snapshotCache *ordersnapshot.ParseCache) types.UserSubscribe {
	if item == nil {
		return types.UserSubscribe{}
	}
	resp := types.UserSubscribe{
		Id:          item.Id,
		UserId:      item.UserId,
		OrderId:     item.OrderId,
		SubscribeId: item.SubscribeId,
		ProductName: item.ProductName,
		StartTime:   timex.UnixMilliOrZero(item.StartTime),
		ExpireTime:  timex.UnixMilliOrZero(item.ExpireTime),
		Traffic:     item.Traffic,
		Download:    item.Download,
		Upload:      item.Upload,
		Token:       item.Token,
		Status:      item.Status,
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if item.FinishedAt != nil {
		resp.FinishedAt = timex.UnixMilliOrZero(*item.FinishedAt)
	}
	snapshot, hasSnapshot := snapshotCache.Parse(item.ProductSnapshot)
	if resp.ProductName == "" && hasSnapshot {
		resp.ProductName = snapshot.Name
	}
	if hasSnapshot && snapshot.Kind == ordersnapshot.KindSubscribe {
		resp.Subscribe = ordersnapshot.SubscribeType(snapshot)
	} else {
		resp.Subscribe = subscribeFromModel(item.Subscribe)
	}
	resp.Short, _ = tool.FixedUniqueString(item.Token, 8, "")
	return resp
}

func userSubscribeListItemFromDetailsWithSnapshotCache(item *userModel.SubscribeDetails, snapshotCache *ordersnapshot.ParseCache, now time.Time) types.UserSubscribeListItem {
	if item == nil {
		return types.UserSubscribeListItem{}
	}
	resp := types.UserSubscribeListItem{
		Id:          item.Id,
		UserId:      item.UserId,
		SubscribeId: item.SubscribeId,
		ProductName: item.ProductName,
		StartTime:   timex.UnixMilliOrZero(item.StartTime),
		ExpireTime:  timex.UnixMilliOrZero(item.ExpireTime),
		Traffic:     item.Traffic,
		Download:    item.Download,
		Upload:      item.Upload,
		Status:      item.Status,
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
	resp.UsedTraffic = subscriptionstate.UsedTraffic(item.Download, item.Upload)
	resp.RemainingTraffic = subscriptionstate.RemainingTraffic(item.Traffic, item.Download, item.Upload)
	resp.EffectiveStatus, resp.StatusReason = subscriptionstate.EffectiveStatus(item.Status, item.ExpireTime, item.Traffic, item.Download, item.Upload, now)
	snapshot, hasSnapshot := snapshotCache.Parse(item.ProductSnapshot)
	if resp.ProductName == "" && hasSnapshot {
		resp.ProductName = snapshot.Name
	}
	if hasSnapshot && snapshot.Kind == ordersnapshot.KindSubscribe {
		resp.Subscribe = userSubscribeProductSummaryFromSnapshot(snapshot)
	} else {
		resp.Subscribe = userSubscribeProductSummaryFromModel(item.Subscribe)
	}
	if resp.Subscribe.Name == "" {
		resp.Subscribe.Name = resp.ProductName
	}
	return resp
}

func userSubscribeRowsFromDetails(rows []*userModel.SubscribeDetails) []types.UserSubscribe {
	if len(rows) == 0 {
		return []types.UserSubscribe{}
	}
	collector := viewutil.NewRowsCollector[types.UserSubscribe](len(rows))
	snapshotCache := ordersnapshot.NewParseCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(userSubscribeFromDetailsWithSnapshotCache(row, snapshotCache))
	}
	return collector.Rows()
}

func userSubscribeListRowsFromDetails(rows []*userModel.SubscribeDetails) []types.UserSubscribeListItem {
	if len(rows) == 0 {
		return []types.UserSubscribeListItem{}
	}
	collector := viewutil.NewRowsCollector[types.UserSubscribeListItem](len(rows))
	snapshotCache := ordersnapshot.NewParseCache(len(rows))
	now := time.Now()
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(userSubscribeListItemFromDetailsWithSnapshotCache(row, snapshotCache, now))
	}
	return collector.Rows()
}

func userSubscribeDetailFromDetails(item *userModel.SubscribeDetails, now time.Time) types.UserSubscribeDetail {
	base := userSubscribeFromDetails(item)
	resp := types.UserSubscribeDetail{
		Id:          base.Id,
		UserId:      base.UserId,
		OrderId:     base.OrderId,
		SubscribeId: base.SubscribeId,
		Subscribe:   base.Subscribe,
		ProductName: base.ProductName,
		StartTime:   base.StartTime,
		ExpireTime:  base.ExpireTime,
		FinishedAt:  base.FinishedAt,
		ResetTime:   base.ResetTime,
		Traffic:     base.Traffic,
		Download:    base.Download,
		Upload:      base.Upload,
		Token:       base.Token,
		Status:      base.Status,
		CreatedAt:   base.CreatedAt,
		UpdatedAt:   base.UpdatedAt,
	}
	if item != nil {
		resp.Note = item.Note
		resp.UsedTraffic = subscriptionstate.UsedTraffic(item.Download, item.Upload)
		resp.RemainingTraffic = subscriptionstate.RemainingTraffic(item.Traffic, item.Download, item.Upload)
		resp.EffectiveStatus, resp.StatusReason = subscriptionstate.EffectiveStatus(item.Status, item.ExpireTime, item.Traffic, item.Download, item.Upload, now)
	}
	if item != nil && item.User != nil {
		resp.User = userDetailFromModel(item.User, now)
	}
	return resp
}

func subscribeFromModel(item *modelSubscribe.Subscribe) types.Subscribe {
	return subscribeview.SubscribeFromModelLenient(item, subscribeview.SubscribeOptions{IncludeTopology: true})
}
