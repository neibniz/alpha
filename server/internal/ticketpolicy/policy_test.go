package ticketpolicy

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeTicket(t *testing.T) {
	title, description, err := NormalizeTicket("  Need help  ", "  Details  ")
	if err != nil || title != "Need help" || description != "Details" {
		t.Fatalf("NormalizeTicket() = %q, %q, %v", title, description, err)
	}
	for _, input := range [][2]string{
		{"", "details"},
		{"title", ""},
		{strings.Repeat("x", MaxTitleRunes+1), "details"},
		{"title", strings.Repeat("x", MaxDescriptionRunes+1)},
	} {
		if _, _, err := NormalizeTicket(input[0], input[1]); err == nil {
			t.Fatalf("NormalizeTicket(%q, description length %d) accepted invalid input", input[0], len(input[1]))
		}
	}
}

func TestNormalizeFollow(t *testing.T) {
	content, err := NormalizeFollow(FollowText, "  reply  ")
	if err != nil || content != "reply" {
		t.Fatalf("NormalizeFollow(text) = %q, %v", content, err)
	}
	image := pngDataURL()
	if got, err := NormalizeFollow(FollowImage, image); err != nil || got != image {
		t.Fatalf("NormalizeFollow(image) = %q, %v", got, err)
	}
	for _, input := range []struct {
		kind    uint8
		content string
	}{
		{FollowText, ""},
		{9, "reply"},
		{FollowImage, "data:image/svg+xml;base64,PHN2Zz4="},
		{FollowImage, "data:image/png;base64,not-base64"},
	} {
		if _, err := NormalizeFollow(input.kind, input.content); err == nil {
			t.Fatalf("NormalizeFollow(%d, %q) accepted invalid input", input.kind, input.content)
		}
	}
}

func TestNormalizeMessageSupportsImageCollectionsAndLegacyImages(t *testing.T) {
	image := pngDataURL()
	content, images, err := NormalizeMessage(FollowText, "  reply  ", []string{image, image})
	if err != nil || content != "reply" || len(images) != 1 || images[0] != image {
		t.Fatalf("NormalizeMessage(text gallery) = %q, %#v, %v", content, images, err)
	}

	content, images, err = NormalizeMessage(FollowImage, image, nil)
	if err != nil || content != "" || len(images) != 1 || images[0] != image {
		t.Fatalf("NormalizeMessage(legacy image) = %q, %#v, %v", content, images, err)
	}

	if _, _, err := NormalizeMessage(FollowText, "", nil); err == nil {
		t.Fatal("empty ticket message was accepted")
	}
	if _, err := NormalizeImages(make([]string, MaxTicketImages+1)); err == nil {
		t.Fatal("oversized image collection was accepted")
	}
	mismatched := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	if _, err := NormalizeImages([]string{mismatched}); err == nil {
		t.Fatal("image with mismatched declared and detected MIME type was accepted")
	}
}

func pngDataURL() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func TestStatusPolicies(t *testing.T) {
	if !AdminStatusAllowed(StatusProcessed) || !AdminStatusAllowed(StatusClosed) || AdminStatusAllowed(StatusWaiting) {
		t.Fatal("admin status policy mismatch")
	}
	if !UserStatusAllowed(StatusPending) || !UserStatusAllowed(StatusClosed) || UserStatusAllowed(StatusProcessed) {
		t.Fatal("user status policy mismatch")
	}
}
