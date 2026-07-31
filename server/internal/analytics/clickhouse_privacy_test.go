package analytics

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/privacy"
)

func TestProtectAccessEventEncryptsPrivateFields(t *testing.T) {
	codec := newAnalyticsPrivacyCodec(t)
	store := &clickHouseStore{privacy: codec}
	event := AccessEvent{
		EventID:         uuid.MustParse("018fdb36-d33e-7a6a-8b5f-98d64ec9a101"),
		SourceIP:        "203.0.113.42",
		DestinationHost: "private.example.com",
		DestinationIP:   "198.51.100.8",
		Reason:          "connection refused by upstream",
	}

	protected, err := store.protectAccessEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if protected.PrivacyVersion != accessPrivacyVersion {
		t.Fatalf("privacy version = %d, want %d", protected.PrivacyVersion, accessPrivacyVersion)
	}
	for name, value := range map[string]string{
		"source IP":        protected.SourceIP,
		"destination host": protected.DestinationHost,
		"destination IP":   protected.DestinationIP,
		"reason":           protected.Reason,
	} {
		if !strings.HasPrefix(value, "enc:v1:") {
			t.Fatalf("%s was not encrypted: %q", name, value)
		}
	}
	if protected.SourceIPDisplay != "203.0.113.*" || protected.DestinationIPDisplay != "198.51.100.*" {
		t.Fatalf("IP displays were not masked: %#v", protected)
	}
	if protected.SourceIPBlindIndex == "" || strings.Contains(protected.SourceIPBlindIndex, event.SourceIP) {
		t.Fatalf("source IP blind index is invalid: %q", protected.SourceIPBlindIndex)
	}
	if protected.SourceNetworkBlindIndex == "" || strings.Contains(protected.SourceNetworkBlindIndex, "203.0.113.42") {
		t.Fatalf("source network blind index is invalid: %q", protected.SourceNetworkBlindIndex)
	}

	opened, err := codec.Open(accessPrivacyScope(event.EventID, "source_ip"), protected.SourceIP)
	if err != nil {
		t.Fatal(err)
	}
	if opened != event.SourceIP {
		t.Fatalf("opened source IP = %q, want %q", opened, event.SourceIP)
	}
}

func TestProtectAccessEventGroupsIPv6By64(t *testing.T) {
	codec := newAnalyticsPrivacyCodec(t)
	store := &clickHouseStore{privacy: codec}
	first, err := store.protectAccessEvent(AccessEvent{EventID: uuid.New(), SourceIP: "2001:db8:1:2::1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.protectAccessEvent(AccessEvent{EventID: uuid.New(), SourceIP: "2001:db8:1:2::ffff"})
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceNetworkBlindIndex == "" || first.SourceNetworkBlindIndex != second.SourceNetworkBlindIndex {
		t.Fatalf("same /64 indexes differ: %q != %q", first.SourceNetworkBlindIndex, second.SourceNetworkBlindIndex)
	}
	if first.SourceIPBlindIndex == second.SourceIPBlindIndex {
		t.Fatal("distinct IPv6 source addresses share the exact-address blind index")
	}
}

func TestAccessFiltersUseBlindIndexesWhenPrivacyEnabled(t *testing.T) {
	codec := newAnalyticsPrivacyCodec(t)
	store := &clickHouseStore{privacy: codec}
	where, args := store.appendAccessFilters(nil, nil, AccessFilter{
		SourceIP:    "203.0.113.42",
		Destination: "Private.Example.com",
	})
	condition := strings.Join(where, " AND ")
	if strings.Contains(condition, "positionCaseInsensitive") || !strings.Contains(condition, "source_ip_bidx") || !strings.Contains(condition, "destination_host_bidx") {
		t.Fatalf("privacy filter condition = %q", condition)
	}
	for _, arg := range args {
		value, _ := arg.(string)
		if strings.Contains(value, "203.0.113.42") || strings.Contains(strings.ToLower(value), "private.example.com") {
			t.Fatalf("filter argument leaked plaintext: %q", value)
		}
	}
	if len(args) != 3 || args[1] != args[2] {
		t.Fatalf("destination blind index arguments = %#v", args)
	}
}

func TestAccessDestinationFilterCoversHostAndIPWithoutPrivacy(t *testing.T) {
	store := &clickHouseStore{privacy: privacy.DisabledCodec()}
	where, args := store.appendAccessFilters(nil, nil, AccessFilter{Destination: "198.51.100.8"})
	condition := strings.Join(where, " AND ")
	if !strings.Contains(condition, "destination_host") || !strings.Contains(condition, "destination_ip") {
		t.Fatalf("destination filter condition = %q", condition)
	}
	if len(args) != 2 || args[0] != "198.51.100.8" || args[1] != "198.51.100.8" {
		t.Fatalf("destination filter arguments = %#v", args)
	}
}

func newAnalyticsPrivacyCodec(t *testing.T) *privacy.Codec {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	indexKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	codec, err := privacy.NewCodec(privacy.Keyring{
		ActiveKeyID:   "test",
		EncryptionKey: map[string]string{"test": key},
		IndexKey:      indexKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return codec
}
