package application

import (
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
)

func TestValidateCreateSubscribeApplicationNormalizesFields(t *testing.T) {
	req := &types.CreateSubscribeApplicationRequest{
		Name:              "  Clash  ",
		Description:       "  Default client  ",
		Icon:              " /assets/clash.png ",
		Scheme:            " clash://install?url=${url} ",
		UserAgent:         " Clash ",
		SubscribeTemplate: "{{ .SiteName }}",
		OutputFormat:      " YAML ",
		DownloadLink: types.DownloadLink{
			Windows: " https://example.com/windows ",
		},
	}

	if err := validateCreateSubscribeApplication(req); err != nil {
		t.Fatal(err)
	}
	if req.Name != "Clash" || req.UserAgent != "Clash" || req.OutputFormat != "yaml" {
		t.Fatalf("normalized request = %#v", req)
	}
	if req.DownloadLink.Windows != "https://example.com/windows" {
		t.Fatalf("normalized download link = %q", req.DownloadLink.Windows)
	}
}

func TestValidateCreateSubscribeApplicationCountsUnicodeCharacters(t *testing.T) {
	req := &types.CreateSubscribeApplicationRequest{
		Name:              strings.Repeat("客", 100),
		Description:       strings.Repeat("简", 100),
		UserAgent:         "Alpha Client",
		SubscribeTemplate: "template",
		OutputFormat:      "yaml",
	}

	if err := validateCreateSubscribeApplication(req); err != nil {
		t.Fatalf("validateCreateSubscribeApplication() error = %v", err)
	}
}

func TestValidateSubscribeApplicationRejectsInvalidContractFields(t *testing.T) {
	tests := []struct {
		name string
		req  types.CreateSubscribeApplicationRequest
	}{
		{
			name: "missing name",
			req: types.CreateSubscribeApplicationRequest{
				UserAgent: "Clash", SubscribeTemplate: "template", OutputFormat: "yaml",
			},
		},
		{
			name: "unsupported output",
			req: types.CreateSubscribeApplicationRequest{
				Name: "Clash", UserAgent: "Clash", SubscribeTemplate: "template", OutputFormat: "xml",
			},
		},
		{
			name: "invalid download URL",
			req: types.CreateSubscribeApplicationRequest{
				Name: "Clash", UserAgent: "Clash", SubscribeTemplate: "template", OutputFormat: "yaml",
				DownloadLink: types.DownloadLink{Windows: "javascript:alert(1)"},
			},
		},
		{
			name: "protocol relative icon URL",
			req: types.CreateSubscribeApplicationRequest{
				Name: "Clash", UserAgent: "Clash", SubscribeTemplate: "template", OutputFormat: "yaml",
				Icon: "//tracker.example/icon.png",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateCreateSubscribeApplication(&test.req); err == nil {
				t.Fatal("validateCreateSubscribeApplication() error = nil")
			}
		})
	}
}

func TestValidateUpdateSubscribeApplicationRejectsBlankRequiredFields(t *testing.T) {
	blank := "   "
	req := &types.UpdateSubscribeApplicationRequest{Name: &blank}
	if err := validateUpdateSubscribeApplication(req); err == nil {
		t.Fatal("validateUpdateSubscribeApplication() error = nil")
	}
}
