package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/types"
	task "github.com/neibniz/alpha/server/queue/types"
)

func TestTrafficStatisticsUsesAPITrafficShape(t *testing.T) {
	input := []types.UserTraffic{
		{SID: 1, Upload: 100, Download: 200},
		{SID: 2, Upload: 300, Download: 400},
	}

	report := types.TrafficReport{
		ReportId:        "00000000-0000-4000-8000-000000000001",
		SampledAt:       1_720_000_000_000,
		IntervalSeconds: 5,
		Sequence:        17,
		Traffic:         input,
	}
	payload := task.TrafficStatistics{
		ServerId: 11,
		Protocol: "xray",
		Reports:  []types.TrafficReport{report},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		ServerId int64 `json:"server_id"`
		Reports  []struct {
			ReportId        string `json:"report_id"`
			SampledAt       int64  `json:"sampled_at"`
			IntervalSeconds int64  `json:"interval_seconds"`
			Sequence        int64  `json:"sequence"`
			Traffic         []struct {
				SID      int64 `json:"uid"`
				Upload   int64 `json:"upload"`
				Download int64 `json:"download"`
			} `json:"traffic"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ServerId != 11 || len(decoded.Reports) != 1 || len(decoded.Reports[0].Traffic) != 2 {
		t.Fatalf("decoded payload = %#v", decoded)
	}
	if decoded.Reports[0].ReportId != report.ReportId || decoded.Reports[0].IntervalSeconds != 5 || decoded.Reports[0].Sequence != 17 {
		t.Fatalf("decoded report metadata = %#v", decoded.Reports[0])
	}
	if decoded.Reports[0].Traffic[0].SID != 1 || decoded.Reports[0].Traffic[0].Upload != 100 || decoded.Reports[0].Traffic[0].Download != 200 {
		t.Fatalf("decoded first log = %#v", decoded.Reports[0].Traffic[0])
	}
}

func TestNormalizeTrafficReportsSupportsV2AndLegacy(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	v2, err := normalizeTrafficReports(&types.ServerPushUserTrafficRequest{
		Reports: []types.TrafficReport{{
			ReportId:        "00000000-0000-4000-8000-000000000001",
			SampledAt:       now.Add(-5 * time.Second).UnixMilli(),
			IntervalSeconds: 5,
			Sequence:        3,
			Traffic:         []types.UserTraffic{{SID: 9, Download: 10}},
		}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(v2) != 1 || v2[0].IntervalSeconds != 5 || v2[0].Sequence != 3 {
		t.Fatalf("v2 reports = %#v", v2)
	}

	legacy, err := normalizeTrafficReports(&types.ServerPushUserTrafficRequest{
		Traffic: []types.UserTraffic{{SID: 9, Upload: 4}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy) != 1 || legacy[0].ReportId == "" || legacy[0].SampledAt != now.UnixMilli() || legacy[0].IntervalSeconds != 30 {
		t.Fatalf("legacy reports = %#v", legacy)
	}
}

func TestNormalizeTrafficReportsRejectsFutureSample(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	_, err := normalizeTrafficReports(&types.ServerPushUserTrafficRequest{
		Reports: []types.TrafficReport{{
			ReportId:        "00000000-0000-4000-8000-000000000001",
			SampledAt:       now.Add(6 * time.Minute).UnixMilli(),
			IntervalSeconds: 5,
			Traffic:         []types.UserTraffic{{SID: 9, Download: 10}},
		}},
	}, now)
	if err == nil {
		t.Fatal("expected future sample to be rejected")
	}
}

func TestNormalizeTrafficReportsRejectsNilRequest(t *testing.T) {
	if _, err := normalizeTrafficReports(nil, time.Now()); err == nil {
		t.Fatal("expected nil request to be rejected")
	}
}
