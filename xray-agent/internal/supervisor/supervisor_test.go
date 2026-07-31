package supervisor

import (
	"context"
	"reflect"
	"testing"
)

func TestXrayTestCommandUsesRunTest(t *testing.T) {
	cmd := xrayTestCommand(context.Background(), "xray", "/tmp/config.json")
	want := []string{"xray", "run", "-test", "-config", "/tmp/config.json"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}
