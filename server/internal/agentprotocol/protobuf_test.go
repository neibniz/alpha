package agentprotocol

import (
	"bytes"
	"testing"

	agentv1 "github.com/neibniz/alpha/server/internal/protocol/alpha/agent/v1"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestTrafficTaskRoundTripIsDeterministic(t *testing.T) {
	task := TrafficTask{
		ServerID: 7,
		Protocol: "xray",
		Reports: []types.TrafficReport{{
			ReportId:        "00000000-0000-4000-8000-000000000001",
			SampledAt:       1_720_000_000_000,
			IntervalSeconds: 5,
			Sequence:        17,
			Traffic:         []types.UserTraffic{{SID: 12, Upload: 100, Download: 200}},
		}},
	}
	first, err := MarshalTrafficTask(task)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalTrafficTask(task)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("deterministic marshal produced different task payloads")
	}
	decoded, err := UnmarshalTrafficTask(first)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ServerID != task.ServerID || decoded.Protocol != task.Protocol || len(decoded.Reports) != 1 || decoded.Reports[0].Traffic[0].Download != 200 {
		t.Fatalf("decoded task = %+v", decoded)
	}
}

func TestAccessRequestMapsFieldsAndRejectsOverflow(t *testing.T) {
	request, err := AccessRequest(&agentv1.PushAccessEventsRequest{
		Protocol: "xray",
		Events: []*agentv1.AccessEvent{{
			EventId:         "00000000-0000-5000-8000-000000000001",
			TimestampMs:     1_720_000_000_000,
			UserSubscribeId: 12,
			SourceIp:        "203.0.113.7",
			SourcePort:      443,
			DestinationHost: "example.com",
			DestinationPort: 8443,
			Network:         "tcp",
			Status:          "accepted",
			ParserVersion:   2,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Protocol != "xray" || len(request.Events) != 1 || request.Events[0].DestinationPort != 8443 || request.Events[0].ParserVersion != 2 {
		t.Fatalf("mapped request = %+v", request)
	}

	_, err = AccessRequest(&agentv1.PushAccessEventsRequest{Events: []*agentv1.AccessEvent{{SourcePort: 65_536}}})
	if err == nil {
		t.Fatal("expected an overflowing port to be rejected")
	}
}

func TestOnlineUsersAndAccessResponseMapping(t *testing.T) {
	online := OnlineUsersRequest(&agentv1.PushOnlineUsersRequest{
		Protocol:   "xray",
		Mode:       "delta",
		InstanceId: "agent-a",
		Sequence:   9,
		Active:     []int64{12, 13},
		Inactive:   []int64{14},
		TtlSeconds: 30,
	})
	if online.Protocol != "xray" || online.Seq != 9 || len(online.Active) != 2 || online.TTLSeconds != 30 {
		t.Fatalf("online request = %+v", online)
	}

	response := AccessResponse(&types.ServerPushAccessLogsResponse{
		Accepted: 1,
		IPLimitDecisions: []types.XrayIPLimitDecision{{
			EventId:         "00000000-0000-5000-8000-000000000001",
			UserSubscribeId: 12,
			SourceNetwork:   "203.0.113.7/32",
			Blocked:         true,
			ExpiresAt:       1_720_000_030_000,
			Active:          2,
			Limit:           2,
		}},
	})
	if response.GetAccepted() != 1 || len(response.GetIpLimitDecisions()) != 1 || !response.GetIpLimitDecisions()[0].GetBlocked() {
		t.Fatalf("access response = %+v", response)
	}
}
