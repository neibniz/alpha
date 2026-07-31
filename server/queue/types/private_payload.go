package types

import (
	"encoding/json"
	"fmt"

	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
)

const (
	EmailPayloadScope = "queue.email.payload"
)

func EncodePrivatePayload(codec *privacycodec.Codec, scope string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if codec == nil {
		return data, nil
	}
	sealed, err := codec.Seal(scope, string(data))
	if err != nil {
		return nil, err
	}
	return []byte(sealed), nil
}

func DecodePrivatePayload(codec *privacycodec.Codec, scope string, data []byte, value any) error {
	plaintext := string(data)
	if codec != nil {
		opened, err := codec.Open(scope, plaintext)
		if err != nil {
			return err
		}
		plaintext = opened
	}
	if err := json.Unmarshal([]byte(plaintext), value); err != nil {
		return fmt.Errorf("decode private queue payload: %w", err)
	}
	return nil
}
