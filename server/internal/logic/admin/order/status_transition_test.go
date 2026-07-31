package order

import "testing"

func TestAllowedAdminOrderStatusTransition(t *testing.T) {
	tests := []struct {
		name    string
		current uint8
		target  uint8
		want    bool
	}{
		{name: "pending to paid", current: 1, target: 2, want: true},
		{name: "pending to cancelled", current: 1, target: 3, want: true},
		{name: "pending to closed", current: 1, target: 4, want: true},
		{name: "cancelled reopens", current: 3, target: 1, want: true},
		{name: "closed reopens", current: 4, target: 1, want: true},
		{name: "paid retry is idempotent", current: 2, target: 2, want: true},
		{name: "pending cannot skip activation", current: 1, target: 5, want: false},
		{name: "cancelled cannot become paid", current: 3, target: 2, want: false},
		{name: "closed cannot become finished", current: 4, target: 5, want: false},
		{name: "paid cannot move backwards", current: 2, target: 1, want: false},
		{name: "finished is immutable", current: 5, target: 2, want: false},
		{name: "zero target is invalid", current: 1, target: 0, want: false},
		{name: "unknown current is invalid", current: 9, target: 1, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := allowedAdminOrderStatusTransition(test.current, test.target); got != test.want {
				t.Fatalf("allowedAdminOrderStatusTransition(%d, %d) = %v, want %v", test.current, test.target, got, test.want)
			}
		})
	}
}
