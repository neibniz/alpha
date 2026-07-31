package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/constant"
)

func authenticatedOrderOwner(ctx context.Context, orderInfo *order.Order) (*user.User, bool) {
	currentUser, ok := ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok || currentUser == nil || currentUser.Id == 0 || orderInfo == nil {
		return nil, false
	}
	return currentUser, orderInfo.UserId == currentUser.Id
}
