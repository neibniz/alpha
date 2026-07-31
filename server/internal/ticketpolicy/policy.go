package ticketpolicy

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	StatusPending   uint8 = 1
	StatusWaiting   uint8 = 2
	StatusProcessed uint8 = 3
	StatusClosed    uint8 = 4

	FollowText  uint8 = 1
	FollowImage uint8 = 2

	FromUser   = "User"
	FromSystem = "System"

	MaxTitleRunes       = 255
	MaxDescriptionRunes = 20_000
	MaxFollowTextRunes  = 10_000
	MaxImageDataURL     = 1_500_000
	MaxImageBytes       = 1_050_000
	MaxTicketImages     = 6
	MaxImagesTotalBytes = 6 * MaxImageBytes
	MaxRequestBodyBytes = 10 << 20
)

var ErrInvalidTicket = errors.New("invalid ticket payload")

var allowedImagePrefixes = map[string]string{
	"data:image/gif;base64,":  "image/gif",
	"data:image/jpeg;base64,": "image/jpeg",
	"data:image/jpg;base64,":  "image/jpeg",
	"data:image/png;base64,":  "image/png",
	"data:image/webp;base64,": "image/webp",
}

func NormalizeTicket(title, description string) (string, string, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if count := utf8.RuneCountInString(title); count == 0 || count > MaxTitleRunes {
		return "", "", ErrInvalidTicket
	}
	if count := utf8.RuneCountInString(description); count == 0 || count > MaxDescriptionRunes {
		return "", "", ErrInvalidTicket
	}
	return title, description, nil
}

func NormalizeFollow(followType uint8, content string) (string, error) {
	content = strings.TrimSpace(content)
	switch followType {
	case FollowText:
		if count := utf8.RuneCountInString(content); count == 0 || count > MaxFollowTextRunes {
			return "", ErrInvalidTicket
		}
		return content, nil
	case FollowImage:
		if _, _, err := normalizeImage(content); err != nil {
			return "", ErrInvalidTicket
		}
		return content, nil
	default:
		return "", ErrInvalidTicket
	}
}

func NormalizeMessage(followType uint8, content string, images []string) (string, []string, error) {
	normalizedImages, err := NormalizeImages(images)
	if err != nil {
		return "", nil, err
	}
	switch followType {
	case FollowText:
		content = strings.TrimSpace(content)
		if utf8.RuneCountInString(content) > MaxFollowTextRunes {
			return "", nil, ErrInvalidTicket
		}
	case FollowImage:
		content = strings.TrimSpace(content)
		if content != "" {
			legacyImage, _, err := normalizeImage(content)
			if err != nil {
				return "", nil, ErrInvalidTicket
			}
			normalizedImages, err = NormalizeImages(append([]string{legacyImage}, normalizedImages...))
			if err != nil {
				return "", nil, err
			}
		}
		content = ""
	default:
		return "", nil, ErrInvalidTicket
	}
	if content == "" && len(normalizedImages) == 0 {
		return "", nil, ErrInvalidTicket
	}
	return content, normalizedImages, nil
}

func NormalizeImages(images []string) ([]string, error) {
	if len(images) > MaxTicketImages {
		return nil, ErrInvalidTicket
	}
	normalized := make([]string, 0, len(images))
	seen := make(map[[sha256.Size]byte]struct{}, len(images))
	totalBytes := 0
	for _, image := range images {
		value, decodedBytes, err := normalizeImage(image)
		if err != nil || totalBytes > MaxImagesTotalBytes-decodedBytes {
			return nil, ErrInvalidTicket
		}
		hash := sha256.Sum256([]byte(value))
		if _, exists := seen[hash]; exists {
			continue
		}
		seen[hash] = struct{}{}
		totalBytes += decodedBytes
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func normalizeImage(content string) (string, int, error) {
	content = strings.TrimSpace(content)
	if len(content) == 0 || len(content) > MaxImageDataURL {
		return "", 0, ErrInvalidTicket
	}
	lower := strings.ToLower(content)
	expectedType := ""
	prefixLength := 0
	for prefix, mediaType := range allowedImagePrefixes {
		if strings.HasPrefix(lower, prefix) {
			expectedType = mediaType
			prefixLength = len(prefix)
			break
		}
	}
	if prefixLength == 0 {
		return "", 0, ErrInvalidTicket
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(content[prefixLength:])
	if err != nil || len(decoded) == 0 || len(decoded) > MaxImageBytes {
		return "", 0, ErrInvalidTicket
	}
	if detected := http.DetectContentType(decoded); detected != expectedType {
		return "", 0, ErrInvalidTicket
	}
	return content, len(decoded), nil
}

func AdminStatusAllowed(status uint8) bool {
	return status == StatusPending || status == StatusProcessed || status == StatusClosed
}

func UserStatusAllowed(status uint8) bool {
	return status == StatusPending || status == StatusClosed
}
