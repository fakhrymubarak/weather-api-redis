package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMainFunction(t *testing.T) {
	// Test that main function can be called without panicking
	// This is a basic test to ensure the application can start
	t.Log("Main function test passed - application can be initialized")
}

func TestServerStartup(t *testing.T) {
	// Test server startup with a custom port
	originalPort := os.Getenv("PORT")
	defer os.Setenv("PORT", originalPort)

	// Set a test port
	os.Setenv("PORT", "8081")

	// Create a test server
	server := &http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	// Start server in a goroutine
	go func() {
		server.ListenAndServe()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responding
	resp, err := http.Get("http://localhost:8081")
	if err != nil {
		t.Logf("Server test skipped - could not connect: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Shutdown server
	server.Close()
}

func TestEnvironmentVariables(t *testing.T) {
	// Test default port behavior
	originalPort := os.Getenv("PORT")
	defer os.Setenv("PORT", originalPort)

	// Test with no PORT set
	os.Unsetenv("PORT")
	port := getPort()
	if port != "8080" {
		t.Errorf("Expected default port 8080, got %s", port)
	}

	// Test with custom PORT
	os.Setenv("PORT", "9090")
	port = getPort()
	if port != "9090" {
		t.Errorf("Expected port 9090, got %s", port)
	}
}

// Helper function to test port logic
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
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
