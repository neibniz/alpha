package config

import "strconv"

// CurrencyConfigKey Currency Config Key
const CurrencyConfigKey = "system:currency_config"

// SiteConfigKey Site Config Key
const SiteConfigKey = "system:site_config"

// SubscribeConfigKey Subscribe Config Key
const SubscribeConfigKey = "system:subscribe_config"

// RegisterConfigKey Register Config Key
const RegisterConfigKey = "system:register_config"

// VerifyConfigKey Verify Config Key
const VerifyConfigKey = "system:verify_config"

// EmailSmtpConfigKey Email Smtp Config Key
const EmailSmtpConfigKey = "system:email_smtp_config"

// NodeConfigKey Node Config Key
const NodeConfigKey = "system:node_config"

// InviteConfigKey Invite Config Key
const InviteConfigKey = "system:invite_config"

// TelegramConfigKey Telegram Config Key
const TelegramConfigKey = "system:telegram_config"

// TosConfigKey Tos配置
const TosConfigKey = "system:tos_config"

// VerifyCodeConfigKey Verify Code Config Key
const VerifyCodeConfigKey = "system:verify_code_config"

// SessionIdKey cache session key
const SessionIdKey = "auth:session_id"
const UserSessionIndexKey = "auth:user_sessions"

func SessionCacheKey(sessionID string) string {
	return SessionIdKey + ":" + sessionID
}

func SessionUserValue(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

func UserSessionCacheKey(userID int64) string {
	return UserSessionIndexKey + ":" + strconv.FormatInt(userID, 10)
}

func TelegramBindCacheKey(token string) string {
	return "auth:telegram_bind:" + token
}

// GlobalConfigKey Global Config Key
const GlobalConfigKey = "system:global_config"

// AuthCodeCacheKey Register Code Cache Key
const AuthCodeCacheKey = "auth:verify:email"

func EmailAuthCodeCacheKey(purpose, email string) string {
	return authCodeCacheKey(AuthCodeCacheKey, purpose, email)
}

func authCodeCacheKey(prefix, purpose, account string) string {
	return prefix + ":" + purpose + ":" + account
}

// CommonStatCacheKey CommonStat Cache Key
const CommonStatCacheKey = "common:stat"

// SendIntervalKeyPrefix Auth Code Send Interval Key Prefix
const SendIntervalKeyPrefix = "send:interval:"

func SendIntervalCacheKey(channel, purpose string) string {
	return SendIntervalKeyPrefix + ":" + channel + ":" + purpose
}

// SendCountLimitKeyPrefix Send Count Limit Key Prefix eg. send:limit:register:email:xxx@alpha.dev
const SendCountLimitKeyPrefix = "send:limit:"

func SendCountLimitValue(channel, purpose, account string) string {
	return channel + ":" + purpose + ":" + account
}
