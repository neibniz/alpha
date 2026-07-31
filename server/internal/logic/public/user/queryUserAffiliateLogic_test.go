package user

import (
	"strings"
	"testing"
	"time"
)

func TestAffiliateUserListSelectColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(affiliateUserListSelectColumns))
	for _, column := range affiliateUserListSelectColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"COALESCE(am.auth_identifier, '') AS identifier",
		"u.avatar",
		"u.created_at AS registered_at",
		"COALESCE(u.enable, false) AS enable",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("affiliate list columns must include %q", column)
		}
	}

	for _, column := range affiliateUserListSelectColumns {
		normalized := strings.ToLower(strings.Trim(column, `"`))
		forbidden := []string{"password", "salt", "rules", "balance", "commission", "auth_identifier AS"}
		for _, field := range forbidden {
			if strings.Contains(normalized, strings.ToLower(field)) {
				t.Fatalf("affiliate list columns must not include %q in %q", field, column)
			}
		}
	}
}

func TestAffiliateIdentifierJoinStaysSingleRow(t *testing.T) {
	if !strings.Contains(affiliateIdentifierJoinSQL, "LEFT JOIN LATERAL") {
		t.Fatal("affiliate identifier lookup must use a lateral join")
	}
	if !strings.Contains(affiliateIdentifierJoinSQL, "LIMIT 1") {
		t.Fatal("affiliate identifier lookup must return one auth method per user")
	}
	if strings.Contains(affiliateIdentifierJoinSQL, "SELECT *") {
		t.Fatal("affiliate identifier lookup must not select full auth method rows")
	}
}

func TestAffiliateIdentifierPriorityUsesCurrentAuthTypeShape(t *testing.T) {
	normalized := strings.ToLower(affiliateIdentifierPrioritySQL)
	for _, required := range []string{
		"when 'email' then 0",
		"when 'mobile' then 1",
		"else 2",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("affiliate identifier priority missing %q: %s", required, affiliateIdentifierPrioritySQL)
		}
	}
	for _, stale := range []string{"'6'", "'7'"} {
		if strings.Contains(normalized, stale) {
			t.Fatalf("affiliate identifier priority must not use stale numeric auth type %s: %s", stale, affiliateIdentifierPrioritySQL)
		}
	}
}

func TestMaskAffiliateIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "short", in: "ab", want: "ab"},
		{name: "email", in: "abcdefghi", want: "abc***ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskAffiliateIdentifier(tt.in); got != tt.want {
				t.Fatalf("maskAffiliateIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAffiliateUserRowsToTypesMasksAndKeepsListShape(t *testing.T) {
	registeredAt := time.UnixMilli(1700000000000)
	rows := []affiliateUserListRow{
		{
			Identifier:   "abcdefghi",
			Avatar:       "avatar-a",
			RegisteredAt: registeredAt,
			Enable:       true,
		},
		{
			Identifier: "ab",
			Avatar:     "avatar-b",
		},
	}

	got := affiliateUserRowsToTypes(rows)
	if len(got) != 2 {
		t.Fatalf("affiliate rows = %d, want 2: %#v", len(got), got)
	}
	if cap(got) != len(rows) {
		t.Fatalf("affiliate rows capacity = %d, want %d", cap(got), len(rows))
	}
	if got[0].Identifier != "abc***ghi" || got[0].Avatar != "avatar-a" || !got[0].Enable {
		t.Fatalf("first affiliate row not mapped or masked: %#v", got[0])
	}
	if got[0].RegisteredAt != registeredAt.UnixMilli() {
		t.Fatalf("registered_at = %d, want %d", got[0].RegisteredAt, registeredAt.UnixMilli())
	}
	if got[1].Identifier != "ab" || got[1].RegisteredAt != 0 {
		t.Fatalf("short identifier or zero time not preserved: %#v", got[1])
	}
}

func TestAffiliateUserRowsToTypesKeepsEmptyListShape(t *testing.T) {
	got := affiliateUserRowsToTypes(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty affiliate rows should return non-nil empty slice: %#v", got)
	}
}

func TestAffiliateCommissionAmountSumSelectStaysProjected(t *testing.T) {
	if !strings.Contains(affiliateCommissionAmountSumSelect, "SUM((content::jsonb->>'amount')::bigint)") {
		t.Fatal("affiliate commission total must sum the amount JSON field in SQL")
	}
	if affiliateCommissionAmountSumSelect == "content" || affiliateCommissionAmountSumSelect == `"content"` {
		t.Fatal("affiliate commission total must not select full content")
	}
}

func TestAffiliateStatsQueryUsesSingleProjectedStatement(t *testing.T) {
	normalized := strings.ToLower(affiliateStatsQuerySQL)
	if got := strings.Count(normalized, "select"); got != 3 {
		t.Fatalf("affiliate stats query should be one statement with two scalar subqueries, select count = %d", got)
	}
	for _, required := range []string{
		`from "user"`,
		"referer_id = ?",
		"deleted_at is null",
		"from system_logs",
		"type = ?",
		"object_id = ?",
		"as registers",
		"as total_commission",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("affiliate stats query missing %q: %s", required, affiliateStatsQuerySQL)
		}
	}
	if strings.Contains(normalized, "select *") {
		t.Fatal("affiliate stats query must not select full rows")
	}
}
