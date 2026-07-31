package authMethod

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/authconfig"
	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/email"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateAuthMethodConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var (
	authMethodRawNull        = []byte("null")
	authMethodRawEmptyObject = json.RawMessage("{}")
)

// Update auth method config
func NewUpdateAuthMethodConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthMethodConfigLogic {
	return &UpdateAuthMethodConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAuthMethodConfigLogic) UpdateAuthMethodConfig(req *types.UpdateAuthMethodConfigRequest) (resp *types.AuthMethodConfig, err error) {
	if isRemovedAuthMethod(req.Method) {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method is no longer supported")
	}
	includeConfig := req.Config != nil || req.Method == "github"
	method, err := l.svcCtx.AuthModel.FindOneByMethodForAdmin(l.ctx, req.Method, includeConfig)
	if err != nil {
		l.Errorw("find one by method failed", logger.Field("method", req.Method), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find one by method failed: %v", err.Error())
	}

	update := auth.AuthFieldsUpdate{}
	if req.Enabled != nil {
		method.Enabled = req.Enabled
		update.Enabled = req.Enabled
	}
	if req.Method == "github" {
		current, err := authconfig.LoadGitHub(method.Config, l.svcCtx.Privacy)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "load github config failed: %v", err)
		}
		if req.Config != nil {
			incoming, err := authconfig.ParseGitHub(string(req.Config))
			if err != nil {
				return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "normalize github config failed: %v", err)
			}
			if incoming.ClientSecret == "" {
				incoming.ClientSecret = current.ClientSecret
			}
			current = incoming
		}
		enabled := method.Enabled != nil && *method.Enabled
		if enabled {
			if err := authconfig.ValidateGitHub(current); err != nil {
				return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "invalid github config: %v", err)
			}
		}
		stored, err := authconfig.StoreGitHub(current, l.svcCtx.Privacy)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "store github config failed: %v", err)
		}
		method.Config = stored
		update.Config = &method.Config
	} else if req.Config != nil {
		config, err := normalizeAuthMethodConfig(req.Method, req.Config)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "normalize config failed: %v", err.Error())
		}
		method.Config = config
		update.Config = &method.Config
	}
	err = l.svcCtx.AuthModel.UpdateFields(l.ctx, method, update)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "update auth method failed: %v", err.Error())
	}

	config := authMethodConfigFromModel(method)
	resp = &config
	if (req.Config != nil || req.Method == "github") && method.Config != "" {
		config, err := authMethodConfigForAdmin(method)
		if err != nil {
			l.Errorw("invalid auth method config", logger.Field("method", req.Method), logger.Field("error", err.Error()))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "invalid auth method config: %v", err.Error())
		}
		resp = &config
	}
	// update global config
	defer l.UpdateGlobal(method.Method)
	return
}

func (l *UpdateAuthMethodConfigLogic) UpdateGlobal(method string) {
	if method == "email" {
		initialize.Email(l.svcCtx)
	}
}

func isRemovedAuthMethod(method string) bool {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "mobile", "device":
		return true
	default:
		return false
	}
}

func normalizeAuthMethodConfig(method string, value json.RawMessage) (string, error) {
	raw, err := authMethodConfigRaw(method, value)
	if err != nil {
		return "", err
	}

	switch method {
	case "email":
		return normalizeEmailAuthConfig(raw)
	case "apple":
		return normalizeSimpleAuthConfig[auth.AppleAuthConfig](raw)
	case "google":
		return normalizeSimpleAuthConfig[auth.GoogleAuthConfig](raw)
	case "github":
		return normalizeSimpleAuthConfig[auth.GithubAuthConfig](raw)
	case "facebook":
		return normalizeSimpleAuthConfig[auth.FacebookAuthConfig](raw)
	case "telegram":
		return normalizeSimpleAuthConfig[auth.TelegramAuthConfig](raw)
	default:
		return "", errors.Errorf("unsupported auth method %q", method)
	}
}

func authMethodConfigRaw(method string, value json.RawMessage) ([]byte, error) {
	raw := bytes.TrimSpace(value)
	if len(raw) == 0 {
		return defaultAuthMethodConfigRaw(method)
	}
	if !json.Valid(raw) {
		return nil, errors.New("invalid config json")
	}
	return raw, nil
}

func defaultAuthMethodConfigRaw(method string) ([]byte, error) {
	config, ok := initializePlatformConfig(method)
	if !ok {
		return nil, errors.Errorf("unsupported auth method %q", method)
	}
	return []byte(config), nil
}

type emailAuthConfigPayload struct {
	Platform                   string          `json:"platform"`
	PlatformConfig             json.RawMessage `json:"platform_config"`
	EnableVerify               bool            `json:"enable_verify"`
	EnableNotify               bool            `json:"enable_notify"`
	EnableDomainSuffix         bool            `json:"enable_domain_suffix"`
	DomainSuffixList           string          `json:"domain_suffix_list"`
	VerifyEmailTemplate        string          `json:"verify_email_template"`
	ExpirationEmailTemplate    string          `json:"expiration_email_template"`
	MaintenanceEmailTemplate   string          `json:"maintenance_email_template"`
	TrafficExceedEmailTemplate string          `json:"traffic_exceed_email_template"`
}

func normalizeEmailAuthConfig(raw []byte) (string, error) {
	var payload emailAuthConfigPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	if payload.Platform == "" {
		payload.Platform = email.SMTP.String()
	}

	platformConfig, err := normalizeAuthPlatformConfigRaw(payload.Platform, payload.PlatformConfig)
	if err != nil {
		return "", err
	}

	config := auth.EmailAuthConfig{
		Platform:                   payload.Platform,
		PlatformConfig:             platformConfig,
		EnableVerify:               payload.EnableVerify,
		EnableNotify:               payload.EnableNotify,
		EnableDomainSuffix:         payload.EnableDomainSuffix,
		DomainSuffixList:           payload.DomainSuffixList,
		VerifyEmailTemplate:        payload.VerifyEmailTemplate,
		ExpirationEmailTemplate:    payload.ExpirationEmailTemplate,
		MaintenanceEmailTemplate:   payload.MaintenanceEmailTemplate,
		TrafficExceedEmailTemplate: payload.TrafficExceedEmailTemplate,
	}
	return config.Marshal(), nil
}

func normalizeSimpleAuthConfig[T any](raw []byte) (string, error) {
	var config T
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", err
	}
	data, err := json.Marshal(&config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeAuthPlatformConfigRaw(platform string, raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, authMethodRawNull) {
		raw = authMethodRawEmptyObject
	}

	switch platform {
	case email.SMTP.String():
		return normalizePlatformConfigRaw[auth.SMTPConfig](raw)
	default:
		return nil, errors.Errorf("invalid platform %q", platform)
	}
}

func normalizePlatformConfigRaw[T any](raw json.RawMessage) (json.RawMessage, error) {
	var config T
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	data, err := json.Marshal(&config)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func initializePlatformConfig(platform string) (string, bool) {
	switch platform {
	case "email":
		return new(auth.EmailAuthConfig).Marshal(), true
	case "apple":
		return new(auth.AppleAuthConfig).Marshal(), true
	case "google":
		return new(auth.GoogleAuthConfig).Marshal(), true
	case "github":
		return new(auth.GithubAuthConfig).Marshal(), true
	case "facebook":
		return new(auth.FacebookAuthConfig).Marshal(), true
	case "telegram":
		return new(auth.TelegramAuthConfig).Marshal(), true
	}
	return "", false
}
