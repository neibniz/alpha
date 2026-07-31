package auth

import "github.com/neibniz/alpha/server/internal/privacy"

func logIP(value string) string {
	return privacy.MaskIP(value)
}

func logIdentifier(value string) string {
	return privacy.MaskIdentifier(value)
}
