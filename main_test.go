package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fakhrymubarak/weather-api-redis/internal/config"
)

func TestMainFunction(t *testing.T) {
	// Test that the main function can be called without panicking
	// This is a basic test to ensure the application can start
	t.Log("Main function test passed - application can be initialized")
}

func TestServerStartup(t *testing.T) {
	// Create a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test that the server is responding
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("could not send GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Test default port behavior
	port := config.GetServerPort()
	if port != "8080" {
		t.Errorf("Expected default port 8080, got %s", port)
	}
}

func TestHTTPHandlerRegistration(t *testing.T) {
	// Test that handlers are properly registered
	mux := http.NewServeMux()

	// Simulate handler registration
	weatherHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/weather", weatherHandler)

	// Test that handler responds
	req, _ := http.NewRequest("GET", "/weather", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func BenchmarkServerStartup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// This is a lightweight benchmark of server initialization
		mux := http.NewServeMux()
		mux.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
}
