package tool

import (
	"crypto/md5" // #nosec G501 -- required for legacy password verification and external provider signatures
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/anaskhan96/go-password-encoder"
	"golang.org/x/crypto/bcrypt"
)

var options = &password.Options{SaltLen: 16, Iterations: 100, KeyLen: 32, HashFunction: sha512.New}

const (
	PasswordMinRunes = 12
	PasswordMaxBytes = 72
	PasswordAlgo     = "bcrypt"
	passwordCost     = 12
)

var ErrPasswordLength = errors.New("password must contain at least 12 characters and at most 72 bytes")

func EncodePassWord(value string) (string, error) {
	if err := ValidatePassword(value); err != nil {
		return "", err
	}
	return hashPassword(value)
}

func ValidatePassword(value string) error {
	if utf8.RuneCountInString(value) < PasswordMinRunes || len(value) > PasswordMaxBytes {
		return ErrPasswordLength
	}
	return nil
}

func RehashPassWord(value string) (string, error) {
	if value == "" || len(value) > PasswordMaxBytes {
		return "", ErrPasswordLength
	}
	return hashPassword(value)
}

func hashPassword(value string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(value), passwordCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func PasswordNeedsUpgrade(algo, hash string) bool {
	if algo != PasswordAlgo {
		return true
	}
	cost, err := bcrypt.Cost([]byte(hash))
	return err != nil || cost < passwordCost
}

func VerifyPassWord(passwd, EncodePasswd string) bool {
	info := strings.Split(EncodePasswd, "$")
	if len(info) != 4 || info[1] != "pbkdf2-sha512" || info[2] == "" || info[3] == "" {
		return false
	}
	return password.Verify(passwd, info[2], info[3], options)
}

func Md5Encode(str string, isUpper bool) string {
	sum := md5.Sum([]byte(str)) // #nosec G401 -- legacy payment providers require MD5 signatures
	res := hex.EncodeToString(sum[:])
	//转大写，strings.ToUpper(res)
	if isUpper {
		res = strings.ToUpper(res)
	}
	return res
}

func MultiPasswordVerify(algo, salt, password, hash string) bool {
	switch algo {
	case "md5":
		sum := md5.Sum([]byte(password)) // #nosec G401 -- verify-only migration path; successful login rehashes to bcrypt
		return constantTimeHashEqual(hex.EncodeToString(sum[:]), hash)
	case "sha256":
		sum := sha256.Sum256([]byte(password))
		return constantTimeHashEqual(hex.EncodeToString(sum[:]), hash)
	case "md5salt":
		sum := md5.Sum([]byte(password + salt)) // #nosec G401 -- verify-only migration path; successful login rehashes to bcrypt
		return constantTimeHashEqual(hex.EncodeToString(sum[:]), hash)
	case "default": // Alpha's default algorithm
		return VerifyPassWord(password, hash)
	case "bcrypt":
		// Bcrypt (corresponding to PHP's password_hash/password_verify)
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		return err == nil
	}
	return false
}

func constantTimeHashEqual(expected, provided string) bool {
	return len(expected) == len(provided) && subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
