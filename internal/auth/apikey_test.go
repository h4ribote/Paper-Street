package auth

import (
	"errors"
	"testing"
)

func TestParseAPIKeyHexValid(t *testing.T) {
	value := "000102030405060708090a0b0c0d0e0f10111213"
	key, err := ParseAPIKeyHex(value)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for i := 0; i < APIKeyByteLength; i++ {
		if key[i] != byte(i) {
			t.Fatalf("expected byte %d to be %d, got %d", i, i, key[i])
		}
	}
	if key.String() != value {
		t.Fatalf("expected hex string %q, got %q", value, key.String())
	}
}

func TestParseAPIKeyHexInvalidLength(t *testing.T) {
	_, err := ParseAPIKeyHex("abcdef")
	if !errors.Is(err, ErrInvalidAPIKeyLength) {
		t.Fatalf("expected invalid length error, got %v", err)
	}
}

func TestParseAPIKeyHexInvalidHex(t *testing.T) {
	_, err := ParseAPIKeyHex("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
	if errors.Is(err, ErrInvalidAPIKeyLength) {
		t.Fatal("expected hex decode error, got length error")
	}
}

func TestAPIKeyCacheContainsHex(t *testing.T) {
	cache := NewAPIKeyCache()
	value := "aabbccddeeff00112233445566778899aabbccdd"
	if err := cache.AddHex(value); err != nil {
		t.Fatalf("expected no error adding key, got %v", err)
	}
	if !cache.ContainsHex("  aabbccddeeff00112233445566778899aabbccdd ") {
		t.Fatal("expected cache to contain key with trimmed input")
	}
	if cache.ContainsHex("aabbccddeeff00112233445566778899aabbccde") {
		t.Fatal("expected cache miss for different key")
	}
}
