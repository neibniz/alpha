package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/trace"
)

// statusByWriter returns a span status code and message for an HTTP status code
// value returned by a server. Status codes in the 400-499 range are not
// returned as errors.
func statusByWriter(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 500 {
		return codes.Error, ""
	}
	return codes.Unset, ""
}

func requestAttributes(req *http.Request) []attribute.KeyValue {
	if req == nil {
		return nil
	}
	protocolName, protocolVersion, found := strings.Cut(req.Proto, "/")
	if !found {
		protocolName = req.Proto
		protocolVersion = ""
	}
	remoteHost, remotePort, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		remoteHost = req.RemoteAddr
		remotePort = ""
	}
	clientAddress := clientip.FromRequest(req).String()
	if clientAddress == "" {
		clientAddress = remoteHost
	}
	requestURL := ""
	requestScheme := ""
	requestFragment := ""
	requestPath := ""
	requestQuery := ""
	if req.URL != nil {
		requestURL = privacy.SanitizeURL(req.URL.String())
		requestScheme = req.URL.Scheme
		requestFragment = privacy.MaskIdentifier(req.URL.Fragment)
		requestPath = privacy.SanitizeURL(req.URL.Path)
		requestQuery = privacy.SanitizeQuery(req.URL.RawQuery)
	}

	return []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(req.Method),
		semconv.HTTPUserAgentKey.String(privacy.MaskUserAgent(req.UserAgent())),
		semconv.HTTPRequestContentLengthKey.Int64(req.ContentLength),

		semconv.URLFullKey.String(requestURL),
		semconv.URLSchemeKey.String(requestScheme),
		semconv.URLFragmentKey.String(requestFragment),
		semconv.URLPathKey.String(requestPath),
		semconv.URLQueryKey.String(requestQuery),

		semconv.NetworkProtocolNameKey.String(strings.ToLower(protocolName)),
		semconv.NetworkProtocolVersionKey.String(protocolVersion),

		semconv.ClientAddressKey.String(privacy.MaskIP(clientAddress)),
		semconv.ClientPortKey.String(remotePort),
	}
}

func TraceMiddleware(_ *svc.ServiceContext) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tracer := trace.TracerFromContext(ctx)

		spanName := c.FullPath()
		method := c.Request.Method

		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("%s %s", method, spanName),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer span.End()

		requestId := trace.TraceIDFromContext(ctx)

		c.Header(trace.RequestIdKey, requestId)

		span.SetAttributes(requestAttributes(c.Request)...)
		span.SetAttributes(
			attribute.String("http.request_id", requestId),
			semconv.HTTPRouteKey.String(c.FullPath()),
		)
		// context with request host
		ctx = context.WithValue(ctx, constant.CtxKeyRequestHost, c.Request.Host)
		// restructure context
		c.Request = c.Request.WithContext(ctx)
		client := clientip.FromRequest(c.Request)
		span.SetAttributes(
			attribute.String("alpha.client_ip.source", string(client.Source)),
			attribute.Bool("alpha.client_ip.verified", client.Verified),
		)

		c.Next()

		// handle response related attributes
		status := c.Writer.Status()
		span.SetStatus(statusByWriter(status))
		if status > 0 {
			span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(status))
		}
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, "request handler error")
			for _, err := range c.Errors {
				span.RecordError(fmt.Errorf("request handler error (%T)", err.Err))
			}
		}

		span.SetAttributes(semconv.HTTPResponseBodySizeKey.Int(c.Writer.Size()))
	}
}
