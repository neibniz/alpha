package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/neibniz/alpha/server/pkg/syncx"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	// ErrNotFound is the error when cache not found.
	ErrNotFound = redis.Nil
)

type (
	// ExecCtxFn defines the sql exec method.
	ExecCtxFn func(conn *gorm.DB) error
	// IndexQueryCtxFn defines the query method that based on unique indexes.
	IndexQueryCtxFn func(conn *gorm.DB, v interface{}) (interface{}, error)
	// PrimaryQueryCtxFn defines the query method that based on primary keys.
	PrimaryQueryCtxFn func(conn *gorm.DB, v, primary interface{}) error
	// QueryCtxFn defines the query method.
	QueryCtxFn func(conn *gorm.DB, v interface{}) error

	CachedConn struct {
		db             *gorm.DB
		cache          *redis.Client
		singleFlight   syncx.SingleFlight
		expiry         time.Duration
		notFoundExpiry time.Duration
	}
)

// NewConn returns a CachedConn with a redis cluster cache.
func NewConn(db *gorm.DB, c *redis.Client, opts ...Option) CachedConn {
	o := newOptions(opts...)
	return CachedConn{
		db:             db,
		cache:          c,
		singleFlight:   syncx.NewSingleFlight(),
		expiry:         o.Expiry,
		notFoundExpiry: o.NotFoundExpiry,
	}
}

// DelCache deletes cache with keys.
func (cc CachedConn) DelCache(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return cc.cache.Del(context.Background(), keys...).Err()
}

// DelCacheCtx deletes cache with keys.
func (cc CachedConn) DelCacheCtx(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return cc.cache.Del(ctx, keys...).Err()
}

// GetCache unmarshals cache with given key into v.
func (cc CachedConn) GetCache(key string, v interface{}) error {
	return cc.GetCacheCtx(context.Background(), key, v)
}

// GetCacheCtx unmarshals cache with given key into v.
func (cc CachedConn) GetCacheCtx(ctx context.Context, key string, v interface{}) error {
	val, err := cc.getCacheBytesCtx(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(val, v)
}

// SetCache sets cache with key and v.
func (cc CachedConn) SetCache(key string, v interface{}) error {
	return cc.SetCacheCtx(context.Background(), key, v)
}

// SetCacheCtx sets cache with key and v.
func (cc CachedConn) SetCacheCtx(ctx context.Context, key string, v interface{}) error {
	// marshal value
	val, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return cc.setCacheBytesCtx(ctx, key, val)
}

// ExecCtx runs given exec on given keys, and returns execution result.
func (cc CachedConn) ExecCtx(ctx context.Context, execCtx ExecCtxFn, keys ...string) error {
	if len(keys) > 0 {
		_ = cc.DelCacheCtx(ctx, keys...)
	}
	err := execCtx(cc.db.WithContext(ctx))
	if err != nil {
		return err
	}
	if err := cc.DelCacheCtx(ctx, keys...); err != nil {
		return err
	}
	return nil
}

// ExecNoCache runs exec with given sql statement, without affecting cache.
func (cc CachedConn) ExecNoCache(exec ExecCtxFn) error {
	return cc.ExecNoCacheCtx(context.Background(), exec)
}

// ExecNoCacheCtx runs exec with given sql statement, without affecting cache.
func (cc CachedConn) ExecNoCacheCtx(ctx context.Context, execCtx ExecCtxFn) (err error) {
	return execCtx(cc.db.WithContext(ctx))
}

func (cc CachedConn) QueryCtx(ctx context.Context, v interface{}, key string, query QueryCtxFn) (err error) {
	err = cc.GetCacheCtx(ctx, key, v)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
	case isJSONCacheValueError(err):
	default:
		return err
	}

	result, fresh, err := cc.singleFlight.DoExCtx(ctx, key, func() (any, error) {
		return cc.queryAndCacheCtx(ctx, v, key, query)
	})
	if err != nil {
		return err
	}
	if fresh {
		return nil
	}
	return json.Unmarshal(result.([]byte), v)
}

func (cc CachedConn) queryAndCacheCtx(
	ctx context.Context,
	v interface{},
	key string,
	query QueryCtxFn,
) ([]byte, error) {
	val, err := cc.getCacheBytesCtx(ctx, key)
	switch {
	case err == nil:
		if err := json.Unmarshal(val, v); err == nil {
			return val, nil
		} else if !isJSONCacheValueError(err) {
			return nil, err
		}
		_ = cc.DelCacheCtx(ctx, key)
	case errors.Is(err, ErrNotFound):
	default:
		return nil, err
	}

	if err := query(cc.db.WithContext(ctx), v); err != nil {
		return nil, err
	}
	val, err = json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if err := cc.setCacheBytesCtx(ctx, key, val); err != nil {
		return nil, err
	}
	return val, nil
}

func (cc CachedConn) getCacheBytesCtx(ctx context.Context, key string) ([]byte, error) {
	return cc.cache.Get(ctx, key).Bytes()
}

func (cc CachedConn) setCacheBytesCtx(ctx context.Context, key string, val []byte) error {
	return cc.cache.Set(ctx, key, val, cc.expiry).Err()
}

func isJSONCacheValueError(err error) bool {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}

// QueryNoCacheCtx runs query with given sql statement, without affecting cache.
func (cc CachedConn) QueryNoCacheCtx(ctx context.Context, v interface{}, query QueryCtxFn) (err error) {
	return query(cc.db.WithContext(ctx), v)
}

// TransactCtx runs given fn in transaction mode.
func (cc CachedConn) TransactCtx(ctx context.Context, fn func(db *gorm.DB) error, opts ...*sql.TxOptions) error {
	return cc.db.WithContext(ctx).Transaction(fn, opts...)
}

// Transact runs given fn in transaction mode.
func (cc CachedConn) Transact(fn func(db *gorm.DB) error, opts ...*sql.TxOptions) error {
	return cc.TransactCtx(context.Background(), fn, opts...)
}
