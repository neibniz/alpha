package authMethod

import (
	"encoding/json"
	"testing"

	modelAuth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestAuthMethodConfigFromModelMapsEnabledPointer(t *testing.T) {
	enabled := true
	got := authMethodConfigFromModel(&modelAuth.Auth{
		Id:      7,
		Method:  "email",
		Enabled: &enabled,
	})
	if got.Id != 7 || got.Method != "email" || !got.Enabled {
		t.Fatalf("auth method mapping mismatch: %+v", got)
	}

	got = authMethodConfigFromModel(&modelAuth.Auth{Id: 8, Method: "mobile"})
	if got.Enabled {
		t.Fatalf("nil enabled pointer should map to false: %+v", got)
	}
}

func TestAuthMethodSummaryRowsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	enabled := true
	got := authMethodSummaryRows([]*modelAuth.Auth{
		nil,
		{Id: 7, Method: "email", Config: `{"secret":"omitted"}`, Enabled: &enabled},
	})
	if len(got) != 1 {
		t.Fatalf("auth method rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Method != "email" || !got[0].Enabled {
		t.Fatalf("auth method row mismatch: %+v", got[0])
	}
	itemPayload, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	var item map[string]any
	if err := json.Unmarshal(itemPayload, &item); err != nil {
		t.Fatal(err)
	}
	if _, ok := item["config"]; ok {
		t.Fatalf("auth method list summary must not expose config: %s", itemPayload)
	}

	empty := authMethodSummaryRows(nil)
	if empty == nil {
		t.Fatal("empty auth method rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.AuthMethodSummary `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty auth method rows marshal = %s, want list []", payload)
	}
}

func TestAuthMethodSummaryRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelAuth.Auth, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelAuth.Auth{Id: 1, Method: "email"}
	rows[len(rows)-1] = &modelAuth.Auth{Id: 2, Method: "mobile"}

	got := authMethodSummaryRows(rows)
	if len(got) != 2 {
		t.Fatalf("auth method rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy auth method rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("auth method row order changed: %#v", got)
	}
}

func TestAuthMethodSummaryRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelAuth.Auth, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelAuth.Auth{Id: int64(index + 1), Method: "email"}
	}

	got := authMethodSummaryRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("auth method rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("auth method rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("auth method row order changed: %#v", got)
	}
}

func TestAuthMethodConfigWithRawConfig(t *testing.T) {
	enabled := true
	got, err := authMethodConfigWithRawConfig(&modelAuth.Auth{
		Id:      9,
		Method:  "google",
		Enabled: &enabled,
		Config:  `{"client_id":"id","client_secret":"secret"}`,
	})
	if err != nil {
		t.Fatalf("raw config should be accepted: %v", err)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal auth method config: %v", err)
	}
	var decoded struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal marshaled auth config: %v", err)
	}
	if decoded.Config["client_id"] != "id" || decoded.Config["client_secret"] != "secret" || string(got.Config) == "" {
		t.Fatalf("config mismatch: %s", payload)
	}
}

func TestAuthMethodConfigWithRawConfigRejectsInvalidJSON(t *testing.T) {
	_, err := authMethodConfigWithRawConfig(&modelAuth.Auth{Config: `invalid`})
	if err == nil {
		t.Fatalf("invalid config should return error")
	}
}

func TestAuthMethodConfigForAdminRedactsGitHubSecret(t *testing.T) {
	enabled := true
	got, err := authMethodConfigForAdmin(&modelAuth.Auth{
		Id:      5,
		Method:  "github",
		Enabled: &enabled,
		Config:  `{"client_id":"id","client_secret":"secret","redirect_url":"https://panel.example/"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(got.Config, &config); err != nil {
		t.Fatal(err)
	}
	if _, exists := config["client_secret"]; exists {
		t.Fatalf("admin config leaked github client secret: %s", got.Config)
	}
	if configured, ok := config["client_secret_configured"].(bool); !ok || !configured {
		t.Fatalf("client_secret_configured = %#v", config["client_secret_configured"])
	}
}
