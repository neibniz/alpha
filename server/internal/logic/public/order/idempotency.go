package order

import (
	"context"
	"regexp"
	"strings"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var orderRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{15,63}$`)

func normalizeOrderRequestID(value string) (string, error) {
	requestID := strings.TrimSpace(value)
	if requestID == "" {
		return "", nil
	}
	if !orderRequestIDPattern.MatchString(requestID) {
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid request_id")
	}
	return requestID, nil
}

func findIdempotentOrderNo(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, orderType uint8, requestID string) (string, bool, error) {
	if requestID == "" {
		return "", false, nil
	}
	orderInfo, err := svcCtx.OrderModel.FindOneByRequestID(ctx, userID, orderType, requestID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return orderInfo.OrderNo, true, nil
}
