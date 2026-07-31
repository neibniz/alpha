package uuidx

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/random"
	"github.com/gofrs/uuid/v5"
)

// NewUUID returns a new UUID.
func NewUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Errorw("[UUID] Generate UUID failed", logger.Field("error", err.Error()))
		return uuid.UUID{}
	}
	return id
}

// ParseUUIDSlice parses the UUID string slice to UUID slice.
func ParseUUIDSlice(ids []string) []uuid.UUID {
	var result []uuid.UUID
	for _, v := range ids {
		p, err := uuid.FromString(v)
		if err != nil {
			return nil
		}
		result = append(result, p)
	}
	return result
}

// ParseUUIDString parses UUID string to UUID type.
func ParseUUIDString(id string) uuid.UUID {
	result, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}
	}
	return result
}

// ParseUUIDSliceToPointer parses the UUID string slice to UUID pointer slice.
func ParseUUIDSliceToPointer(ids []string) []*uuid.UUID {
	var result []*uuid.UUID
	for _, v := range ids {
		p, err := uuid.FromString(v)
		if err != nil {
			return nil
		}
		result = append(result, &p)
	}
	return result
}

// ParseUUIDStringToPointer parses UUID string to UUID pointer.
func ParseUUIDStringToPointer(id *string) *uuid.UUID {
	if id == nil {
		return nil
	}

	result, err := uuid.FromString(*id)
	if err != nil {
		return nil
	}
	return &result
}

func UserInviteCode(id int64) string {
	return "u" + random.EncodeBase62(id+time.Now().UnixMilli())
}

func AffiliateInviteCode(id int64) string {
	return "A" + random.EncodeBase62(id)
}

func NewSubscribeToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func UUIDToBase64(uuid string, length int) string {
	// 截取 uuid 的前 length 个字符
	if length > len(uuid) {
		length = len(uuid)
	}
	shortUUID := uuid[:length]

	// 对截取的字符串进行 Base64 编码
	encoded := base64.StdEncoding.EncodeToString([]byte(shortUUID))

	return encoded
}
