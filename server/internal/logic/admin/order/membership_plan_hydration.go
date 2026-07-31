package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/membershipview"
	"github.com/neibniz/alpha/server/internal/model/membership"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
)

const orderTypeMembership uint8 = 5

func applyOrderSnapshot(orderInfo *types.Order, source *modelOrder.Details) {
	applyOrderSnapshotWithCache(orderInfo, source, nil)
}

func applyOrderSnapshotWithCache(orderInfo *types.Order, source *modelOrder.Details, snapshotCache *ordersnapshot.ParseCache) {
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
	if orderInfo.Type == orderTypeMembership && hasSnapshot && snapshot.Kind == ordersnapshot.KindMembership {
		orderInfo.MembershipPlan = ordersnapshot.MembershipPlanType(snapshot)
	}
}

func hydrateOrderMembershipPlans(ctx context.Context, svcCtx *svc.ServiceContext, orders []types.Order) error {
	planIDSet := membershipview.NewPlanIDSet(len(orders))
	for i := range orders {
		if orders[i].Type == orderTypeMembership && orders[i].MembershipPlan == nil && orders[i].SubscribeId > 0 {
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
		if orders[i].Type != orderTypeMembership {
			continue
		}
		if plan, ok := planIndex.Find(orders[i].SubscribeId); ok {
			orders[i].MembershipPlan = plan
		}
	}
	return nil
}
