package traffic

import (
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	internaltypes "github.com/neibniz/alpha/server/internal/types"
	queuetypes "github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

func TestDecodeTrafficStatisticsTaskSupportsProtobufAndLegacyJSON(t *testing.T) {
	report := internaltypes.TrafficReport{
		ReportId:        "00000000-0000-4000-8000-000000000001",
		SampledAt:       1_720_000_000_000,
		IntervalSeconds: 5,
		Sequence:        17,
		Traffic:         []internaltypes.UserTraffic{{SID: 12, Upload: 100, Download: 200}},
	}
	protobufPayload, err := agentprotocol.MarshalTrafficTask(agentprotocol.TrafficTask{ServerID: 7, Protocol: "xray", Reports: []internaltypes.TrafficReport{report}})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeTrafficStatisticsTask(asynq.NewTask(queuetypes.ForthwithTrafficStatisticsV2, protobufPayload))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ServerId != 7 || decoded.Protocol != "xray" || len(decoded.Reports) != 1 || decoded.Reports[0].Sequence != 17 {
		t.Fatalf("protobuf task = %+v", decoded)
	}

	legacy := queuetypes.TrafficStatistics{ServerId: 8, Protocol: "xray", Reports: []internaltypes.TrafficReport{report}}
	legacyPayload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = decodeTrafficStatisticsTask(asynq.NewTask(queuetypes.ForthwithTrafficStatistics, legacyPayload))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ServerId != 8 || len(decoded.Reports) != 1 {
		t.Fatalf("legacy task = %+v", decoded)
	}
}

func TestDecodeTrafficStatisticsTaskRejectsMalformedProtobuf(t *testing.T) {
	if _, err := decodeTrafficStatisticsTask(asynq.NewTask(queuetypes.ForthwithTrafficStatisticsV2, []byte{0xff})); err == nil {
		t.Fatal("expected malformed protobuf task to fail")
	}
}
