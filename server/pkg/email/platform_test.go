package email

import "testing"

func TestPlatformStringUsesStableNames(t *testing.T) {
	if got := SMTP.String(); got != "smtp" {
		t.Fatalf("SMTP.String() = %q, want smtp", got)
	}
	if got := unsupported.String(); got != "unsupported" {
		t.Fatalf("unsupported.String() = %q, want unsupported", got)
	}
	if got := Platform(99).String(); got != "unsupported" {
		t.Fatalf("unknown platform String() = %q, want unsupported", got)
	}
}

func TestParsePlatformKeepsSupportedNames(t *testing.T) {
	if got := parsePlatform("smtp"); got != SMTP {
		t.Fatalf("parsePlatform(smtp) = %v, want SMTP", got)
	}
	if got := parsePlatform("missing"); got != unsupported {
		t.Fatalf("parsePlatform(missing) = %v, want unsupported", got)
	}
}

func TestGetSupportedPlatformsKeepsEmailShape(t *testing.T) {
	platforms := GetSupportedPlatforms()
	if len(platforms) != 1 || platforms[0].Platform != "smtp" {
		t.Fatalf("email platforms = %#v", platforms)
	}
	if platforms[0].PlatformFieldDescription == nil || platforms[0].PlatformFieldDescription["host"] == "" {
		t.Fatalf("email platform fields = %#v", platforms[0].PlatformFieldDescription)
	}
}
