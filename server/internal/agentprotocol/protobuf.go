package agentprotocol

import (
	"fmt"
	"math"

	agentv1 "github.com/neibniz/alpha/server/internal/protocol/alpha/agent/v1"
	"github.com/neibniz/alpha/server/internal/types"
	"google.golang.org/protobuf/proto"
)

const ContentType = "application/vnd.alpha.agent.v1+protobuf"

type TrafficTask struct {
	ServerID int64
	Protocol string
	Reports  []types.TrafficReport
}

func TrafficRequest(message *agentv1.PushTrafficReportsRequest) *types.ServerPushUserTrafficRequest {
	if message == nil {
		return &types.ServerPushUserTrafficRequest{}
	}
	return &types.ServerPushUserTrafficRequest{
		ServerCommon: types.ServerCommon{Protocol: message.GetProtocol()},
		Reports:      trafficReportsFromProto(message.GetReports()),
	}
}

func AccessRequest(message *agentv1.PushAccessEventsRequest) (*types.ServerPushAccessLogsRequest, error) {
	request := &types.ServerPushAccessLogsRequest{}
	if message == nil {
		return request, nil
	}
	request.Protocol = message.GetProtocol()
	request.Events = make([]types.XrayAccessEvent, 0, len(message.GetEvents()))
	for _, event := range message.GetEvents() {
		if event == nil {
			continue
		}
		if event.GetSourcePort() > math.MaxUint16 || event.GetDestinationPort() > math.MaxUint16 {
			return nil, fmt.Errorf("access event port exceeds %d", uint32(math.MaxUint16))
		}
		if event.GetParserVersion() > math.MaxUint16 {
			return nil, fmt.Errorf("access event parser version exceeds %d", uint32(math.MaxUint16))
		}
		request.Events = append(request.Events, types.XrayAccessEvent{
			EventId:         event.GetEventId(),
			Timestamp:       event.GetTimestampMs(),
			UserSubscribeId: event.GetUserSubscribeId(),
			SourceIP:        event.GetSourceIp(),
			SourcePort:      uint16(event.GetSourcePort()),
			DestinationHost: event.GetDestinationHost(),
			DestinationIP:   event.GetDestinationIp(),
			DestinationPort: uint16(event.GetDestinationPort()),
			Network:         event.GetNetwork(),
			Status:          event.GetStatus(),
			OutboundTag:     event.GetOutboundTag(),
			Reason:          event.GetReason(),
			ParserVersion:   uint16(event.GetParserVersion()),
		})
	}
	return request, nil
}

func AccessResponse(response *types.ServerPushAccessLogsResponse) *agentv1.PushAccessEventsResponse {
	if response == nil {
		return &agentv1.PushAccessEventsResponse{}
	}
	decisions := make([]*agentv1.IPLimitDecision, 0, len(response.IPLimitDecisions))
	for _, decision := range response.IPLimitDecisions {
		decisions = append(decisions, &agentv1.IPLimitDecision{
			EventId:         decision.EventId,
			UserSubscribeId: decision.UserSubscribeId,
			SourceNetwork:   decision.SourceNetwork,
			Blocked:         decision.Blocked,
			ExpiresAtMs:     decision.ExpiresAt,
			Active:          decision.Active,
			Limit:           decision.Limit,
		})
	}
	return &agentv1.PushAccessEventsResponse{
		Accepted:         uint32(response.Accepted),
		Rejected:         uint32(response.Rejected),
		IpLimitDecisions: decisions,
	}
}

func OnlineUsersRequest(message *agentv1.PushOnlineUsersRequest) *types.ServerPushOnlineUsersRequest {
	if message == nil {
		return &types.ServerPushOnlineUsersRequest{}
	}
	return &types.ServerPushOnlineUsersRequest{
		ServerCommon: types.ServerCommon{Protocol: message.GetProtocol()},
		Mode:         message.GetMode(),
		InstanceID:   message.GetInstanceId(),
		Seq:          message.GetSequence(),
		Active:       append([]int64(nil), message.GetActive()...),
		Inactive:     append([]int64(nil), message.GetInactive()...),
		TTLSeconds:   message.GetTtlSeconds(),
	}
}

func MarshalTrafficTask(task TrafficTask) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(&agentv1.TrafficStatisticsTask{
		ServerId: task.ServerID,
		Protocol: task.Protocol,
		Reports:  trafficReportsToProto(task.Reports),
	})
}

func UnmarshalTrafficTask(payload []byte) (TrafficTask, error) {
	var message agentv1.TrafficStatisticsTask
	if err := proto.Unmarshal(payload, &message); err != nil {
		return TrafficTask{}, err
	}
	return TrafficTask{
		ServerID: message.GetServerId(),
		Protocol: message.GetProtocol(),
		Reports:  trafficReportsFromProto(message.GetReports()),
	}, nil
}

func trafficReportsToProto(reports []types.TrafficReport) []*agentv1.TrafficReport {
	result := make([]*agentv1.TrafficReport, 0, len(reports))
	for _, report := range reports {
		traffic := make([]*agentv1.UserTraffic, 0, len(report.Traffic))
		for _, row := range report.Traffic {
			traffic = append(traffic, &agentv1.UserTraffic{
				UserSubscribeId: row.SID,
				UploadBytes:     row.Upload,
				DownloadBytes:   row.Download,
			})
		}
		result = append(result, &agentv1.TrafficReport{
			ReportId:        report.ReportId,
			SampledAtMs:     report.SampledAt,
			IntervalSeconds: report.IntervalSeconds,
			Sequence:        report.Sequence,
			Traffic:         traffic,
		})
	}
	return result
}

func trafficReportsFromProto(reports []*agentv1.TrafficReport) []types.TrafficReport {
	result := make([]types.TrafficReport, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		traffic := make([]types.UserTraffic, 0, len(report.GetTraffic()))
		for _, row := range report.GetTraffic() {
			if row == nil {
				continue
			}
			traffic = append(traffic, types.UserTraffic{
				SID:      row.GetUserSubscribeId(),
				Upload:   row.GetUploadBytes(),
				Download: row.GetDownloadBytes(),
			})
		}
		result = append(result, types.TrafficReport{
			ReportId:        report.GetReportId(),
			SampledAt:       report.GetSampledAtMs(),
			IntervalSeconds: report.GetIntervalSeconds(),
			Sequence:        report.GetSequence(),
			Traffic:         traffic,
		})
	}
	return result
}
