package types

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestNodeRequestsRequireNonZeroPorts(t *testing.T) {
	validate := validator.New()
	tests := []struct {
		name    string
		request any
		valid   bool
	}{
		{name: "create rejects zero", request: CreateNodeRequest{}, valid: false},
		{name: "create accepts port", request: CreateNodeRequest{Port: 443}, valid: true},
		{name: "update rejects zero", request: UpdateNodeRequest{}, valid: false},
		{name: "update accepts port", request: UpdateNodeRequest{Port: 443}, valid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validate.Struct(test.request)
			if test.valid && err != nil {
				t.Fatalf("expected request to be valid: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected request validation to fail")
			}
		})
	}
}
