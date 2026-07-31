package user

import (
	"sort"
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

const smallPublicAuthMethodLookupThreshold = 8

func isVisibleSignInMethod(authType string) bool {
	return authType != "mobile" && authType != "device"
}

func publicUserFromModel(item *userModel.User, now time.Time) types.User {
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
		Rules:                 userview.RulesFromRaw(item.Rules),
		CreatedAt:             timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:             timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if item.DeletedAt.Valid {
		resp.DeletedAt = timex.UnixMilliOrZero(item.DeletedAt.Time)
	}
	resp.AuthMethods = publicAuthMethodsFromValues(item.AuthMethods, true)
	return resp
}

func publicAuthMethodsFromPointers(methods []*userModel.AuthMethods, sortByPriority bool) []types.UserAuthMethod {
	if len(methods) == 0 {
		return []types.UserAuthMethod{}
	}
	collector := newPublicAuthMethodCollector(len(methods))
	for _, method := range methods {
		if method == nil {
			continue
		}
		if !isVisibleSignInMethod(method.AuthType) {
			continue
		}
		collector.Add(publicAuthMethodFromModel(*method))
	}
	resp := collector.Methods()
	if sortByPriority {
		sortAuthMethodsByPriority(resp)
	}
	return resp
}

func publicAuthMethodsFromValues(methods []userModel.AuthMethods, sortByPriority bool) []types.UserAuthMethod {
	if len(methods) == 0 {
		return []types.UserAuthMethod{}
	}
	collector := newPublicAuthMethodCollector(len(methods))
	for _, method := range methods {
		if !isVisibleSignInMethod(method.AuthType) {
			continue
		}
		collector.Add(publicAuthMethodFromModel(method))
	}
	resp := collector.Methods()
	if sortByPriority {
		sortAuthMethodsByPriority(resp)
	}
	return resp
}

type publicAuthMethodCollector struct {
	methods []types.UserAuthMethod
	seen    map[string]struct{}
	capHint int
}

func newPublicAuthMethodCollector(capHint int) publicAuthMethodCollector {
	return publicAuthMethodCollector{capHint: capHint}
}

func (c *publicAuthMethodCollector) Add(method types.UserAuthMethod) {
	if method.AuthType == "" {
		return
	}
	if c.seen != nil {
		if _, ok := c.seen[method.AuthType]; ok {
			return
		}
		c.seen[method.AuthType] = struct{}{}
		c.methods = append(c.methods, method)
		return
	}
	for _, item := range c.methods {
		if item.AuthType == method.AuthType {
			return
		}
	}
	if c.methods == nil {
		c.methods = make([]types.UserAuthMethod, 0, publicAuthMethodInitialCap(c.capHint))
	}
	c.methods = append(c.methods, method)
	if len(c.methods) > smallPublicAuthMethodLookupThreshold {
		c.upgradeLookup()
	}
}

func (c publicAuthMethodCollector) Methods() []types.UserAuthMethod {
	if len(c.methods) == 0 {
		return []types.UserAuthMethod{}
	}
	return c.methods
}

func (c *publicAuthMethodCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := publicAuthMethodLookupCap(len(c.methods), c.capHint)
	if nextCap > cap(c.methods) {
		methods := make([]types.UserAuthMethod, len(c.methods), nextCap)
		copy(methods, c.methods)
		c.methods = methods
	}
	c.seen = make(map[string]struct{}, nextCap)
	for _, item := range c.methods {
		c.seen[item.AuthType] = struct{}{}
	}
}

func publicAuthMethodInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallPublicAuthMethodLookupThreshold+1 {
		return smallPublicAuthMethodLookupThreshold + 1
	}
	return capHint
}

func publicAuthMethodLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func appendPublicAuthMethod(
	methods []types.UserAuthMethod,
	method types.UserAuthMethod,
	seen map[string]struct{},
) ([]types.UserAuthMethod, map[string]struct{}) {
	collector := publicAuthMethodCollector{
		methods: methods,
		seen:    seen,
		capHint: cap(methods),
	}
	collector.Add(method)
	return collector.methods, collector.seen
}

func publicAuthMethodFromModel(method userModel.AuthMethods) types.UserAuthMethod {
	identifier := method.AuthIdentifier
	if method.AuthIdentifierDisplay != "" {
		identifier = method.AuthIdentifierDisplay
	} else {
		switch method.AuthType {
		case "email":
		default:
			identifier = maskOpenID(identifier)
		}
	}
	return types.UserAuthMethod{
		AuthType:       method.AuthType,
		AuthIdentifier: identifier,
		Verified:       method.Verified,
	}
}

func sortAuthMethodsByPriority(methods []types.UserAuthMethod) {
	sort.Slice(methods, func(i, j int) bool {
		return getAuthTypePriority(methods[i].AuthType) < getAuthTypePriority(methods[j].AuthType)
	})
}

func userBoolValue(value *bool) bool {
	return value != nil && *value
}

func affiliateUserRowsToTypes(rows []affiliateUserListRow) []types.UserAffiliate {
	if len(rows) == 0 {
		return []types.UserAffiliate{}
	}
	resp := make([]types.UserAffiliate, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, types.UserAffiliate{
			Identifier:   maskAffiliateIdentifier(row.Identifier),
			Avatar:       row.Avatar,
			RegisteredAt: timex.UnixMilliOrZero(row.RegisteredAt),
			Enable:       row.Enable,
		})
	}
	return resp
}

