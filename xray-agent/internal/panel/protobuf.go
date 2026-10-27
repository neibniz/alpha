package panel

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	agentv1 "github.com/neibniz/alpha/xray-agent/internal/protocol/alpha/agent/v1"
	"google.golang.org/protobuf/proto"
)

const (
	agentProtobufContentType = "application/vnd.alpha.agent.v1+protobuf"
	protobufRetryInterval    = 5 * time.Minute
	// A valid access batch can return up to 2,000 IP-limit decisions.
	maxPanelResponseBytes = 2 << 20
)

type protocolCapability struct {
	unsupportedUntil atomic.Int64
}

func (c *protocolCapability) shouldTry() bool {
	return time.Now().UnixNano() >= c.unsupportedUntil.Load()
}

func (c *protocolCapability) markSupported() {
	c.unsupportedUntil.Store(0)
}

func (c *protocolCapability) markUnsupported() {
	c.unsupportedUntil.Store(time.Now().Add(protobufRetryInterval).UnixNano())
}

func (c *Client) postTrafficReportsProtobuf(ctx context.Context, protocolName string, reports []TrafficReport) (bool, error) {
	if !c.trafficProtobuf.shouldTry() {
		return false, nil
	}
	message := &agentv1.PushTrafficReportsRequest{
		Protocol: protocolName,
		Reports:  make([]*agentv1.TrafficReport, 0, len(reports)),
	}
	for _, report := range reports {
		traffic := make([]*agentv1.UserTraffic, 0, len(report.Traffic))
		for _, row := range report.Traffic {
			traffic = append(traffic, &agentv1.UserTraffic{
				UserSubscribeId: row.SID,
				UploadBytes:     row.Upload,
				DownloadBytes:   row.Download,
			})
		}
		message.Reports = append(message.Reports, &agentv1.TrafficReport{
			ReportId:        report.ReportID,
			SampledAtMs:     report.SampledAt,
			IntervalSeconds: report.IntervalSeconds,
			Sequence:        report.Sequence,
			Traffic:         traffic,
		})
	}
	status, body, err := c.postProtobuf(ctx, "/v2/server/traffic-reports", message)
	if err != nil {
		return true, err
	}
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		c.trafficProtobuf.markSupported()
		return true, nil
	}
	if protobufUnsupportedStatus(status) {
		c.trafficProtobuf.markUnsupported()
		return false, nil
	}
	return true, protobufResponseError(status, body)
}

func (c *Client) postOnlineUsersProtobuf(ctx context.Context, update OnlineUsersUpdate) (bool, error) {
	if !c.onlineUsersProtobuf.shouldTry() {
		return false, nil
	}
	message := &agentv1.PushOnlineUsersRequest{
		Protocol:   "xray",
		Mode:       update.Mode,
		InstanceId: update.InstanceID,
		Sequence:   update.Seq,
		Active:     update.Active,
		Inactive:   update.Inactive,
		TtlSeconds: update.TTLSeconds,
	}
	status, body, err := c.postProtobuf(ctx, "/v2/server/online-users", message)
	if err != nil {
		return true, err
	}
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		c.onlineUsersProtobuf.markSupported()
		return true, nil
	}
	if protobufUnsupportedStatus(status) {
		c.onlineUsersProtobuf.markUnsupported()
		return false, nil
	}
	return true, protobufResponseError(status, body)
}

func (c *Client) postAccessEventsProtobuf(ctx context.Context, events []AccessEvent) (AccessPostResult, bool, error) {
	if !c.accessEventsProtobuf.shouldTry() {
		return AccessPostResult{}, false, nil
	}
	message := &agentv1.PushAccessEventsRequest{
		Protocol: "xray",
		Events:   make([]*agentv1.AccessEvent, 0, len(events)),
	}
	for _, event := range events {
		message.Events = append(message.Events, &agentv1.AccessEvent{
			EventId:         event.EventID,
			TimestampMs:     event.Timestamp,
			UserSubscribeId: event.UserSubscribeID,
			SourceIp:        event.SourceIP,
			SourcePort:      uint32(event.SourcePort),
			DestinationHost: event.DestinationHost,
			DestinationIp:   event.DestinationIP,
			DestinationPort: uint32(event.DestinationPort),
			Network:         event.Network,
			Status:          event.Status,
			OutboundTag:     event.OutboundTag,
			Reason:          event.Reason,
			ParserVersion:   uint32(event.ParserVersion),
		})
	}
	status, body, err := c.postProtobuf(ctx, "/v2/server/access-events", message)
	if err != nil {
		return AccessPostResult{}, true, err
	}
	if protobufUnsupportedStatus(status) {
		c.accessEventsProtobuf.markUnsupported()
		return AccessPostResult{}, false, nil
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return AccessPostResult{}, true, protobufResponseError(status, body)
	}
	var response agentv1.PushAccessEventsResponse
	if err := proto.Unmarshal(body, &response); err != nil {
		return AccessPostResult{}, true, fmt.Errorf("decode protobuf access response: %w", err)
	}
	result := AccessPostResult{
		Accepted:         int(response.GetAccepted()),
		Rejected:         int(response.GetRejected()),
		IPLimitDecisions: make([]IPLimitDecision, 0, len(response.GetIpLimitDecisions())),
	}
	for _, decision := range response.GetIpLimitDecisions() {
		if decision == nil {
			continue
		}
		result.IPLimitDecisions = append(result.IPLimitDecisions, IPLimitDecision{
			EventID:         decision.GetEventId(),
			UserSubscribeID: decision.GetUserSubscribeId(),
			SourceNetwork:   decision.GetSourceNetwork(),
			Blocked:         decision.GetBlocked(),
			ExpiresAt:       decision.GetExpiresAtMs(),
			Active:          decision.GetActive(),
			Limit:           decision.GetLimit(),
		})
	}
	if err := validateAccessAcknowledgement(result, len(events)); err != nil {
		return AccessPostResult{}, true, err
	}
	c.accessEventsProtobuf.markSupported()
	return result, true, nil
}

func (c *Client) postProtobuf(ctx context.Context, path string, message proto.Message) (int, []byte, error) {
	endpoint, err := c.url(path, nil)
	if err != nil {
		return 0, nil, err
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		return 0, nil, err
	}
	requestBody, contentEncoding, err := encodePanelPayload(payload)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(requestBody))
	if err != nil {
		return 0, nil, err
	}
	c.authenticate(request)
	request.Header.Set("Content-Type", agentProtobufContentType)
	request.Header.Set("Accept", agentProtobufContentType)
	if contentEncoding != "" {
		request.Header.Set("Content-Encoding", contentEncoding)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPanelResponseBytes+1))
	if err != nil {
		return 0, nil, err
	}
	if len(body) > maxPanelResponseBytes {
		return 0, nil, fmt.Errorf("panel response exceeds %d bytes", maxPanelResponseBytes)
	}
	return response.StatusCode, body, nil
}

func encodePanelPayload(payload []byte) ([]byte, string, error) {
	if len(payload) < 4<<10 {
		return payload, "", nil
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(payload); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), "gzip", nil
}

func protobufUnsupportedStatus(status int) bool {
	switch status {
	case http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusUnsupportedMediaType,
		http.StatusNotImplemented:
		return true
	default:
		return false
	}
}

func protobufResponseError(status int, body []byte) error {
	var response agentv1.ErrorResponse
	if len(body) > 0 && proto.Unmarshal(body, &response) == nil && response.GetMessage() != "" {
		return fmt.Errorf("panel returned http %d, code %d: %s", status, response.GetCode(), response.GetMessage())
	}
	return fmt.Errorf("panel returned http %d", status)
}
