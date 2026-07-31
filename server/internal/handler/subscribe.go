package handler

import (
	"net/http"
	"strings"

	"github.com/neibniz/alpha/server/internal/logic/subscribe"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/queryparams"
	"github.com/neibniz/alpha/server/internal/subscribeua"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/gin-gonic/gin"
)

func SubscribeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.SubscribeRequest
		if c.Request.Header.Get("token") != "" {
			req.Token = c.Request.Header.Get("token")
		} else {
			req.Token = c.Query("token")
		}
		ua := c.GetHeader("User-Agent")
		req.UA = c.Request.Header.Get("User-Agent")
		req.Flag = c.Query("flag")
		req.Type = c.Query("type")
		req.Profile = c.Query("profile")
		req.Preset = c.Query("preset")
		// 获取所有查询参数
		req.Params = queryparams.FirstValues(c.Request)

		subscribeConfig := svcCtx.ConfigSnapshot().Subscribe
		if subscribeConfig.PanDomain {
			domain := c.Request.Host
			domainPrefix, _, _ := strings.Cut(domain, ".")
			short, err := tool.FixedUniqueString(req.Token, 8, "")
			if err != nil {
				logger.Errorf("[SubscribeHandler] Generate short token failed: %v", err)
				c.String(http.StatusInternalServerError, "Internal Server")
				c.Abort()
				return
			}
			if strings.ToLower(short) != strings.ToLower(domainPrefix) {
				logger.Debugf("[SubscribeHandler] Generated token prefix mismatch, short: %s, domain: %s", privacy.MaskToken(short), privacy.MaskIdentifier(domainPrefix))
				c.String(http.StatusForbidden, "Access denied")
				c.Abort()
				return
			}
		}

		if subscribeConfig.UserAgentLimit {
			if ua == "" {
				c.String(http.StatusForbidden, "Access denied")
				c.Abort()
				return
			}

			// query client user agents only; subscription templates are not needed here.
			clientUserAgentsFromDB, err := svcCtx.ClientModel.ListEnabledUserAgents(c.Request.Context())
			if err != nil {
				logger.Errorw("[PanDomainMiddleware] Query client list failed", logger.Field("error", err.Error()))
			}
			clientUserAgents := subscribeua.NormalizeKeywords(subscribeConfig.UserAgentList, clientUserAgentsFromDB)

			if !subscribeua.AllowsUserAgent(ua, clientUserAgents) {
				c.String(http.StatusForbidden, "Access denied")
				c.Abort()
				return
			}
		}

		l := subscribe.NewSubscribeLogic(c, svcCtx)
		resp, err := l.Handler(&req)
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server")
			return
		}
		c.Header("subscription-userinfo", resp.Header)
		c.String(200, "%s", string(resp.Config))
	}
}

func RegisterSubscribeHandlers(router *gin.Engine, serverCtx *svc.ServiceContext) {
	path := serverCtx.ConfigSnapshot().Subscribe.SubscribePath
	if path == "" {
		path = "/v1/subscribe/config"
	}
	router.GET(path, SubscribeHandler(serverCtx))
}
