package portal

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/constant"
)

func TestAuthenticatedOrderOwner(t *testing.T) {
	orderInfo := &order.Order{UserId: 42}
	owner := &user.User{Id: 42}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, owner)

	got, ok := authenticatedOrderOwner(ctx, orderInfo)
	if !ok || got != owner {
		t.Fatalf("authenticated order owner = %#v/%v, want owner/true", got, ok)
	}

	otherCtx := context.WithValue(context.Background(), constant.CtxKeyUser, &user.User{Id: 7})
	if _, ok := authenticatedOrderOwner(otherCtx, orderInfo); ok {
		t.Fatal("different user must not be accepted as order owner")
	}
	if _, ok := authenticatedOrderOwner(context.Background(), orderInfo); ok {
		t.Fatal("missing authenticated user must not be accepted")
	}
	if _, ok := authenticatedOrderOwner(ctx, nil); ok {
		t.Fatal("nil order must not be accepted")
	}
}
