package bots

import "testing"

func TestFirstAPIKey(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "single", raw: "abc123", want: "abc123"},
		{name: "trimmed", raw: "  abc123  ", want: "abc123"},
		{name: "commaSeparated", raw: "first,second", want: "first"},
		{name: "leadingEmpty", raw: " , , second ", want: "second"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FirstAPIKey(test.raw); got != test.want {
				t.Fatalf("FirstAPIKey(%q) = %q, want %q", test.raw, got, test.want)
			}
		})
	}
}
