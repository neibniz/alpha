package subscribe

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/adapter"
	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"

	"github.com/neibniz/alpha/server/internal/model/user"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

//goland:noinspection GoNameStartsWithPackageName
type SubscribeLogic struct {
	ctx *gin.Context
	svc *svc.ServiceContext
	logger.Logger
}

func NewSubscribeLogic(ctx *gin.Context, svc *svc.ServiceContext) *SubscribeLogic {
	return &SubscribeLogic{
		ctx:    ctx,
		svc:    svc,
		Logger: logger.WithContext(ctx.Request.Context()),
	}
}

func (l *SubscribeLogic) Handler(req *types.SubscribeRequest) (resp *types.SubscribeResponse, err error) {
	// Match by lightweight client metadata first; fetch the large template only for the selected app.
	clients, err := l.svc.ClientModel.ListEnabledMatchers(l.ctx.Request.Context())
	if err != nil {
		l.Errorw("[SubscribeLogic] Query client list failed", logger.Field("error", err.Error()))
		return nil, err
	}

	userAgent := strings.ToLower(l.ctx.Request.UserAgent())
	requestedClient := strings.ToLower(strings.TrimSpace(req.Type))
	if requestedClient == "" {
		requestedClient = strings.ToLower(strings.TrimSpace(req.Flag))
	}

	var targetMatcher, defaultMatcher *client.SubscribeApplication

	for _, item := range clients {
		u := strings.ToLower(item.UserAgent)
		name := strings.ToLower(item.Name)
		if item.IsDefault {
			defaultMatcher = item
		}

		if requestedClient != "" && (requestedClient == u || requestedClient == name) {
			targetMatcher = item
			break
		}

		if requestedClient == "" && strings.Contains(userAgent, u) {
			// Special handling for Stash
			if strings.Contains(userAgent, "stash") && !strings.Contains(u, "stash") {
				continue
			}
			targetMatcher = item
			break
		}
	}
	if targetMatcher == nil {
		l.Debugw("[SubscribeLogic] No matching client found", logger.Field("userAgent", privacy.MaskUserAgent(userAgent)))
		if defaultMatcher == nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "No matching client found for user agent: %s", userAgent)
		}
		targetMatcher = defaultMatcher
	}
	targetApp, err := l.svc.ClientModel.FindOneForPreview(l.ctx.Request.Context(), targetMatcher.Id)
	if err != nil {
		l.Errorw("[SubscribeLogic] Query target client template failed", logger.Field("error", err.Error()), logger.Field("clientId", targetMatcher.Id))
		return nil, err
	}
	// Find user subscribe by token
	userSubscribe, err := l.getUserSubscribe(req.Token)
	if err != nil {
		l.Errorw("[SubscribeLogic] Get user subscribe failed", logger.Field("error", err.Error()), logger.Field("token_prefix", log.TokenPrefix(req.Token)))
		return nil, err
	}

	var subscribeStatus = false
	defer func() {
		l.logSubscribeActivity(subscribeStatus, userSubscribe, req)
	}()
	// find subscribe info
	subscribeInfo, fromSnapshot, err := l.getSubscribeProduct(userSubscribe)
	if err != nil {
		return nil, err
	}

	// Find subscription items by user subscribe and optional scope filters.
	servers, selection, err := l.getSubscriptionNodes(userSubscribe, subscribeInfo, fromSnapshot, req)
	if err != nil {
		return nil, err
	}
	xrayProxies := l.buildXraySubscriptionProxies(servers, userSubscribe, selection)
	a := adapter.NewAdapter(
		targetApp.SubscribeTemplate,
		adapter.WithProxies(xrayProxies),
		adapter.WithSiteName(l.svc.ConfigSnapshot().Site.SiteName),
		adapter.WithSubscribeName(subscribeInfo.Name),
		adapter.WithOutputFormat(targetApp.OutputFormat),
		adapter.WithUserInfo(adapter.User{
			Password:     userSubscribe.UUID,
			ExpiredAt:    userSubscribe.ExpireTime,
			Download:     userSubscribe.Download,
			Upload:       userSubscribe.Upload,
			Traffic:      userSubscribe.Traffic,
			SubscribeURL: l.getSubscribeV2URL(),
		}),
		adapter.WithParams(req.Params),
	)

	logger.Debugf("[SubscribeLogic] Building client config for user %d with URI %s", userSubscribe.UserId, l.getSubscribeV2URL())

	// Get client config
	adapterClient, err := a.Client()
	if err != nil {
		l.Errorw("[SubscribeLogic] Client error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(500), "Client error: %v", err.Error())
	}
	bytes, err := adapterClient.Build()
	if err != nil {
		l.Errorw("[SubscribeLogic] Build client config failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(500), "Build client config failed: %v", err.Error())
	}

	var formats = []string{"json", "yaml", "conf"}

	for _, format := range formats {
		if format == strings.ToLower(targetApp.OutputFormat) {
			l.ctx.Header("content-disposition", fmt.Sprintf("attachment;filename*=UTF-8''%s.%s", url.QueryEscape(l.svc.ConfigSnapshot().Site.SiteName), format))
			l.ctx.Header("Content-Type", "application/octet-stream; charset=UTF-8")

		}
	}

	resp = &types.SubscribeResponse{
		Config: bytes,
		Header: fmt.Sprintf(
			"upload=%d;download=%d;total=%d;expire=%d",
			userSubscribe.Upload, userSubscribe.Download, userSubscribe.Traffic, userSubscribe.ExpireTime.Unix(),
		),
	}
	subscribeStatus = true
	return
}

func (l *SubscribeLogic) getSubscribeV2URL() string {

	uri := l.ctx.Request.RequestURI
	// is gateway mode, add /sub prefix
	if report.IsGatewayMode() {
		uri = "/sub" + uri
	}
	// use custom domain if configured
	subscribeConfig := l.svc.ConfigSnapshot().Subscribe
	if subscribeConfig.SubscribeDomain != "" {
		domains := strings.Split(subscribeConfig.SubscribeDomain, "\n")
		return fmt.Sprintf("https://%s%s", domains[0], uri)
	}
	// use current request host
	return fmt.Sprintf("https://%s%s", l.ctx.Request.Host, uri)
}

func (l *SubscribeLogic) getUserSubscribe(token string) (*user.Subscribe, error) {
	userSub, err := l.svc.UserModel.FindOneSubscribeByTokenForRuntime(l.ctx.Request.Context(), token)
	if err != nil {
		l.Infow("[Generate Subscribe]find subscribe error", logger.Field("error", err.Error()), logger.Field("token_prefix", log.TokenPrefix(token)))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe error: %v", err.Error())
	}
	if !subscriptionstate.Available(userSub.Status, userSub.ExpireTime, userSub.Traffic, userSub.Download, userSub.Upload, time.Now()) {
		l.Infow("[Generate Subscribe]subscribe is not available", logger.Field("status", int(userSub.Status)), logger.Field("subscribe_id", userSub.Id), logger.Field("token_prefix", log.TokenPrefix(token)))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.SubscribeNotAvailable), "subscribe is not available")
	}
	userInfo, err := l.svc.UserModel.FindOneForSubscribeAccess(l.ctx.Request.Context(), userSub.UserId)
	if err != nil {
		l.Infow("[Generate Subscribe]find user error: %v", logger.Field("error", err.Error()), logger.Field("user_id", userSub.UserId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user error: %v", err.Error())
	}
	if userInfo.DeletedAt.Valid || userInfo.Enable == nil || !*userInfo.Enable {
		l.Infow("[Generate Subscribe]user is not available", logger.Field("user_id", userSub.UserId), logger.Field("subscribe_id", userSub.Id), logger.Field("token_prefix", log.TokenPrefix(token)))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.SubscribeNotAvailable), "subscribe is not available")
	}

	return userSub, nil
}

