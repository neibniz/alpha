package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
)

const systemLogKnownFieldCount = 8

type GetSystemLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetSystemLogLogic Get System Log
func NewGetSystemLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSystemLogLogic {
	return &GetSystemLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSystemLogLogic) GetSystemLog() (resp *types.LogResponse, err error) {
	lines, err := logger.ReadLastNLines(l.svcCtx.Config.Logger.Path, 50)
	if err != nil {
		l.Error(err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "get system log error: %v", err.Error())
	}

	return &types.LogResponse{
		List: systemLogLinesToEntries(lines, l.Logger),
	}, nil
}

func systemLogLinesToEntries(lines []string, logr logger.Logger) []types.SystemLogEntry {
	list := viewutil.NewRowsCollector[types.SystemLogEntry](len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, ok := systemLogLineToEntry(line)
		if !ok {
			if logr != nil {
				logr.Errorf("[GetSystemLog] invalid json log line")
			}
			continue
		}
		list.Add(entry)
	}
	return list.Rows()
}

func systemLogLineToEntry(line string) (types.SystemLogEntry, bool) {
	fields := make(map[string]json.RawMessage, systemLogKnownFieldCount+2)
	if err := json.Unmarshal([]byte(line), &fields); err != nil || len(fields) == 0 {
		return types.SystemLogEntry{}, false
	}

	entry := types.SystemLogEntry{
		Timestamp: systemLogString(fields, "timestamp"),
		Level:     systemLogString(fields, "level"),
		Caller:    systemLogString(fields, "caller"),
		Content:   systemLogRaw(fields, "content"),
		Duration:  systemLogString(fields, "duration"),
		Trace:     systemLogString(fields, "trace"),
		Span:      systemLogString(fields, "span"),
		Truncated: systemLogBool(fields, "truncated"),
		Fields:    systemLogExtraFields(fields),
	}
	return entry, true
}

func systemLogString(fields map[string]json.RawMessage, key string) string {
	raw, ok := popSystemLogField(fields, key)
	if !ok {
		return ""
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return ""
	}
	value, err := strconv.Unquote(string(trimmed))
	if err != nil {
		return ""
	}
	return value
}

func systemLogRaw(fields map[string]json.RawMessage, key string) json.RawMessage {
	raw, ok := popSystemLogField(fields, key)
	if !ok || len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	return privacy.MaskLogFieldJSON(key, raw)
}

func systemLogBool(fields map[string]json.RawMessage, key string) bool {
	raw, ok := popSystemLogField(fields, key)
	if !ok {
		return false
	}
	return bytes.Equal(bytes.TrimSpace(raw), []byte("true"))
}

func popSystemLogField(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, ok := fields[key]
	if ok {
		delete(fields, key)
	}
	return raw, ok
}

func systemLogExtraFields(fields map[string]json.RawMessage) json.RawMessage {
	if len(fields) == 0 {
		return nil
	}
	for key, raw := range fields {
		fields[key] = privacy.MaskLogFieldJSON(key, raw)
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return nil
	}
	return data
}
