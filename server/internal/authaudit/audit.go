package authaudit

import (
	"context"
	"errors"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
)

var ErrInvalidEvent = errors.New("invalid authentication audit event")

type Inserter interface {
	Insert(context.Context, *modelLog.SystemLog) error
}

type LoginEvent struct {
	UserID    int64
	Method    string
	IP        string
	UserAgent string
	Success   bool
	At        time.Time
}

type RegisterEvent struct {
	UserID     int64
	Method     string
	Identifier string
	IP         string
	UserAgent  string
	At         time.Time
}

func RecordLogin(ctx context.Context, model Inserter, event LoginEvent) error {
	if model == nil || event.UserID <= 0 || event.Method == "" {
		return ErrInvalidEvent
	}
	event.At = eventTime(event.At)
	content, err := (&modelLog.Login{
		Method:    event.Method,
		LoginIP:   event.IP,
		UserAgent: event.UserAgent,
		Success:   event.Success,
		Timestamp: event.At.UnixMilli(),
	}).Marshal()
	if err != nil {
		return err
	}
	return model.Insert(ctx, &modelLog.SystemLog{
		Type:     modelLog.TypeLogin.Uint8(),
		Date:     event.At.Format(time.DateOnly),
		ObjectID: event.UserID,
		Content:  string(content),
	})
}

func RecordRegister(ctx context.Context, model Inserter, event RegisterEvent) error {
	if model == nil || event.UserID <= 0 || event.Method == "" {
		return ErrInvalidEvent
	}
	event.At = eventTime(event.At)
	content, err := (&modelLog.Register{
		AuthMethod: event.Method,
		Identifier: event.Identifier,
		RegisterIP: event.IP,
		UserAgent:  event.UserAgent,
		Timestamp:  event.At.UnixMilli(),
	}).Marshal()
	if err != nil {
		return err
	}
	return model.Insert(ctx, &modelLog.SystemLog{
		Type:     modelLog.TypeRegister.Uint8(),
		Date:     event.At.Format(time.DateOnly),
		ObjectID: event.UserID,
		Content:  string(content),
	})
}

func eventTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now()
	}
	return at
}