func (l *SubscribeLogic) logSubscribeActivity(subscribeStatus bool, userSub *user.Subscribe, req *types.SubscribeRequest) {
	if !subscribeStatus {
		return
	}

	err := l.svc.LogModel.InsertSubscribeAccessLog(l.ctx.Request.Context(), &log.SubscribeAccessLog{
		UserId:          userSub.UserId,
		UserSubscribeId: userSub.Id,
		TokenHash:       log.HashSubscribeToken(req.Token),
		TokenPrefix:     log.TokenPrefix(req.Token),
		UserAgent:       req.UA,
		ClientIP:        l.ctx.ClientIP(),
		Success:         true,
	})
	if err != nil {
		l.Errorw("[Generate Subscribe]insert subscribe access log error: %v", logger.Field("error", err.Error()))
	}
}

func (l *SubscribeLogic) getSubscriptionNodes(userSub *user.Subscribe, product *modelSubscribe.Subscribe, productFromSnapshot bool, req *types.SubscribeRequest) ([]*node.Node, *subscribeprofile.Selection, error) {
	if l.isSubscriptionExpired(userSub) {
		return l.createExpiredServers(), nil, nil
	}
	var (
		nodes []*node.Node
		err   error
	)
	if productFromSnapshot {
		nodes, err = subscribeprofile.QueryAvailableNodes(l.ctx.Request.Context(), l.svc, userSub)
	} else {
		nodes, err = subscribeprofile.QueryAvailableNodesForProduct(l.ctx.Request.Context(), l.svc, product)
	}
	if err != nil {
		return nil, nil, err
	}
	selection, err := subscribeprofile.ResolveSelection(l.ctx.Request.Context(), l.svc, userSub, req.Profile, req.Preset)
	if err != nil {
		return nil, nil, err
	}
	return subscribeprofile.FilterNodes(nodes, selection), selection, nil
}

