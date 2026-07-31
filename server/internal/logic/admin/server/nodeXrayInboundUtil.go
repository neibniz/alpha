package server

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

func nodeRequestInboundAlias(inboundAlias string, protocol string) string {
	if strings.TrimSpace(inboundAlias) != "" {
		return strings.TrimSpace(inboundAlias)
	}
	return strings.TrimSpace(protocol)
}

func validateNodeInboundAlias(ctx context.Context, svcCtx *svc.ServiceContext, serverID int64, alias string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "inbound is required"), "inbound is required")
	}
	bindings, err := svcCtx.NodeModel.QueryServerXrayTemplates(ctx, serverID, true)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server xray templates failed: %v", err)
	}
	for _, binding := range bindings {
		if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
			continue
		}
		if xraytemplate.BindingMatchesInbound(binding, alias) {
			return nil
		}
	}
	return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "inbound is not bound to server"), "inbound %q is not bound to server %d", alias, serverID)
}
