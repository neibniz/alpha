package application

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/adapter"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type PreviewSubscribeTemplateLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Preview Template
func NewPreviewSubscribeTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewSubscribeTemplateLogic {
	return &PreviewSubscribeTemplateLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewSubscribeTemplateLogic) PreviewSubscribeTemplate(req *types.PreviewSubscribeTemplateRequest) (resp *types.PreviewSubscribeTemplateResponse, err error) {
	data, err := l.svcCtx.ClientModel.FindOneForPreview(l.ctx, req.Id)
	if err != nil {
		l.Errorf("[PreviewSubscribeTemplateLogic] FindOneForPreview error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneForPreview error: %v", err.Error())
	}

	proxies, err := l.previewProxiesFromEnabledNodes()
	if err != nil {
		return nil, err
	}
	if len(proxies) == 0 {
		proxies = previewProxies()
	}

	sub := adapter.NewAdapter(data.SubscribeTemplate,
		adapter.WithProxies(proxies),
		adapter.WithSiteName("AlphaPanel"),
		adapter.WithSubscribeName("Test Subscribe"),
		adapter.WithOutputFormat(data.OutputFormat),
		adapter.WithUserInfo(adapter.User{
			Password:     "preview-user",
			ExpiredAt:    time.Now().AddDate(1, 0, 0),
			Download:     0,
			Upload:       0,
			Traffic:      1000,
			SubscribeURL: "https://example.com/subscribe",
		}))
	// Get client config
	a, err := sub.Client()
	if err != nil {
		l.Errorf("[PreviewSubscribeTemplateLogic] Client error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrMsg(err.Error()), "Client error: %v", err.Error())
	}
	bytes, err := a.Build()
	if err != nil {
		l.Errorf("[PreviewSubscribeTemplateLogic] Build error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrMsg(err.Error()), "Build error: %v", err.Error())
	}
	return &types.PreviewSubscribeTemplateResponse{
		Template: string(bytes),
	}, nil
}

func (l *PreviewSubscribeTemplateLogic) previewProxiesFromEnabledNodes() ([]adapter.Proxy, error) {
	enabled := true
	_, nodes, err := l.svcCtx.NodeModel.FilterNodeList(l.ctx, &node.FilterNodeParams{
		Page:    1,
		Size:    1000,
		MaxSize: 1000,
		Enabled: &enabled,
		Preload: true,
	})
	if err != nil {
		l.Errorf("[PreviewSubscribeTemplateLogic] query enabled nodes error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query enabled nodes error: %v", err.Error())
	}

	proxies := viewutil.NewRowsCollector[adapter.Proxy](len(nodes))
	templateCache, err := subscribeprofile.QueryEnabledServerXrayTemplateCache(l.ctx, l.svcCtx, nodes)
	if err != nil {
		l.Errorw("[PreviewSubscribeTemplateLogic] batch query server xray templates failed", logger.Field("error", err.Error()))
		templateCache = map[int64][]*node.ServerXrayTemplate{}
	}
	var bindingIndexes previewInboundBindingIndexCache
	for _, item := range nodes {
		if item == nil || item.Server == nil {
			continue
		}
		bindings := templateCache[item.ServerId]
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

		srv := *item.Server
		if item.Name != "" {
			srv.Name = item.Name
		}
		if item.Address != "" {
			srv.Address = item.Address
		}
		renderedProxies, err := xraytemplate.RenderSubscriptionProxyList(&srv, bindings, matchedBindings, xraytemplate.UserInfo{
			Password: "preview-user",
			UUID:     "preview-user",
		})
		if err != nil {
			l.Errorw("[PreviewSubscribeTemplateLogic] render subscription proxy failed", logger.Field("error", err.Error()), logger.Field("serverId", item.ServerId))
			continue
		}
		for _, rendered := range renderedProxies {
			proxy := rendered.Proxy
			if proxy == nil {
				continue
			}
			if item.Name != "" {
				proxy.Name = item.Name
			}
			proxy.Sort = item.Sort
			if item.Port > 0 {
				proxy.Port = uint16(item.Port)
			}
			proxies.Add(*proxy)
		}
	}
	return proxies.Rows(), nil
}

const previewInboundBindingIndexLinearLimit = 8

type previewInboundBindingIndexEntry struct {
	serverID int64
	index    *xraytemplate.InboundBindingIndex
}

type previewInboundBindingIndexCache struct {
	rows  []previewInboundBindingIndexEntry
	index map[int64]int
}

func (c *previewInboundBindingIndexCache) Get(serverID int64) (*xraytemplate.InboundBindingIndex, bool) {
	if c == nil {
		return nil, false
	}
	if c.index != nil {
		pos, ok := c.index[serverID]
		if !ok {
			return nil, false
		}
		return c.rows[pos].index, true
	}
	for _, row := range c.rows {
		if row.serverID == serverID {
			return row.index, true
		}
	}
	return nil, false
}

func (c *previewInboundBindingIndexCache) Set(serverID int64, bindingIndex *xraytemplate.InboundBindingIndex) {
	if c == nil || serverID <= 0 {
		return
	}
	if c.index != nil {
		if pos, ok := c.index[serverID]; ok {
			c.rows[pos].index = bindingIndex
			return
		}
		c.index[serverID] = len(c.rows)
		c.rows = append(c.rows, previewInboundBindingIndexEntry{serverID: serverID, index: bindingIndex})
		return
	}
	for i, row := range c.rows {
		if row.serverID == serverID {
			c.rows[i].index = bindingIndex
			return
		}
	}
	c.rows = append(c.rows, previewInboundBindingIndexEntry{serverID: serverID, index: bindingIndex})
	if len(c.rows) > previewInboundBindingIndexLinearLimit {
		c.index = make(map[int64]int, len(c.rows))
		for i, row := range c.rows {
			c.index[row.serverID] = i
		}
	}
}

func previewProxies() []adapter.Proxy {
	return []adapter.Proxy{
		{
			Sort:             1,
			Name:             "Preview VLESS Reality gRPC",
			Server:           "node.example.com",
			Port:             443,
			Type:             "vless",
			Security:         "reality",
			TLS:              true,
			SNI:              "www.microsoft.com",
			Fingerprint:      "chrome",
			RealityPublicKey: "preview-reality-public-key",
			RealityShortId:   "0123456789abcdef",
			Transport:        "grpc",
			ServiceName:      "grpc-service",
			ServerKey:        "preview-user",
			Flow:             "xtls-rprx-vision",
		},
		{
			Sort:        2,
			Name:        "Preview VMess WS TLS",
			Server:      "vmess.example.com",
			Port:        8443,
			Type:        "vmess",
			Security:    "tls",
			TLS:         true,
			SNI:         "vmess.example.com",
			Transport:   "ws",
			Host:        "cdn.example.com",
			Path:        "/ws",
			ServerKey:   "preview-user",
			Fingerprint: "chrome",
		},
		{
			Sort:      3,
			Name:      "Preview Shadowsocks",
			Server:    "ss.example.com",
			Port:      8388,
			Type:      "shadowsocks",
			Method:    "2022-blake3-aes-128-gcm",
			ServerKey: "preview-ss-server-key",
		},
		{
			Sort:         4,
			Name:         "Preview Hysteria2",
			Server:       "hy2.example.com",
			Port:         443,
			Type:         "hysteria2",
			SNI:          "hy2.example.com",
			ObfsPassword: "preview-obfs", // #nosec G101 -- nonfunctional template preview placeholder
			HopPorts:     "20000-20100",
			HopInterval:  30,
			UpMbps:       100,
			DownMbps:     200,
		},
		{
			Sort:                 5,
			Name:                 "Preview TUIC",
			Server:               "tuic.example.com",
			Port:                 443,
			Type:                 "tuic",
			ServerKey:            "preview-user",
			SNI:                  "tuic.example.com",
			UDPRelayMode:         "native",
			CongestionController: "bbr",
			ReduceRtt:            true,
		},
	}
}