func (l *SubscribeLogic) getSubscribeProduct(userSub *user.Subscribe) (*modelSubscribe.Subscribe, bool, error) {
	if snapshotSub, ok := ordersnapshot.SubscribeModel(userSub.ProductSnapshot); ok {
		return snapshotSub, true, nil
	}
	subDetails, err := l.svc.SubscribeModel.FindOneForRuntime(l.ctx.Request.Context(), userSub.SubscribeId)
	if err == nil {
		return subDetails, false, nil
	}
	l.Errorw("[SubscribeLogic] Find subscribe info failed", logger.Field("error", err.Error()), logger.Field("subscribeId", userSub.SubscribeId))
	return nil, false, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Find subscribe info failed: %v", err.Error())
}

func (l *SubscribeLogic) buildXraySubscriptionProxies(servers []*node.Node, userSubscribe *user.Subscribe, selection *subscribeprofile.Selection) []adapter.Proxy {
	if len(servers) == 0 {
		return []adapter.Proxy{}
	}
	xrayProxies := viewutil.NewRowsCollector[adapter.Proxy](len(servers))
	ctx := l.ctx.Request.Context()
	templateCache, err := subscribeprofile.QueryEnabledServerXrayTemplateCache(ctx, l.svc, servers)
	if err != nil {
		l.Errorw("[SubscribeLogic] batch query server xray templates failed", logger.Field("error", err.Error()))
		templateCache = map[int64][]*node.ServerXrayTemplate{}
	}
	if templateCache == nil {
		templateCache = map[int64][]*node.ServerXrayTemplate{}
	}
	bindingIndexes := newSubscribeInboundBindingIndexCache(len(servers))

	for _, item := range servers {
		if item == nil || item.Server == nil {
			continue
		}

		bindings, ok := templateCache[item.ServerId]
		if !ok {
			bindings, err = l.svc.NodeModel.QueryServerXrayTemplates(ctx, item.ServerId, true)
			if err != nil {
				l.Errorw("[SubscribeLogic] query server xray templates failed", logger.Field("error", err.Error()), logger.Field("serverId", item.ServerId))
				continue
			}
			templateCache[item.ServerId] = bindings
		}
		if len(bindings) == 0 {
			continue
		}
		index, ok := bindingIndexes.Get(item.ServerId)
		if !ok {
			index = xraytemplate.NewInboundBindingIndex(bindings)
			bindingIndexes.Set(item.ServerId, index)
		}
		matchedBindings := index.Match(item.Inbound())
		if len(matchedBindings) == 0 {
			continue
		}

		srv := subscriptionServer(item)
		inboundAlias := item.Inbound()
		renderedProxies, err := xraytemplate.RenderSubscriptionProxyList(srv, bindings, matchedBindings, xraytemplate.UserInfo{
			Password: userSubscribe.UUID,
			UUID:     userSubscribe.UUID,
		})
		if err != nil {
			l.Errorw("[SubscribeLogic] render xray subscription proxy failed", logger.Field("error", err.Error()), logger.Field("serverId", item.ServerId))
			continue
		}
		for _, rendered := range renderedProxies {
			proxy := rendered.Proxy
			if proxy == nil {
				continue
			}
			if displayName := selection.DisplayName(item.Id, inboundAlias); displayName != "" {
				proxy.Name = displayName
			} else if item.Name != "" {
				proxy.Name = item.Name
			}
			proxy.Sort = item.Sort
			if item.Port > 0 {
				proxy.Port = uint16(item.Port)
			}
			xrayProxies.Add(*proxy)
		}
	}

	return xrayProxies.Rows()
}

