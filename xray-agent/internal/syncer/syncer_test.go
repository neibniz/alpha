package syncer

import (
	"testing"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

func TestEffectiveUsersUsesCacheForUnchangedVersion(t *testing.T) {
	runner := &Runner{
		userVersion: "v1",
		userCache: []panel.XrayUser{
			{Email: "u_1", UUID: "uuid-1", Inbounds: []string{"main"}},
		},
	}

	users, err := runner.effectiveUsers(panel.XrayUsersResponse{
		Version: "v1",
		Full:    false,
	})
	if err != nil {
		t.Fatalf("effectiveUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Email != "u_1" {
		t.Fatalf("effectiveUsers() = %#v", users)
	}

	users[0].Inbounds[0] = "mutated"
	if runner.userCache[0].Inbounds[0] != "main" {
		t.Fatalf("effectiveUsers() returned mutable cache reference")
	}
}

func TestEffectiveUsersRefreshesCacheForFullResponse(t *testing.T) {
	runner := &Runner{
		userVersion: "v1",
		userCache: []panel.XrayUser{
			{Email: "u_1", UUID: "uuid-1"},
		},
	}

	users, err := runner.effectiveUsers(panel.XrayUsersResponse{
		Version: "v2",
		Full:    true,
		Users: []panel.XrayUser{
			{Email: "u_2", UUID: "uuid-2"},
		},
	})
	if err != nil {
		t.Fatalf("effectiveUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Email != "u_2" {
		t.Fatalf("effectiveUsers() = %#v", users)
	}
	if len(runner.userCache) != 1 || runner.userCache[0].Email != "u_2" {
		t.Fatalf("userCache = %#v", runner.userCache)
	}
}
