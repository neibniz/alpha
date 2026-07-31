package middleware

import "testing"

func TestPathHasSegment(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		segment string
		want    bool
	}{
		{name: "root admin", path: "/admin", segment: "admin", want: true},
		{name: "nested admin", path: "/api/admin/users", segment: "admin", want: true},
		{name: "trailing slash", path: "/admin/", segment: "admin", want: true},
		{name: "repeated slashes", path: "//api//admin//users", segment: "admin", want: true},
		{name: "relative path", path: "admin/users", segment: "admin", want: true},
		{name: "prefix is not segment", path: "/api/administrator", segment: "admin", want: false},
		{name: "suffix is not segment", path: "/api/user-admin", segment: "admin", want: false},
		{name: "missing segment", path: "/api/user", segment: "admin", want: false},
		{name: "empty path", path: "", segment: "admin", want: false},
		{name: "empty segment", path: "/admin", segment: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathHasSegment(tt.path, tt.segment); got != tt.want {
				t.Fatalf("pathHasSegment(%q, %q) = %v, want %v", tt.path, tt.segment, got, tt.want)
			}
		})
	}
}
