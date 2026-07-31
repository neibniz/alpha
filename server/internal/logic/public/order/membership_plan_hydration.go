package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/membershipview"
	"github.com/neibniz/alpha/server/internal/model/membership"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/orderview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
)

func applyOrderSnapshot(orderInfo *types.OrderDetail, source *modelOrder.Details) {
	applyOrderSnapshotWithCache(orderInfo, source, nil)
}

func applyOrderSnapshotWithCache(orderInfo *types.OrderDetail, source *modelOrder.Details, snapshotCache *ordersnapshot.ParseCache) {
	if orderInfo == nil || source == nil {
		return
	}
	if orderInfo.ProductName == "" {
		orderInfo.ProductName = source.ProductName
	}
	snapshot, hasSnapshot := snapshotCache.Parse(source.ProductSnapshot)
	if orderInfo.ProductName == "" && hasSnapshot {
		orderInfo.ProductName = snapshot.Name
	}
	switch orderInfo.Type {
	case OrderTypeSubscribe, OrderTypeRenewal, OrderTypeResetTraffic:
		if hasSnapshot && snapshot.Kind == ordersnapshot.KindSubscribe {
			orderInfo.Subscribe = orderview.SubscribeProductFromSnapshot(snapshot)
		}
	case OrderTypeMembership:
		orderInfo.Subscribe = types.OrderSubscribeProduct{}
		if hasSnapshot && snapshot.Kind == ordersnapshot.KindMembership {
			orderInfo.MembershipPlan = ordersnapshot.MembershipPlanType(snapshot)
		}
	}
}

func applyPublicOrderSnapshotWithCache(orderInfo *types.PublicOrder, source *modelOrder.Details, snapshotCache *ordersnapshot.ParseCache) {
	if orderInfo == nil || source == nil {
		return
	}
	if orderInfo.ProductName == "" {
		orderInfo.ProductName = source.ProductName
	}
	snapshot, hasSnapshot := snapshotCache.Parse(source.ProductSnapshot)
	if orderInfo.ProductName == "" && hasSnapshot {
		orderInfo.ProductName = snapshot.Name
	}
	if orderInfo.ProductName == "" && source.Subscribe != nil {
		orderInfo.ProductName = source.Subscribe.Name
	}
	if orderInfo.Type == OrderTypeMembership && hasSnapshot && snapshot.Kind == ordersnapshot.KindMembership {
		orderInfo.MembershipPlan = ordersnapshot.MembershipPlanType(snapshot)
	}
}

func hydrateOrderMembershipPlans(ctx context.Context, svcCtx *svc.ServiceContext, orders []types.OrderDetail) error {
	planIDSet := membershipview.NewPlanIDSet(len(orders))
	for i := range orders {
		if orders[i].Type != OrderTypeMembership {
			continue
		}
		orders[i].Subscribe = types.OrderSubscribeProduct{}
		if orders[i].MembershipPlan != nil {
			continue
		}
		if orders[i].SubscribeId > 0 {
			planIDSet.Add(orders[i].SubscribeId)
		}
	}
	planIDs := planIDSet.Values()
	if len(planIDs) == 0 {
		return nil
	}

	plans := membershipview.NewPlanRows(planIDs)
	if err := svcCtx.DB.WithContext(ctx).
		Model(&membership.Plan{}).
		Select(membership.PlanTypeColumns()).
		Where("id IN ?", planIDs).
		Find(&plans).Error; err != nil {
		return err
	}

	planIndex := membershipview.NewPlanIndex(plans)
	for i := range orders {
		if orders[i].Type != OrderTypeMembership {
			continue
		}
		if plan, ok := planIndex.Find(orders[i].SubscribeId); ok {
			orders[i].MembershipPlan = plan
		}
	}
	return nil
}

func hydratePublicOrderMembershipPlans(ctx context.Context, svcCtx *svc.ServiceContext, orders []types.PublicOrder) error {
	planIDSet := membershipview.NewPlanIDSet(len(orders))
	for i := range orders {
		if orders[i].Type != OrderTypeMembership || orders[i].MembershipPlan != nil {
			continue
		}
		if orders[i].SubscribeId > 0 {
			planIDSet.Add(orders[i].SubscribeId)
		}
	}
	planIDs := planIDSet.Values()
	if len(planIDs) == 0 {
		return nil
	}

	plans := membershipview.NewPlanRows(planIDs)
	if err := svcCtx.DB.WithContext(ctx).
		Model(&membership.Plan{}).
		Select(membership.PlanTypeColumns()).
		Where("id IN ?", planIDs).
		Find(&plans).Error; err != nil {
		return err
	}

	planIndex := membershipview.NewPlanIndex(plans)
	for i := range orders {
		if orders[i].Type != OrderTypeMembership {
			continue
		}
		if plan, ok := planIndex.Find(orders[i].SubscribeId); ok {
			orders[i].MembershipPlan = plan
		}
	}
	return nil
}
