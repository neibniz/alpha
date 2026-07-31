package common

import (
	"context"

	modelAuth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetGlobalConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get global config
func NewGetGlobalConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGlobalConfigLogic {
	return &GetGlobalConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGlobalConfigLogic) GetGlobalConfig() (resp *types.GetGlobalConfigResponse, err error) {
	resp = new(types.GetGlobalConfigResponse)

	currencyCfg, err := l.svcCtx.SystemModel.GetCurrencyConfig(l.ctx)
	if err != nil {
		l.Logger.Error("[GetGlobalConfigLogic] GetCurrencyConfig error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetCurrencyConfig error: %v", err.Error())
	}
	verifyCodeCfg, err := l.svcCtx.SystemModel.GetVerifyCodeConfig(l.ctx)
	if err != nil {
		l.Logger.Error("[GetGlobalConfigLogic] GetVerifyCodeConfig error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetVerifyCodeConfig error: %v", err.Error())
	}

	runtime := l.svcCtx.ConfigSnapshot()
	resp.Site = siteConfigFromRuntime(runtime.Site)
	resp.Subscribe = subscribeConfigFromRuntime(runtime.Subscribe)
	resp.Auth.Email = emailAuthConfigFromRuntime(runtime.Email)
	resp.Auth.Register = registerConfigFromRuntime(runtime.Register)
	resp.Verify = verifyConfigFromRuntime(runtime.Verify)
	resp.Invite = inviteConfigFromRuntime(runtime.Invite)
	tool.SystemConfigSliceReflectToStruct(currencyCfg, &resp.Currency)
	tool.SystemConfigSliceReflectToStruct(verifyCodeCfg, &resp.VerifyCode)

	if report.IsGatewayMode() {
		resp.Subscribe.SubscribePath = "/sub" + runtime.Subscribe.SubscribePath
	}

	// auth methods
	authMethods, err := l.svcCtx.AuthModel.FindAllForGlobalConfig(l.ctx)
	if err != nil {
		l.Logger.Error("[GetGlobalConfigLogic] FindAll error: ", logger.Field("error", err.Error()))
	}

	resp.OAuthMethods = globalAuthMethodsFromModels(authMethods)

	webAds, err := l.svcCtx.SystemModel.FindOneByKey(l.ctx, "WebAD")
	if err != nil {
		l.Logger.Error("[GetGlobalConfigLogic] FindOneByKey error: ", logger.Field("error", err.Error()), logger.Field("key", "WebAD"))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneByKey error: %v", err.Error())
	}
	// web ads config
	resp.WebAd = webAds.Value == "true"
	return
}

func globalAuthMethodsFromModels(authMethods []*modelAuth.Auth) []string {
	if len(authMethods) == 0 {
		return []string{}
	}
	methods := newAuthMethodCollector(len(authMethods))
	for _, method := range authMethods {
		if method == nil || method.Enabled == nil || !*method.Enabled {
			continue
		}
		if method.Method == "github" {
			methods.Add(method.Method)
		}
	}
	if methods.Len() == 0 {
		return []string{}
	}
	return methods.Items()
}

const authMethodLinearLimit = 8

type authMethodCollector struct {
	items   []string
	index   map[string]struct{}
	capHint int
}

func newAuthMethodCollector(capHint int) authMethodCollector {
	return authMethodCollector{capHint: capHint}
}

func (c *authMethodCollector) Add(method string) {
	if method == "" {
		return
	}
	if c.index != nil {
		if _, ok := c.index[method]; ok {
			return
		}
		c.index[method] = struct{}{}
		c.items = append(c.items, method)
		return
	}
	for _, existing := range c.items {
		if existing == method {
			return
		}
	}
	if c.items == nil {
		capHint := c.capHint
		if capHint <= 0 || capHint > authMethodLinearLimit+1 {
			capHint = authMethodLinearLimit + 1
		}
		c.items = make([]string, 0, capHint)
	}
	c.items = append(c.items, method)
	if len(c.items) > authMethodLinearLimit {
		c.index = make(map[string]struct{}, len(c.items))
		for _, existing := range c.items {
			c.index[existing] = struct{}{}
		}
	}
}

func (c authMethodCollector) Len() int {
	return len(c.items)
}

func (c authMethodCollector) Items() []string {
	if len(c.items) == 0 {
		return []string{}
	}
	return c.items
}
