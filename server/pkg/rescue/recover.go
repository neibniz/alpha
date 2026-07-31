package rescue

import (
	"context"
	"runtime/debug"

	"github.com/neibniz/alpha/server/pkg/logger"
)

// Recover is used with defer to do cleanup on panics.
// Use it like:
//
//	defer Recover(func() {})
func Recover(cleanups ...func()) {
	for _, cleanup := range cleanups {
		cleanup()
	}

	if p := recover(); p != nil {
		logger.Errorf("%+v\n%s", p, debug.Stack())
	}
}

// RecoverCtx is used with defer to do cleanup on panics.
func RecoverCtx(ctx context.Context, cleanups ...func()) {
	for _, cleanup := range cleanups {
		cleanup()
	}

	if p := recover(); p != nil {
		logger.WithContext(ctx).Errorf("%+v\n%s", p, debug.Stack())
	}
}
