package constant

type CtxKey string

const (
	CtxKeyUser        CtxKey = "user"
	CtxKeySessionID   CtxKey = "sessionId"
	CtxKeyRequestHost CtxKey = "requestHost"
	CtxKeyClientIP    CtxKey = "clientIP"
	CtxKeyPlatform    CtxKey = "platform"
	CtxKeyPayment     CtxKey = "payment"
	CtxKeyServerID    CtxKey = "serverId"
	LoginType         CtxKey = "loginType"
)
