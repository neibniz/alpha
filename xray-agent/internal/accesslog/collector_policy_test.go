package accesslog

import (
	"context"
	"errors"
	"testing"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

type testEventPoster struct {
	result panel.AccessPostResult
}

func (p testEventPoster) PostAccessEvents(context.Context, []panel.AccessEvent) (panel.AccessPostResult, error) {
	return p.result, nil
}

type testPolicySink struct {
	decisions []panel.IPLimitDecision
	err       error
}

func (s *testPolicySink) ApplyIPLimitDecisions(_ context.Context, decisions []panel.IPLimitDecision) error {
	s.decisions = append([]panel.IPLimitDecision(nil), decisions...)
	return s.err
}

func TestFlushAppliesPolicyAndAcknowledgesIndependently(t *testing.T) {
	dir := t.TempDir()
	decision := panel.IPLimitDecision{UserSubscribeID: 12, SourceNetwork: "203.0.113.7/32", Blocked: true}
	sink := &testPolicySink{err: errors.New("xray api temporarily unavailable")}
	collector, err := NewCollector(7, "", testAccessConfig(dir+"/spool", dir+"/state"), testEventPoster{
		result: panel.AccessPostResult{Accepted: 1, IPLimitDecisions: []panel.IPLimitDecision{decision}},
	}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collector.spool.appendRecords([]collectedRecord{{
		Event: panel.AccessEvent{EventID: "event", Status: "accepted"}, EndOffset: 3,
	}}, 1, 2, 0); err != nil {
		t.Fatal(err)
	}
	if err := collector.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.Pending != 0 {
		t.Fatalf("pending = %d, want 0", collector.spool.state.Pending)
	}
	if len(sink.decisions) != 1 || sink.decisions[0] != decision {
		t.Fatalf("decisions = %+v", sink.decisions)
	}
}

func TestFlushIncompleteAcknowledgementKeepsEventQueued(t *testing.T) {
	dir := t.TempDir()
	collector, err := NewCollector(7, "", testAccessConfig(dir+"/spool", dir+"/state"), testEventPoster{
		result: panel.AccessPostResult{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collector.spool.appendRecords([]collectedRecord{{
		Event: panel.AccessEvent{EventID: "event", Status: "accepted"}, EndOffset: 3,
	}}, 1, 2, 0); err != nil {
		t.Fatal(err)
	}
	if err := collector.flush(context.Background()); err == nil {
		t.Fatal("incomplete acknowledgement should fail")
	}
	if collector.spool.state.Pending != 1 || collector.spool.state.AckOffset != 0 {
		t.Fatalf("state = %+v", collector.spool.state)
	}
}
