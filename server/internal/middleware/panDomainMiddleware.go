package middleware

import (
	"net/http"
	"strings"

	"github.com/neibniz/alpha/server/internal/logic/subscribe"
	"github.com/neibniz/alpha/server/internal/queryparams"
	"github.com/neibniz/alpha/server/internal/subscribeua"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/gin-gonic/gin"
)

func PanDomainMiddleware(svc *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		subscribeConfig := svc.ConfigSnapshot().Subscribe
		if subscribeConfig.PanDomain && c.Request.URL.Path == "/" {
			// intercept browser
			ua := c.GetHeader("User-Agent")

			if subscribeConfig.UserAgentLimit {
				if ua == "" {
					c.String(http.StatusForbidden, "Access denied")
					c.Abort()
					return
				}

				// query client user agents only; subscription templates are not needed here.
				clientUserAgents, err := svc.ClientModel.ListEnabledUserAgents(c.Request.Context())
				if err != nil {
					logger.Errorw("[PanDomainMiddleware] Query client list failed", logger.Field("error", err.Error()))
				}
				browserKeywords := subscribeua.NormalizeKeywords(subscribeConfig.UserAgentList, clientUserAgents)

				if !subscribeua.AllowsUserAgent(ua, browserKeywords) {
					c.String(http.StatusForbidden, "Access denied")
					c.Abort()
					return
				}
			}

			domain := c.Request.Host
			domainFirst, rest, _ := strings.Cut(domain, ".")
			flag, _, _ := strings.Cut(rest, ".")
			request := types.SubscribeRequest{
				Token:   domainFirst,
				Flag:    flag,
				Type:    c.Query("type"),
				Profile: c.Query("profile"),
				Preset:  c.Query("preset"),
				UA:      c.Request.Header.Get("User-Agent"),
				Params:  queryparams.FirstValues(c.Request),
			}
			l := subscribe.NewSubscribeLogic(c, svc)
			resp, err := l.Handler(&request)
			if err != nil {
				return
			}
			c.Header("subscription-userinfo", resp.Header)
			c.String(200, "%s", string(resp.Config))
			c.Abort()
			return
		}
		c.Next()
	}
}