const smallSubscribeInboundBindingIndexCacheThreshold = 8

type subscribeInboundBindingIndexCacheEntry struct {
	serverID int64
	index    *xraytemplate.InboundBindingIndex
}

type subscribeInboundBindingIndexCache struct {
	rows    []subscribeInboundBindingIndexCacheEntry
	index   map[int64]int
	capHint int
}

func newSubscribeInboundBindingIndexCache(capHint int) subscribeInboundBindingIndexCache {
	return subscribeInboundBindingIndexCache{capHint: capHint}
}

func (c *subscribeInboundBindingIndexCache) Get(serverID int64) (*xraytemplate.InboundBindingIndex, bool) {
	if c.index != nil {
		pos, ok := c.index[serverID]
		if !ok {
			return nil, false
		}
		return c.rows[pos].index, true
	}
	for pos := range c.rows {
		if c.rows[pos].serverID == serverID {
			return c.rows[pos].index, true
		}
	}
	return nil, false
}

func (c *subscribeInboundBindingIndexCache) Set(serverID int64, bindingIndex *xraytemplate.InboundBindingIndex) {
	if serverID <= 0 {
		return
	}
	if c.index != nil {
		if pos, ok := c.index[serverID]; ok {
			c.rows[pos].index = bindingIndex
			return
		}
		c.index[serverID] = len(c.rows)
		c.rows = append(c.rows, subscribeInboundBindingIndexCacheEntry{serverID: serverID, index: bindingIndex})
		return
	}
	for pos := range c.rows {
		if c.rows[pos].serverID == serverID {
			c.rows[pos].index = bindingIndex
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, subscribeInboundBindingIndexCacheEntry{serverID: serverID, index: bindingIndex})
	if len(c.rows) > smallSubscribeInboundBindingIndexCacheThreshold {
		c.upgradeLookup()
	}
}

func (c *subscribeInboundBindingIndexCache) ensureRows() {
	if c.rows != nil {
		return
	}
	c.rows = make([]subscribeInboundBindingIndexCacheEntry, 0, subscribeInboundBindingIndexCacheInitialCap(c.capHint))
}

func (c *subscribeInboundBindingIndexCache) upgradeLookup() {
	if c.index != nil {
		return
	}
	nextCap := subscribeInboundBindingIndexCacheLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]subscribeInboundBindingIndexCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.index = make(map[int64]int, nextCap)
	for pos, row := range c.rows {
		c.index[row.serverID] = pos
	}
}

func subscribeInboundBindingIndexCacheInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSubscribeInboundBindingIndexCacheThreshold+1 {
		return smallSubscribeInboundBindingIndexCacheThreshold + 1
	}
	return capHint
}

func subscribeInboundBindingIndexCacheLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func subscriptionServer(item *node.Node) *node.Server {
	srv := *item.Server
	if item.Name != "" {
		srv.Name = item.Name
	}
	if item.Address != "" {
		srv.Address = item.Address
	}
	return &srv
}

func (l *SubscribeLogic) isSubscriptionExpired(userSub *user.Subscribe) bool {
	return subscriptionstate.IsExpired(userSub.ExpireTime, time.Now())
}

func (l *SubscribeLogic) createExpiredServers() []*node.Node {
	return []*node.Node{}
}
