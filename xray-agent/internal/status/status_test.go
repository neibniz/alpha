package status

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPayloadMarshalIncludesEmptyClearableStatusFields(t *testing.T) {
	data, err := json.Marshal(Payload{ServerID: 1})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if !strings.Contains(string(data), `"xray_stats_error":""`) {
		t.Fatalf("payload does not clear xray stats error: %s", data)
	}
	if !strings.Contains(string(data), `"pending_config_hash":""`) {
		t.Fatalf("payload does not clear pending config hash: %s", data)
	}
	if !strings.Contains(string(data), `"last_config_error":""`) {
		t.Fatalf("payload does not clear last config error: %s", data)
	}
	if !strings.Contains(string(data), `"last_apply_error":""`) {
		t.Fatalf("payload does not clear last apply error: %s", data)
	}
}
