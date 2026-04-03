package db

import "testing"

func TestNewConnectionRequiresDSN(t *testing.T) {
	if _, err := NewConnection("   "); err == nil {
		t.Fatal("expected error for empty dsn")
	}
}