func withdrawalRowsToTypes(rows []userModel.Withdrawal) []types.WithdrawalLog {
	if len(rows) == 0 {
		return []types.WithdrawalLog{}
	}
	resp := make([]types.WithdrawalLog, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, withdrawalFromModel(row))
	}
	return resp
}

func withdrawalFromModel(item userModel.Withdrawal) types.WithdrawalLog {
	return types.WithdrawalLog{
		Id:        item.Id,
		UserId:    item.UserId,
		Amount:    item.Amount,
		Content:   item.Content,
		Status:    item.Status,
		Reason:    item.Reason,
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func userSubscribeFromDetails(item *userModel.SubscribeDetails, includeResetTime bool) types.PublicUserSubscribe {
	return userSubscribeFromDetailsAt(item, includeResetTime, time.Now())
}

func userSubscribeRowsFromDetailsAt(rows []*userModel.SubscribeDetails, includeResetTime bool, now time.Time) []types.PublicUserSubscribe {
	if len(rows) == 0 {
		return []types.PublicUserSubscribe{}
	}
	collector := viewutil.NewRowsCollector[types.PublicUserSubscribe](len(rows))
	snapshotCache := ordersnapshot.NewParseCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(userSubscribeFromDetailsAtWithSnapshotCache(row, includeResetTime, now, snapshotCache))
	}
	return collector.Rows()
}

func userSubscribeFromDetailsAt(item *userModel.SubscribeDetails, includeResetTime bool, now time.Time) types.PublicUserSubscribe {
	return userSubscribeFromDetailsAtWithSnapshotCache(item, includeResetTime, now, nil)
}

func userSubscribeFromDetailsAtWithSnapshotCache(item *userModel.SubscribeDetails, includeResetTime bool, now time.Time, snapshotCache *ordersnapshot.ParseCache) types.PublicUserSubscribe {
	if item == nil {
		return types.PublicUserSubscribe{}
	}
	resp := types.PublicUserSubscribe{
		Id:               item.Id,
		ProductName:      item.ProductName,
		ExpireTime:       timex.UnixMilliOrZero(item.ExpireTime),
		Traffic:          item.Traffic,
		Download:         item.Download,
		Upload:           item.Upload,
		UsedTraffic:      subscriptionstate.UsedTraffic(item.Download, item.Upload),
		RemainingTraffic: subscriptionstate.RemainingTraffic(item.Traffic, item.Download, item.Upload),
		Token:            item.Token,
		Status:           item.Status,
	}
	resp.EffectiveStatus, resp.StatusReason = subscriptionstate.EffectiveStatus(
		item.Status,
		item.ExpireTime,
		item.Traffic,
		item.Download,
		item.Upload,
		now,
	)
	productAvailable := item.Subscribe != nil
	renewalEnabled := productAvailable && subscribeview.BoolValue(item.Subscribe.AllowRenewal)
	resetEnabled := productAvailable && subscribeview.BoolValue(item.Subscribe.AllowResetTraffic)
	renewal := subscriptionstate.RenewalCapability(item.Status, item.ExpireTime, productAvailable, renewalEnabled, now)
	reset := subscriptionstate.ResetTrafficCapability(item.Status, item.ExpireTime, productAvailable, resetEnabled, now)
	resp.CanRenew, resp.RenewalReason = renewal.Allowed, renewal.Reason
	resp.CanResetTraffic, resp.ResetTrafficReason = reset.Allowed, reset.Reason
	snapshot, hasSnapshot := snapshotCache.Parse(item.ProductSnapshot)
	if resp.ProductName == "" && hasSnapshot {
		resp.ProductName = snapshot.Name
	}
	resetCycle := int64(0)
	if hasSnapshot && snapshot.Kind == ordersnapshot.KindSubscribe {
		subscribe := ordersnapshot.SubscribeType(snapshot)
		resp.Subscribe = publicSubscribeProductFromSubscribe(subscribe)
		resetCycle = subscribe.ResetCycle
	} else {
		resp.Subscribe, resetCycle = publicSubscribeProductFromModel(item.Subscribe)
	}
	resp.Short, _ = tool.FixedUniqueString(item.Token, 8, "")
	if includeResetTime {
		resp.ResetTime = calculateNextResetTime(resp.ExpireTime, resetCycle, now)
	}
	return resp
}

func publicSubscribeProductFromModel(item *modelSubscribe.Subscribe) (types.PublicUserSubscribeProduct, int64) {
	subscribe := subscribeview.SubscribeFromModelLenient(item, subscribeview.SubscribeOptions{})
	return publicSubscribeProductFromSubscribe(subscribe), subscribe.ResetCycle
}

func publicSubscribeProductFromSubscribe(item types.Subscribe) types.PublicUserSubscribeProduct {
	discount := item.Discount
	if discount == nil {
		discount = []types.SubscribeDiscount{}
	}
	return types.PublicUserSubscribeProduct{
		Id:                item.Id,
		Name:              item.Name,
		UnitPrice:         item.UnitPrice,
		UnitTime:          item.UnitTime,
		Discount:          discount,
		Replacement:       item.Replacement,
		Traffic:           item.Traffic,
		SpeedLimit:        item.SpeedLimit,
		DeviceLimit:       item.DeviceLimit,
		AllowRenewal:      item.AllowRenewal,
		AllowResetTraffic: item.AllowResetTraffic,
		AllowDeduction:    item.AllowDeduction,
		ShowOriginalPrice: item.ShowOriginalPrice,
	}
}
