package system

import (
	"context"
	"net"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var subscribePathPattern = regexp.MustCompile(`^/[A-Za-z0-9._~-]+(?:/[A-Za-z0-9._~-]+)*$`)
var subscribeDomainLabelPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
var subscribeIPv4CandidatePattern = regexp.MustCompile(`^\d+(?:\.\d+){3}$`)

func validateSubscribePath(path string) error {
	if path == "" {
		return nil
	}
	if len(path) > 160 || !subscribePathPattern.MatchString(path) {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid static subscription path")
	}
	return nil
}

func normalizeSubscribeConfig(req *types.SubscribeConfig) error {
	req.SubscribePath = strings.TrimSpace(req.SubscribePath)
	if err := validateSubscribePath(req.SubscribePath); err != nil {
		return err
	}

	domains, err := normalizeConfigLines(req.SubscribeDomain, 8192, validateSubscribeDomain)
	if err != nil {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid subscription domain list")
	}
	userAgents, err := normalizeConfigLines(req.UserAgentList, 16_384, validateSubscribeUserAgent)
	if err != nil {
		return xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid subscription user-agent list")
	}
	req.SubscribeDomain = domains
	req.UserAgentList = userAgents
	return nil
}

func normalizeConfigLines(value string, maxBytes int, validate func(string) bool) (string, error) {
	if len(value) > maxBytes {
		return "", xerr.NewErrCode(xerr.InvalidParams)
	}
	seen := make(map[string]struct{})
	lines := make([]string, 0)
	for _, raw := range strings.Split(value, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if !validate(line) {
			return "", xerr.NewErrCode(xerr.InvalidParams)
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

func validateSubscribeDomain(value string) bool {
	if len(value) == 0 || len(value) > 253 || strings.Contains(value, "*") {
		return false
	}
	if strings.ContainsAny(value, ":/@?#\\") || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return false
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.To4() != nil
	}
	if subscribeIPv4CandidatePattern.MatchString(value) {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if !subscribeDomainLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

func validateSubscribeUserAgent(value string) bool {
	count := utf8.RuneCountInString(value)
	return count > 0 && count <= 255 && strings.IndexFunc(value, unicode.IsControl) < 0
}

type UpdateSubscribeConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSubscribeConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSubscribeConfigLogic {
	return &UpdateSubscribeConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSubscribeConfigLogic) UpdateSubscribeConfig(req *types.SubscribeConfig) error {
	if err := normalizeSubscribeConfig(req); err != nil {
		return errors.Wrap(err, "validate subscribe config")
	}

	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "subscribe", values); err != nil {
			return err
		}
		return l.svcCtx.Redis.Del(l.ctx, config.SubscribeConfigKey, config.GlobalConfigKey).Err()
	})

	if err != nil {
		l.Errorw("[UpdateSubscribeConfigLogic] update subscribe config error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update subscribe config error: %v", err)
	}

	if l.svcCtx.ConfigSnapshot().Subscribe.SubscribePath != req.SubscribePath {
		go func(svc *svc.ServiceContext) {
			if restartErr := svc.Restart(); restartErr != nil {
				l.Errorw("[UpdateSubscribeConfigLogic] restart error: ", logger.Field("error", restartErr.Error()))
			}
		}(l.svcCtx)
		return nil
	}

	initialize.Subscribe(l.svcCtx)
	return nil
}
