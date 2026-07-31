package logger

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm/logger"
)

type GormLogger struct {
}

const TAG = "[GORM]"

func (l *GormLogger) LogMode(logger.LogLevel) logger.Interface {
	var sysLevel string
	switch logLevel {
	case DebugLevel:
		sysLevel = "debug"
	case InfoLevel:
		sysLevel = "info"
	case SevereLevel:
		sysLevel = "severe"
	case disableLevel:
		sysLevel = "disable"
	default:
		sysLevel = "unknown"
	}
	Infof("%s System Log Level is %s", TAG, sysLevel)
	return l
}

func (l *GormLogger) Info(ctx context.Context, str string, args ...interface{}) {
	WithContext(ctx).WithCallerSkip(2).Info(TAG, " Info: ", fmt.Sprintf(str, args...))
}

func (l *GormLogger) Warn(ctx context.Context, str string, args ...interface{}) {
	WithContext(ctx).WithCallerSkip(2).Info(TAG, " Warn: ", fmt.Sprintf(str, args...))
}

func (l *GormLogger) Error(ctx context.Context, str string, args ...interface{}) {
	WithContext(ctx).WithCallerSkip(2).Error(TAG, " Error: ", fmt.Sprintf(str, args...))
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	_, rowsAffected := fc()
	fields := []LogField{
		{
			Key:   "rows",
			Value: rowsAffected,
		},
	}
	if err != nil {
		fields = append(fields, LogField{
			Key:   "error",
			Value: err.Error(),
		})
		WithContext(ctx).WithCallerSkip(6).WithDuration(time.Since(begin)).Errorw(TAG, fields...)
	} else {
		WithContext(ctx).WithCallerSkip(6).WithDuration(time.Since(begin)).Infow(TAG+" SQL Executed", fields...)
	}
}
