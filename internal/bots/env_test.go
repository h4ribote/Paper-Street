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

func TestParseInt64List(t *testing.T) {
	values, err := ParseInt64List("1, 2,3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
		t.Fatalf("unexpected values: %#v", values)
	}
	values, err = ParseInt64List(" , ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected empty list, got %#v", values)
	}
}
