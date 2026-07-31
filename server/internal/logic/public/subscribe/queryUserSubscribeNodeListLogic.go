package subscribe

import (
	"context"
	"time"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryUserSubscribeNodeListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subscribe node info
func NewQueryUserSubscribeNodeListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserSubscribeNodeListLogic {
	return &QueryUserSubscribeNodeListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserSubscribeNodeListLogic) QueryUserSubscribeNodeList() (resp *types.QueryUserSubscribeNodeListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	userSubscribes, err := l.svcCtx.UserModel.QueryUserSubscribeRuntime(l.ctx, u.Id, 1, 2)
	if err != nil {
		logger.Errorw("failed to query user subscribe", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "DB_ERROR")
	}

	resp = &types.QueryUserSubscribeNodeListResponse{}
	activeSubscriptions := selectActiveSubscriptions(userSubscribes, time.Now())

	presetsBySubscribeID, err := subscribeprofile.QueryPresetsBySubscribeIDs(l.ctx, l.svcCtx, u.Id, activeSubscriptions.ids)
	if err != nil {
		return nil, err
	}
	if err = subscribeprofile.HydrateSubscribeProductsForDetails(l.ctx, l.svcCtx, activeSubscriptions.details); err != nil {
		return nil, err
	}
	nodeSelectorCache, profileCache := subscribeNodeListRuntimeCaches(activeSubscriptions.Len())
	list := viewutil.NewRowsCollector[types.UserSubscribeInfo](len(userSubscribes))
	activeCursor := 0
	for _, us := range userSubscribes {
		if us == nil {
			continue
		}
		var nodeInfos []types.UserSubscribeNodeInfo
		var profiles []types.UserSubscribeProfile
		var presets []types.UserSubscribePreset
		if activeSubscriptions.NextDetailIs(&activeCursor, us) {
			nodes, err := subscribeprofile.QueryAvailableNodesForSubscribeDetailsWithCache(l.ctx, l.svcCtx, us, nodeSelectorCache)
			if err != nil {
				return nil, err
			}
			nodeInfos, profiles, err = subscribeprofile.BuildNodeInfoAndProfilesWithRuntimeCache(l.ctx, l.svcCtx, nodes, us.UUID, profileCache)
			if err != nil {
				return nil, err
			}
			presets = presetsBySubscribeID[us.Id]
		}
		list.Add(userSubscribeInfoFromRuntimeDetails(us, nodeInfos, profiles, presets))
	}
	resp.List = list.Rows()

	return
}

func userSubscribeInfoFromRuntimeDetails(
	item *user.SubscribeDetails,
	nodes []types.UserSubscribeNodeInfo,
	profiles []types.UserSubscribeProfile,
	presets []types.UserSubscribePreset,
) types.UserSubscribeInfo {
	if item == nil {
		return types.UserSubscribeInfo{}
	}
	info := types.UserSubscribeInfo{
		Id: item.Id,
	}
	info.Nodes = nodeInfoRows(nodes)
	info.Profiles = profileRows(profiles)
	info.Presets = presetRows(presets)
	return info
}

func nodeInfoRows(nodes []types.UserSubscribeNodeInfo) []types.UserSubscribeNodeInfo {
	if len(nodes) == 0 {
		return []types.UserSubscribeNodeInfo{}
	}
	for i, node := range nodes {
		if !emptyNodeInfo(node) {
			continue
		}
		collector := viewutil.NewRowsCollector[types.UserSubscribeNodeInfo](len(nodes))
		for _, node := range nodes[:i] {
			collector.Add(node)
		}
		for _, rest := range nodes[i+1:] {
			if !emptyNodeInfo(rest) {
				collector.Add(rest)
			}
		}
		return collector.Rows()
	}
	return nodes
}

func emptyNodeInfo(node types.UserSubscribeNodeInfo) bool {
	return node.Id == 0 && node.Name == "" && node.InboundAlias == "" && node.ItemKey == ""
}

func profileRows(profiles []types.UserSubscribeProfile) []types.UserSubscribeProfile {
	if len(profiles) == 0 {
		return []types.UserSubscribeProfile{}
	}
	return profiles
}

func presetRows(presets []types.UserSubscribePreset) []types.UserSubscribePreset {
	if len(presets) == 0 {
		return []types.UserSubscribePreset{}
	}
	return presets
}

func subscribeNodeListRuntimeCaches(activeCount int) (map[string][]*modelNode.Node, *subscribeprofile.NodeProfileRuntimeCache) {
	capHint := subscribeNodeListRuntimeCacheCap(activeCount)
	if capHint == 0 {
		return nil, nil
	}
	return make(map[string][]*modelNode.Node, capHint), subscribeprofile.NewNodeProfileRuntimeCache(capHint)
}

func subscribeNodeListRuntimeCacheCap(activeCount int) int {
	if activeCount <= 1 {
		return 0
	}
	if activeCount > viewutil.SmallRowsCapacityThreshold+1 {
		return viewutil.SmallRowsCapacityThreshold + 1
	}
	return activeCount
}

type activeSubscriptionSelection struct {
	ids     []int64
	details []*user.SubscribeDetails
}

func selectActiveSubscriptions(items []*user.SubscribeDetails, now time.Time) activeSubscriptionSelection {
	var selection activeSubscriptionSelection
	for i, item := range items {
		if item == nil || subscriptionExpiredAt(item.ExpireTime, now) {
			continue
		}
		if selection.ids == nil {
			capHint := len(items) - i
			selection.ids = make([]int64, 0, capHint)
			selection.details = make([]*user.SubscribeDetails, 0, capHint)
		}
		selection.ids = append(selection.ids, item.Id)
		selection.details = append(selection.details, item)
	}
	return selection
}

func (s activeSubscriptionSelection) Len() int {
	return len(s.ids)
}

func (s activeSubscriptionSelection) NextDetailIs(cursor *int, item *user.SubscribeDetails) bool {
	if cursor == nil || *cursor >= len(s.details) || s.details[*cursor] != item {
		return false
	}
	*cursor = *cursor + 1
	return true
}

func subscriptionExpiredAt(expireTime time.Time, now time.Time) bool {
	expireUnix := expireTime.Unix()
	return expireUnix < now.Unix() && expireUnix != 0
}
