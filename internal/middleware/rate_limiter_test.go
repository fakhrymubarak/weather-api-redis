package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Note: The burst for both global and per-param is 2, so only 2 requests are allowed instantly.
// The 3rd request is blocked unless you wait for token refill (not practical for unit tests).

func TestRateLimitMiddleware_GlobalBurst(t *testing.T) {
	ResetVisitors()
	SetParamKey("location")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mw := RateLimitMiddleware(h)
	ip := "1.2.3.4:1234"
	w := httptest.NewRecorder()

	// 2 unique params should be allowed instantly (burst)
	for i := 0; i < 10; i++ {
		param := fmt.Sprintf("city%d", i)
		req := httptest.NewRequest("GET", "/weather?location="+param, nil)
		req.RemoteAddr = ip
		mw.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d on request %d", w.Result().StatusCode, i+1)
		}
		w = httptest.NewRecorder()
	}
	// Global burst should block the 11th request (new param)
	req := httptest.NewRequest("GET", "/weather?location=", nil)
	req.Header.Set("X-Forwarded-For", ip)

	req.RemoteAddr = ip
	mw.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d on 3rd request", w.Result().StatusCode)
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["error"].(string), "Rate limit exceeded") {
		t.Errorf("expected global limit error, got %v", resp["error"])
	}
}

func TestRateLimitMiddleware_PerParamBurst(t *testing.T) {
	ResetVisitors()
	SetParamKey("location")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mw := RateLimitMiddleware(h)
	ip := "2.3.4.5:2345"
	w := httptest.NewRecorder()

	// 2 requests to the same param allowed instantly (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/weather?location=London", nil)
		req.RemoteAddr = ip
		mw.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d on request %d", w.Result().StatusCode, i+1)
		}
		w = httptest.NewRecorder()
	}
	// Per-param burst should block the 3rd request to the same param
	req := httptest.NewRequest("GET", "/weather?location=London", nil)
	req.RemoteAddr = ip
	mw.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d on 3rd request", w.Result().StatusCode)
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["error"].(string), "Rate limit exceeded") {
		t.Errorf("expected per-param limit error, got %v", resp["error"])
	}
}

func TestCleanupGlobalVisitors_RemovesStaleEntries(t *testing.T) {
	ResetVisitors()
	ip := "9.8.7.6:9999"
	limiter := &visitor{
		limiter:  nil,
		lastSeen: time.Now().Add(-2 * time.Second), // older than 1s timeout in config_test.yaml
	}
	globalVisitors[ip] = limiter
	cleanupGlobalVisitorsOnce()
	muGlobal.Lock()
	_, exists := globalVisitors[ip]
	muGlobal.Unlock()
	if exists {
		t.Errorf("Expected global visitor to be cleaned up, but still exists")
	}
}

func TestCleanupParamVisitors_RemovesStaleEntries(t *testing.T) {
	ResetVisitors()
	ip := "8.7.6.5:8888"
	param := "testparam"
	limiter := &paramVisitor{
		limiter:  nil,
		lastSeen: time.Now().Add(-2 * time.Second), // older than 1s timeout in config_test.yaml
	}
	paramVisitors[ip] = map[string]*paramVisitor{param: limiter}
	cleanupParamVisitorsOnce()
	muParam.Lock()
	_, exists := paramVisitors[ip]
	muParam.Unlock()
	if exists {
		t.Errorf("Expected param visitor to be cleaned up, but still exists")
	}
}

func TestStartRateLimiterCleanup_DoesNotPanic(t *testing.T) {
	// Just ensure it starts goroutines without panic
	StartRateLimiterCleanup()
}
