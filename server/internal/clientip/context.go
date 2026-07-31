package clientip

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/neibniz/alpha/server/pkg/constant"
)

type Source string

const (
	SourceUnknown       Source = "unknown"
	SourceDirect        Source = "direct"
	SourceEdgeHeader    Source = "edge_header"
	SourceCloudflare    Source = "cloudflare"
	SourceXForwardedFor Source = "x_forwarded_for"
	SourceXRealIP       Source = "x_real_ip"
)

type Result struct {
	Address  netip.Addr
	Source   Source
	Verified bool
}

func (r Result) String() string {
	if !r.Address.IsValid() {
		return ""
	}
	return r.Address.Unmap().String()
}

type resultContextKey struct{}

func WithResult(ctx context.Context, result Result) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, resultContextKey{}, result)
	ctx = context.WithValue(ctx, constant.CtxKeyClientIP, result.String())
	ctx = context.WithValue(ctx, constant.CtxKeyClientIPSource, string(result.Source))
	return context.WithValue(ctx, constant.CtxKeyClientIPVerified, result.Verified)
}

func FromContext(ctx context.Context) (Result, bool) {
	if ctx == nil {
		return Result{}, false
	}
	result, ok := ctx.Value(resultContextKey{}).(Result)
	return result, ok && result.Address.IsValid()
}

func FromRequest(req *http.Request) Result {
	if req == nil {
		return Result{Source: SourceUnknown}
	}
	if result, ok := FromContext(req.Context()); ok {
		return result
	}
	return Result{
		Address:  parseRemoteAddress(req.RemoteAddr),
		Source:   SourceDirect,
		Verified: true,
	}
}

func Address(req *http.Request) string {
	return FromRequest(req).String()
}
