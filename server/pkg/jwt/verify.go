package jwt

import "github.com/golang-jwt/jwt/v5"

var (
	InvalidToken = jwt.ErrTokenInvalidId
	ExpiredToken = jwt.ErrTokenExpired
)

func VerifyToken(tokenString, secret string) error {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return err
	}
	return nil
}
