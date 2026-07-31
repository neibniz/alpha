package application

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

var supportedSubscribeOutputFormats = map[string]struct{}{
	"base64": {},
	"yaml":   {},
	"json":   {},
	"conf":   {},
	"plain":  {},
}

func validateCreateSubscribeApplication(req *types.CreateSubscribeApplicationRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Scheme = strings.TrimSpace(req.Scheme)
	req.UserAgent = strings.TrimSpace(req.UserAgent)
	req.OutputFormat = strings.ToLower(strings.TrimSpace(req.OutputFormat))

	if err := validateSubscribeApplicationIdentity(req.Name, req.Description, req.Icon, req.Scheme, req.UserAgent); err != nil {
		return err
	}
	if strings.TrimSpace(req.SubscribeTemplate) == "" {
		return invalidSubscribeApplication("subscription template is required")
	}
	if err := validateSubscribeOutputFormat(req.OutputFormat); err != nil {
		return err
	}
	return normalizeAndValidateDownloadLinks(&req.DownloadLink)
}

func validateUpdateSubscribeApplication(req *types.UpdateSubscribeApplicationRequest) error {
	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
		if *req.Name == "" || utf8.RuneCountInString(*req.Name) > 255 {
			return invalidSubscribeApplication("client name must contain 1 to 255 characters")
		}
	}
	if req.Description != nil {
		*req.Description = strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(*req.Description) > 255 {
			return invalidSubscribeApplication("client description must not exceed 255 characters")
		}
	}
	if req.Icon != nil {
		*req.Icon = strings.TrimSpace(*req.Icon)
		if !isValidImageSource(*req.Icon) {
			return invalidSubscribeApplication("invalid client icon source")
		}
	}
	if req.Scheme != nil {
		*req.Scheme = strings.TrimSpace(*req.Scheme)
		if utf8.RuneCountInString(*req.Scheme) > 255 {
			return invalidSubscribeApplication("client scheme must not exceed 255 characters")
		}
	}
	if req.UserAgent != nil {
		*req.UserAgent = strings.TrimSpace(*req.UserAgent)
		if *req.UserAgent == "" || utf8.RuneCountInString(*req.UserAgent) > 255 {
			return invalidSubscribeApplication("user agent must contain 1 to 255 characters")
		}
	}
	if req.SubscribeTemplate != nil && strings.TrimSpace(*req.SubscribeTemplate) == "" {
		return invalidSubscribeApplication("subscription template is required")
	}
	if req.OutputFormat != nil {
		*req.OutputFormat = strings.ToLower(strings.TrimSpace(*req.OutputFormat))
		if err := validateSubscribeOutputFormat(*req.OutputFormat); err != nil {
			return err
		}
	}
	if req.DownloadLink != nil {
		return normalizeAndValidateDownloadLinks(req.DownloadLink)
	}
	return nil
}

func validateSubscribeApplicationIdentity(name, description, icon, scheme, userAgent string) error {
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return invalidSubscribeApplication("client name must contain 1 to 255 characters")
	}
	if utf8.RuneCountInString(description) > 255 {
		return invalidSubscribeApplication("client description must not exceed 255 characters")
	}
	if !isValidImageSource(icon) {
		return invalidSubscribeApplication("invalid client icon source")
	}
	if utf8.RuneCountInString(scheme) > 255 {
		return invalidSubscribeApplication("client scheme must not exceed 255 characters")
	}
	if userAgent == "" || utf8.RuneCountInString(userAgent) > 255 {
		return invalidSubscribeApplication("user agent must contain 1 to 255 characters")
	}
	return nil
}

func validateSubscribeOutputFormat(value string) error {
	if _, ok := supportedSubscribeOutputFormats[value]; !ok {
		return invalidSubscribeApplication("unsupported subscription output format")
	}
	return nil
}

func isValidImageSource(value string) bool {
	if value == "" || strings.HasPrefix(value, "data:image/") {
		return true
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	return isHTTPURL(value)
}

func normalizeAndValidateDownloadLinks(link *types.DownloadLink) error {
	values := []*string{
		&link.IOS,
		&link.Android,
		&link.Windows,
		&link.Mac,
		&link.Linux,
		&link.Harmony,
	}
	for _, value := range values {
		*value = strings.TrimSpace(*value)
		if *value != "" && !isHTTPURL(*value) {
			return invalidSubscribeApplication("download links must use HTTP or HTTPS")
		}
	}
	return nil
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func invalidSubscribeApplication(message string) error {
	return xerr.NewErrCodeMsg(xerr.InvalidParams, message)
}
