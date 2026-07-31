package user

import (
	"context"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const privateIdentityRevealTTL = time.Minute

type RevealUserPrivateIdentityLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevealUserPrivateIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevealUserPrivateIdentityLogic {
	return &RevealUserPrivateIdentityLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevealUserPrivateIdentityLogic) RevealUserPrivateIdentity(
	req *types.RevealUserPrivateIdentityRequest,
	clientIP string,
	userAgent string,
) (*types.RevealUserPrivateIdentityResponse, error) {
	if req == nil || req.UserId <= 0 || !validPrivateIdentityRevealReason(req.Reason) {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid private identity reveal request")
	}
	operator, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok || operator == nil || operator.Id <= 0 || operator.IsAdmin == nil || !*operator.IsAdmin {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "administrator identity is required")
	}

	now := time.Now().UTC()
	if _, err := l.svcCtx.UserModel.FindOneForAdminCheck(l.ctx, req.UserId); err != nil {
		_ = l.recordAudit(operator.Id, req.UserId, req.Reason, nil, false, now, clientIP, userAgent)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.UserNotExist), "user not found")
		}
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to query user")
	}

	methods, err := l.svcCtx.UserModel.RevealUserAuthMethods(l.ctx, req.UserId)
	if err != nil {
		_ = l.recordAudit(operator.Id, req.UserId, req.Reason, nil, false, now, clientIP, userAgent)
		l.Errorw("failed to reveal user private identities",
			logger.Field("operator_user_id", operator.Id),
			logger.Field("target_user_id", req.UserId),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to reveal private identities")
	}

	authMethods := userAuthMethodsFromModels(methods, false)
	authTypes := make([]string, 0, len(authMethods))
	email := ""
	for _, method := range authMethods {
		authTypes = append(authTypes, method.AuthType)
		if email == "" && method.AuthType == "email" {
			email = method.AuthIdentifier
		}
	}
	if err := l.recordAudit(operator.Id, req.UserId, req.Reason, authTypes, true, now, clientIP, userAgent); err != nil {
		l.Errorw("failed to audit user private identity reveal",
			logger.Field("operator_user_id", operator.Id),
			logger.Field("target_user_id", req.UserId),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "failed to record private data access")
	}

	return &types.RevealUserPrivateIdentityResponse{
		UserId:      req.UserId,
		Email:       email,
		AuthMethods: authMethods,
		RevealedAt:  now.UnixMilli(),
		ExpiresAt:   now.Add(privateIdentityRevealTTL).UnixMilli(),
	}, nil
}

func validPrivateIdentityRevealReason(reason string) bool {
	switch reason {
	case "customer_support", "account_verification", "security_incident", "billing_dispute", "compliance":
		return true
	default:
		return false
	}
}

func (l *RevealUserPrivateIdentityLogic) recordAudit(
	operatorUserID int64,
	targetUserID int64,
	reason string,
	authTypes []string,
	success bool,
	now time.Time,
	clientIP string,
	userAgent string,
) error {
	content, err := (&modelLog.PrivateDataReveal{
		OperatorUserId: operatorUserID,
		TargetUserId:   targetUserID,
		Reason:         reason,
		AuthTypes:      authTypes,
		Timestamp:      now.UnixMilli(),
	}).Marshal()
	if err != nil {
		return err
	}
	return l.svcCtx.LogModel.Insert(l.ctx, &modelLog.SystemLog{
		Type:      modelLog.TypePrivateDataReveal.Uint8(),
		Date:      now.Format(time.DateOnly),
		ObjectID:  targetUserID,
		IP:        clientIP,
		UserAgent: userAgent,
		Success:   success,
		EventAt:   now.UnixMilli(),
		Content:   string(content),
	})
}
