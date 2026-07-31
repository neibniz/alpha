package middleware

import (
	"context"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/jwt"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func AuthMiddleware(svc *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		jwtConfig := svc.Config.JwtAuth
		// get token from header
		token := c.GetHeader("Authorization")
		if token == "" {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] Token Empty")
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenEmpty), "Token Empty"))
			c.Abort()
			return
		}
		// parse token
		claims, err := jwt.ParseJwtToken(token, jwtConfig.AccessSecret)
		if err != nil {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] ParseJwtToken", logger.Field("error", err.Error()))
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenExpire), "Token Invalid"))
			c.Abort()
			return
		}

		loginType, ok := stringClaim(claims, "LoginType", true)
		if !ok {
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Token"))
			c.Abort()
			return
		}
		userId, ok := int64Claim(claims, "UserId")
		if !ok {
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Token"))
			c.Abort()
			return
		}
		sessionId, ok := stringClaim(claims, "SessionId", false)
		if !ok {
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Token"))
			c.Abort()
			return
		}
		// get session id from redis
		sessionIdCacheKey := config.SessionCacheKey(sessionId)
		value, err := svc.Redis.Get(c.Request.Context(), sessionIdCacheKey).Result()
		if err != nil {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] Redis Get", logger.Field("error", err.Error()))
			if errors.Is(err, redis.Nil) {
				result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Session"))
			} else {
				result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Session Store Unavailable"))
			}
			c.Abort()
			return
		}

		//verify user id
		if value != config.SessionUserValue(userId) {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] Invalid Access", logger.Field("userId", userId))
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Session"))
			c.Abort()
			return
		}

		userInfo, err := svc.UserModel.FindOneForContext(c, userId)
		if err != nil {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] UserModel FindOneForContext", logger.Field("error", err.Error()), logger.Field("userId", userId))
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Session"))
			} else {
				result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Database Query Error"))
			}
			c.Abort()
			return
		}
		if userInfo.Enable == nil || !*userInfo.Enable || userInfo.DeletedAt.Valid {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] Disabled User", logger.Field("userId", userId))
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.ErrorTokenInvalid), "Invalid Session"))
			c.Abort()
			return
		}
		// admin verify
		if pathHasSegment(c.Request.URL.Path, "admin") && (userInfo.IsAdmin == nil || !*userInfo.IsAdmin) {
			logger.WithContext(c.Request.Context()).Debug("[AuthMiddleware] Not Admin User", logger.Field("userId", userId))
			result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access"))
			c.Abort()
			return
		}
		ctx = context.WithValue(ctx, constant.LoginType, loginType)
		ctx = context.WithValue(ctx, constant.CtxKeyUser, userInfo)
		ctx = context.WithValue(ctx, constant.CtxKeySessionID, sessionId)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func pathHasSegment(path string, segment string) bool {
	if segment == "" {
		return false
	}
	for len(path) > 0 {
		for len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if path == "" {
			return false
		}
		index := strings.IndexByte(path, '/')
		if index < 0 {
			return path == segment
		}
		if path[:index] == segment {
			return true
		}
		path = path[index+1:]
	}
	return false
}

func stringClaim(claims map[string]any, key string, optional bool) (string, bool) {
	value, exists := claims[key]
	if !exists || value == nil {
		return "", optional
	}
	text, ok := value.(string)
	if !ok || (!optional && text == "") {
		return "", false
	}
	return text, true
}

func int64Claim(claims map[string]any, key string) (int64, bool) {
	value, exists := claims[key]
	if !exists || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		if v <= 0 || v != float64(int64(v)) {
			return 0, false
		}
		return int64(v), true
	case int64:
		return v, v > 0
	case int:
		return int64(v), v > 0
	case string:
		id, err := strconv.ParseInt(v, 10, 64)
		return id, err == nil && id > 0
	default:
		return 0, false
	}
}
