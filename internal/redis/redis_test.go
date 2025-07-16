package redis

import (
	"testing"
)

func TestGetClient(t *testing.T) {
	client := GetClient()
	if client == nil {
		t.Error("Expected Redis client to be created")
	}

	// Test that we can get the same client multiple times (singleton pattern)
	client2 := GetClient()
	if client != client2 {
		t.Error("Expected same client instance (singleton pattern)")
	}
}

func TestGetContext(t *testing.T) {
	ctx := GetContext()
	if ctx == nil {
		t.Error("Expected context to be created")
	}

	// Test that context is not cancelled
	select {
	case <-ctx.Done():
		t.Error("Expected context to not be cancelled")
	default:
		// Context is not cancelled, which is expected
	}
}

func TestRedisSingletonPattern(t *testing.T) {
	// Test that multiple calls return the same client
	client1 := GetClient()
	client2 := GetClient()
	client3 := GetClient()

	if client1 != client2 || client2 != client3 {
		t.Error("Expected all clients to be the same instance (singleton)")
	}
}

func TestResetClientForTest(t *testing.T) {
	client1 := GetClient()
	ResetClientForTest()
	client2 := GetClient()
	if client1 == client2 {
		t.Error("Expected a new client instance after reset")
	}
}

func BenchmarkGetClient(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetClient()
	}
}

func BenchmarkGetContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetContext()
	}
}
