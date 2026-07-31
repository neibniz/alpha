package user

import (
	"reflect"
	"testing"
)

func TestUnbindDeviceLookupColumnsStayMinimal(t *testing.T) {
	want := []string{"id", "user_id", "identifier"}
	if !reflect.DeepEqual(unbindDeviceLookupColumns, want) {
		t.Fatalf("unbind device lookup columns = %#v, want %#v", unbindDeviceLookupColumns, want)
	}
}
