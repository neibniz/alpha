package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/svc"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *logBodyBuffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	_, _ = r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r responseBodyWriter) WriteString(s string) (int, error) {
	_, _ = r.body.WriteString(s)
	return r.ResponseWriter.WriteString(s)
}

const (
	maxLoggedBodyBytes  = 16 * 1024
	defaultMaxBodyBytes = int64(8 << 20)
	maskedLogValue      = "***"
)

var sensitiveLogFields = map[string]struct{}{
	"accesstoken":           {},
	"apikey":                {},
	"authorization":         {},
	"authidentifier":        {},
	"authidentifierdisplay": {},
	"clientsecret":          {},
	"clientip":              {},
	"destinationip":         {},
	"deviceid":              {},
	"email":                 {},
	"identifier":            {},
	"ip":                    {},
	"loginip":               {},
	"logoutip":              {},
	"mobile":                {},
	"newpassword":           {},
	"oldpassword":           {},
	"password":              {},
	"phone":                 {},
	"pin":                   {},
	"privatekey":            {},
	"recipient":             {},
	"registerip":            {},
	"refreshtoken":          {},
	"secret":                {},
	"secretkey":             {},
	"session":               {},
	"sourceip":              {},
	"subscribetoken":        {},
	"telephone":             {},
	"to":                    {},
	"token":                 {},
	"otp":                   {},
	"useragent":             {},
	"uuid":                  {},
	"verificationcode":      {},
	"verifycode":            {},
	"webhooksecret":         {},
}

func LoggerMiddleware(svc *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		// get response body
		w := &responseBodyWriter{body: newLogBodyBuffer(maxLoggedBodyBytes), ResponseWriter: c.Writer}
		c.Writer = w
		// get request body
		requestBody := newLogBodyBuffer(maxLoggedBodyBytes)
		if c.Request.Body != nil {
			maxBodyBytes := svc.Config.HTTP.MaxBodyBytes
			if maxBodyBytes <= 0 {
				maxBodyBytes = defaultMaxBodyBytes
			}
			limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
			c.Request.Body = &logBodyReadCloser{ReadCloser: limited, body: requestBody}
		}
		// start time
		start := time.Now()
		c.Next()
		// Start recording logs
		cost := time.Since(start)
		responseStatus := c.Writer.Status()

		host := privacy.SanitizeHost(c.Request.Host)

		logs := []logger.LogField{
			{
				Key:   "status",
				Value: responseStatus,
			},
			{
				Key:   "request",
				Value: c.Request.Method + " " + host + privacy.SanitizeURL(c.Request.URL.String()),
			},
			{
				Key:   "query",
				Value: privacy.SanitizeQuery(c.Request.URL.RawQuery),
			},
			{
				Key:   "ip",
				Value: privacy.MaskIP(clientip.Address(c.Request)),
			},
			{
				Key:   "user-agent",
				Value: privacy.MaskUserAgent(c.Request.UserAgent()),
			},
		}
		if c.Errors.Last() != nil {
			var e *xerr.CodeError
			var errMessage string
			if errors.As(c.Errors.Last().Err, &e) {
				errMessage = e.GetErrMsg()
			} else {
				errMessage = c.Errors.Last().Error()
			}
			logs = append(logs, logger.Field("error", errMessage))
		}
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			logs = append(logs,
				logger.Field("request_body_bytes", requestBody.total),
				logger.Field("response_body_bytes", w.body.total),
			)
		}
		logs = append(logs, logger.Field("duration", cost))
		if responseStatus >= 500 && responseStatus <= 599 {
			logger.WithContext(c.Request.Context()).Errorw("HTTP Error", logs...)
		} else {
			logger.WithContext(c.Request.Context()).Infow("HTTP Request", logs...)
		}

		if responseStatus == 404 {
			logger.WithContext(c.Request.Context()).Debugf("404 Not Found: Host:%s Path:%s IsPanDomain:%v", host, c.Request.URL.Path, svc.ConfigSnapshot().Subscribe.PanDomain)
		}
	}
}

type logBodyReadCloser struct {
	io.ReadCloser
	body *logBodyBuffer
}

func (r *logBodyReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		_, _ = r.body.Write(p[:n])
	}
	return n, err
}

func maskSensitiveFields(data []byte, fieldsToMask []string) []byte {
	original := data
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return data
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()

	var jsonData any
	if err := decoder.Decode(&jsonData); err != nil {
		return original
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return original
	}
	if !maskSensitiveValue(jsonData, fieldsToMask) {
		return original
	}

	maskedData, err := json.Marshal(jsonData)
	if err != nil {
		return original
	}
	return maskedData
}

func maskSensitiveValue(value any, extraFields []string) bool {
	switch data := value.(type) {
	case map[string]any:
		changed := false
		for key, item := range data {
			if isSensitiveLogField(key, extraFields) {
				data[key] = maskedLogValue
				changed = true
				continue
			}
			if maskSensitiveValue(item, extraFields) {
				changed = true
			}
		}
		return changed
	case []any:
		changed := false
		for _, item := range data {
			if maskSensitiveValue(item, extraFields) {
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

func isSensitiveLogField(key string, extraFields []string) bool {
	normalized := normalizeLogField(key)
	if _, ok := sensitiveLogFields[normalized]; ok {
		return true
	}
	for _, field := range extraFields {
		if normalized == normalizeLogField(field) {
			return true
		}
	}
	return false
}

func normalizeLogField(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}

func logBodyString(data []byte, fieldsToMask []string, limit int) string {
	if limit > 0 && len(data) > limit {
		return omittedBodyMessage(len(data), limit)
	}
	return string(maskSensitiveFields(data, fieldsToMask))
}

func omittedBodyMessage(size int, limit int) string {
	return fmt.Sprintf("<body omitted: %d bytes exceeds %d byte log limit>", size, limit)
}

type logBodyBuffer struct {
	bytes.Buffer
	limit int
	total int
}

func newLogBodyBuffer(limit int) *logBodyBuffer {
	return &logBodyBuffer{limit: limit}
}

func (b *logBodyBuffer) Write(p []byte) (int, error) {
	b.total += len(p)
	if b.limit <= 0 || b.Buffer.Len() >= b.limit {
		return len(p), nil
	}

	remaining := b.limit - b.Buffer.Len()
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		return len(p), nil
	}
	_, _ = b.Buffer.Write(p)
	return len(p), nil
}

func (b *logBodyBuffer) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}

func (b *logBodyBuffer) LogString(fieldsToMask []string) string {
	if b.limit > 0 && b.total > b.limit {
		return omittedBodyMessage(b.total, b.limit)
	}
	return logBodyString(b.Buffer.Bytes(), fieldsToMask, b.limit)
}
