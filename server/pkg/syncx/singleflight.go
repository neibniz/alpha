package syncx

import (
	"context"
	"sync"
)

type (
	// SingleFlight lets the concurrent calls with the same key to share the call result.
	// For example, A called F, before it's done, B called F. Then B would not execute F,
	// and shared the result returned by F which called by A.
	// The calls with the same key are dependent, concurrent calls share the returned values.
	// A ------->calls F with key<------------------->returns val
	// B --------------------->calls F with key------>returns val
	SingleFlight interface {
		Do(key string, fn func() (any, error)) (any, error)
		DoEx(key string, fn func() (any, error)) (any, bool, error)
		DoCtx(ctx context.Context, key string, fn func() (any, error)) (any, error)
		DoExCtx(ctx context.Context, key string, fn func() (any, error)) (any, bool, error)
	}

	call struct {
		done chan struct{}
		val  any
		err  error
	}

	flightGroup struct {
		calls map[string]*call
		lock  sync.Mutex
	}
)

// NewSingleFlight returns a SingleFlight.
func NewSingleFlight() SingleFlight {
	return &flightGroup{
		calls: make(map[string]*call),
	}
}

func (g *flightGroup) Do(key string, fn func() (any, error)) (any, error) {
	return g.DoCtx(context.Background(), key, fn)
}

func (g *flightGroup) DoEx(key string, fn func() (any, error)) (val any, fresh bool, err error) {
	return g.DoExCtx(context.Background(), key, fn)
}

func (g *flightGroup) DoCtx(ctx context.Context, key string, fn func() (any, error)) (any, error) {
	val, _, err := g.DoExCtx(ctx, key, fn)
	return val, err
}

func (g *flightGroup) DoExCtx(
	ctx context.Context,
	key string,
	fn func() (any, error),
) (val any, fresh bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	c, fresh := g.createCall(key)
	if !fresh {
		select {
		case <-c.done:
			return c.val, false, c.err
		default:
		}
		select {
		case <-c.done:
			return c.val, false, c.err
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}

	g.makeCall(c, key, fn)
	return c.val, true, c.err
}

func (g *flightGroup) createCall(key string) (c *call, fresh bool) {
	g.lock.Lock()
	if c, ok := g.calls[key]; ok {
		g.lock.Unlock()
		return c, false
	}

	c = &call{done: make(chan struct{})}
	g.calls[key] = c
	g.lock.Unlock()

	return c, true
}

func (g *flightGroup) makeCall(c *call, key string, fn func() (any, error)) {
	defer func() {
		g.lock.Lock()
		delete(g.calls, key)
		g.lock.Unlock()
		close(c.done)
	}()

	c.val, c.err = fn()
}
