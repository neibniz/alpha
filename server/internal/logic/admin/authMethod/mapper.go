package authMethod

import (
	"errors"
	"strings"

	"github.com/neibniz/alpha/server/internal/authconfig"
	"github.com/neibniz/alpha/server/internal/jsonview"
	modelAuth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func authMethodConfigFromModel(method *modelAuth.Auth) types.AuthMethodConfig {
	if method == nil {
		return types.AuthMethodConfig{}
	}
	resp := types.AuthMethodConfig{
		Id:     method.Id,
		Method: method.Method,
	}
	if method.Enabled != nil {
		resp.Enabled = *method.Enabled
	}
	return resp
}

func authMethodSummaryFromModel(method *modelAuth.Auth) types.AuthMethodSummary {
	if method == nil {
		return types.AuthMethodSummary{}
	}
	resp := types.AuthMethodSummary{
		Id:     method.Id,
		Method: method.Method,
	}
	if method.Enabled != nil {
		resp.Enabled = *method.Enabled
	}
	return resp
}

func authMethodSummaryRows(methods []*modelAuth.Auth) []types.AuthMethodSummary {
	if len(methods) == 0 {
		return []types.AuthMethodSummary{}
	}
	collector := viewutil.NewRowsCollector[types.AuthMethodSummary](len(methods))
	for _, method := range methods {
		if method == nil {
			continue
		}
		collector.Add(authMethodSummaryFromModel(method))
	}
	return collector.Rows()
}

func authMethodConfigWithRawConfig(method *modelAuth.Auth) (types.AuthMethodConfig, error) {
	resp := authMethodConfigFromModel(method)
	if method == nil || strings.TrimSpace(method.Config) == "" {
		return resp, nil
	}
	raw, ok := jsonview.RawObject(method.Config)
	if !ok {
		return resp, errors.New("invalid auth method config json")
	}
	resp.Config = raw
	return resp, nil
}

func authMethodConfigForAdmin(method *modelAuth.Auth) (types.AuthMethodConfig, error) {
	resp, err := authMethodConfigWithRawConfig(method)
	if err != nil || method == nil || method.Method != "github" {
		return resp, err
	}
	resp.Config, err = authconfig.RedactGitHub(method.Config)
	return resp, err
}
