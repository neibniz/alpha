package oauth

import "github.com/neibniz/alpha/server/internal/privacy"

func logIP(value string) string {
	return privacy.MaskIP(value)
}

func logIdentifier(value string) string {
	return privacy.MaskIdentifier(value)
}

func logUserAgent(value string) string {
	return privacy.MaskUserAgent(value)
}
