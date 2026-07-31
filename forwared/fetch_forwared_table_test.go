package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestTableFromForwardItemsUsesActualHashWhenApplyIsPartial(t *testing.T) {
	desired := &ForwaredTable{
		InstanceID: "instance",
		Version:    "v1-desired",
		Hash:       "desired-hash",
		ForwaredItems: []ForwaredItem{
			{RuleID: "a", LocalPort: 1001, TargetHost: "10.0.0.1", TargetPort: 443},
			{RuleID: "b", LocalPort: 1002, TargetHost: "10.0.0.2", TargetPort: 443},
		},
	}
	actual := tableFromForwardItems(desired, map[uint][]ForwaredItem{
		1001: {desired.ForwaredItems[0]},
	})
	if actual.Hash == "" || actual.Hash == desired.Hash {
		t.Fatalf("partial table must have its own hash: %q", actual.Hash)
	}
	if len(actual.ForwaredItems) != 1 || actual.Version[:7] != "actual-" {
		t.Fatalf("unexpected actual table: %#v", actual)
	}
}

func TestTableFromForwardItemsPreservesDesiredVersionWhenHashesMatch(t *testing.T) {
	desired := &ForwaredTable{
		InstanceID: "instance",
		Version:    "v1-desired",
		ForwaredItems: []ForwaredItem{
			{RuleID: "a", LocalPort: 1001, TargetHost: "10.0.0.1", TargetPort: 443},
		},
	}
	first := tableFromForwardItems(desired, groupForwardItems(desired))
	desired.Hash = first.Hash
	actual := tableFromForwardItems(desired, groupForwardItems(desired))
	if actual.Version != desired.Version {
		t.Fatalf("expected desired version %q, got %q", desired.Version, actual.Version)
	}
}

func TestTableFromForwardItemsMatchesServerWireHash(t *testing.T) {
	wire := `[{"rule_id":"a","listen_port":1001,"target_host":"10.0.0.1","target_port":443}]`
	sum := sha256.Sum256([]byte(wire))
	desired := &ForwaredTable{
		Version: "v1-wire",
		Hash:    hex.EncodeToString(sum[:]),
		ForwaredItems: []ForwaredItem{
			{RuleID: "a", LocalPort: 1001, TargetHost: "10.0.0.1", TargetPort: 443},
		},
	}
	actual := tableFromForwardItems(desired, groupForwardItems(desired))
	if actual.Hash != desired.Hash || actual.Version != desired.Version {
		t.Fatalf("wire-equivalent rule must remain synchronized: %#v", actual)
	}
}
