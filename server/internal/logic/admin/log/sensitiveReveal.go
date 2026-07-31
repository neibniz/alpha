package log

import (
	"context"
	"strings"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type sensitiveLogReveal struct {
	enabled  bool
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	operator *modelUser.User
	scope    string
	reason   string
}

func prepareSensitiveLogReveal(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	enabled bool,
	reason string,
	scope string,
) (*sensitiveLogReveal, error) {
	reason = strings.TrimSpace(reason)
	reveal := &sensitiveLogReveal{enabled: enabled, ctx: ctx, svcCtx: svcCtx, scope: scope, reason: reason}
	if !enabled {
		return reveal, nil
	}
	if svcCtx == nil || svcCtx.Privacy == nil || !svcCtx.Privacy.Enabled() {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "sensitive log reveal requires privacy encryption")
	}
	if !validSensitiveLogRevealReason(reason) {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid sensitive log reveal reason")
	}
	operator, ok := ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok || operator == nil || operator.Id <= 0 || operator.IsAdmin == nil || !*operator.IsAdmin {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "administrator identity is required")
	}
	reveal.operator = operator
	return reveal, nil
}

func validSensitiveLogRevealReason(reason string) bool {
	switch reason {
	case "customer_support", "account_verification", "security_incident", "billing_dispute", "compliance":
		return true
	default:
		return false
	}
}

func (r *sensitiveLogReveal) audit(recordCount int) error {
	if r == nil || !r.enabled {
		return nil
	}
	now := time.Now().UTC()
	content, err := (&modelLog.SensitiveLogReveal{
		OperatorUserId: r.operator.Id,
		Scope:          r.scope,
		Reason:         r.reason,
		RecordCount:    recordCount,
		Timestamp:      now.UnixMilli(),
	}).Marshal()
	if err != nil {
		return err
	}
	clientIP, _ := r.ctx.Value(constant.CtxKeyClientIP).(string)
	userAgent, _ := r.ctx.Value(constant.CtxKeyUserAgent).(string)
	return r.svcCtx.LogModel.Insert(r.ctx, &modelLog.SystemLog{
		Type:      modelLog.TypeSensitiveLogReveal.Uint8(),
		Date:      now.Format(time.DateOnly),
		ObjectID:  r.operator.Id,
		IP:        clientIP,
		UserAgent: userAgent,
		Success:   true,
		EventAt:   now.UnixMilli(),
		Content:   string(content),
	})
}
