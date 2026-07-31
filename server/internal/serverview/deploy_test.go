package serverview

import (
	"testing"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestDeployRowsSkipsNilAndMapsResponseShape(t *testing.T) {
	got := DeployRows([]*modelNode.Server{
		nil,
		{Id: 3, Name: "edge", Address: "edge.example.com", Country: "US", City: "Denver"},
	})

	if len(got) != 1 {
		t.Fatalf("expected nil row to be skipped, got %d rows", len(got))
	}
	if got[0].Id != 3 || got[0].Name != "edge" || got[0].Address != "edge.example.com" {
		t.Fatalf("deploy server row not mapped: %#v", got[0])
	}
}

func TestDeployRowsKeepEmptyListShape(t *testing.T) {
	got := DeployRows(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty deploy rows = %#v, want non-nil empty slice", got)
	}
}

func TestDeployRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelNode.Server, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelNode.Server{Id: 1, Name: "edge-a"}
	rows[len(rows)-1] = &modelNode.Server{Id: 2, Name: "edge-b"}

	got := DeployRows(rows)
	if len(got) != 2 {
		t.Fatalf("deploy rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy deploy rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("deploy rows order/ids not preserved: %#v", got)
	}
}

func TestDeployRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelNode.Server, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelNode.Server{Id: int64(index + 1), Name: "edge"}
	}

	got := DeployRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("deploy rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("deploy rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("deploy rows order/ids not preserved: %#v", got)
	}
}
