package http

import (
	"strings"
	"testing"
)

func TestMaskAPIKey(t *testing.T) {
	t.Run("mask long key", func(t *testing.T) {
		key := "7061807972fbda86d89f899bc73124dcbee53a5a31b0e526cdd157110a6a9be3"
		expected := key[:10] + strings.Repeat("*", len(key)-16) + key[len(key)-6:]
		if got := maskAPIKey(key); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("short key unchanged", func(t *testing.T) {
		key := "123456789012345" // length 15
		if got := maskAPIKey(key); got != key {
			t.Fatalf("expected %q, got %q", key, got)
		}
	})
}
