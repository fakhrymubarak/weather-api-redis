package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fakhrymubarak/weather-api-redis/internal/model"
	"golang.org/x/time/rate"
)

// paramKey is the query parameter key used for per-param rate limiting (default: "location").
var paramKey = "location"

// SetParamKey sets the query parameter key for per-param rate limiting. Used primarily for testing.
func SetParamKey(key string) {
	paramKey = key
}

// the visitor holds the rate limiter and last seen time for a specific IP address.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// paramVisitor holds the rate limiter and last seen time for a specific IP and parameter value.
type paramVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	// globalVisitors maps IP addresses to their corresponding visitor struct for global rate limiting.
	globalVisitors = make(map[string]*visitor) // key: ip
	// paramVisitors maps IP addresses and parameter values to their corresponding paramVisitor struct for per-param rate limiting.
	paramVisitors = make(map[string]map[string]*paramVisitor) // key: ip -> paramValue -> visitor
	muGlobal      sync.Mutex
	muParam       sync.Mutex
)

// getGlobalLimiter returns the rate limiter for the given IP address, creating one if it does not exist.
// The global limiter allows 10 requests per minute with a burst of 10.
func getGlobalLimiter(ip string) *rate.Limiter {
	muGlobal.Lock()
	defer muGlobal.Unlock()
	v, exists := globalVisitors[ip]
	if !exists {
		limiter := rate.NewLimiter(10.0/60.0, 10)
		globalVisitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// getParamLimiter returns the rate limiter for the given IP address and parameter value, creating one if it does not exist.
// The per-param limiter allows 2 requests per minute with a burst of 2.
func getParamLimiter(ip, param string) *rate.Limiter {
	muParam.Lock()
	defer muParam.Unlock()
	if _, ok := paramVisitors[ip]; !ok {
		paramVisitors[ip] = make(map[string]*paramVisitor)
	}
	v, exists := paramVisitors[ip][param]
	if !exists {
		limiter := rate.NewLimiter(2.0/60.0, 2)
		paramVisitors[ip][param] = &paramVisitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupGlobalVisitors periodically removes globalVisitors entries that have not been seen for over 3 minutes.
func cleanupGlobalVisitors() {
	for {
		time.Sleep(time.Minute)
		muGlobal.Lock()
		for ip, v := range globalVisitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(globalVisitors, ip)
			}
		}
		muGlobal.Unlock()
	}
}

// cleanupParamVisitors periodically removes paramVisitors entries that have not been seen for over 3 minutes.
func cleanupParamVisitors() {
	for {
		time.Sleep(time.Minute)
		muParam.Lock()
		for ip, paramMap := range paramVisitors {
			for param, v := range paramMap {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(paramMap, param)
				}
			}
			if len(paramMap) == 0 {
				delete(paramVisitors, ip)
			}
		}
		muParam.Unlock()
	}
}

// StartRateLimiterCleanup starts background goroutines to clean up stale visitors for both global and per-param limiters.
func StartRateLimiterCleanup() {
	go cleanupGlobalVisitors()
	go cleanupParamVisitors()
}

// ResetVisitors clears all visitor states for both global and per-param limiters. Used primarily for testing.
func ResetVisitors() {
	muGlobal.Lock()
	for k := range globalVisitors {
		delete(globalVisitors, k)
	}
	muGlobal.Unlock()
	muParam.Lock()
	for k := range paramVisitors {
		delete(paramVisitors, k)
	}
	muParam.Unlock()
}

// getIP extracts the client's IP address from the HTTP request, considering X-Forwarded-For headers.
func getIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // fallback
	}
	return ip
}

// getParam extracts the value of the configured query parameter from the HTTP request.
func getParam(r *http.Request) string {
	return r.URL.Query().Get(paramKey)
}

// RateLimitMiddleware returns an HTTP middleware that enforces global and per-parameter rate limiting.
// If the rate limit is exceeded, it responds with a 429 status and a JSON error message.
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		param := getParam(r)
		if param == "" {
			// If param is missing, treat as a single bucket
			param = "__none__"
		}
		globalLimiter := getGlobalLimiter(ip)
		paramLimiter := getParamLimiter(ip, param)
		if !globalLimiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			errMsg := "Rate limit exceeded: max 10 requests per minute per user/IP"
			resp := model.Response{
				Error:   &errMsg,
				Message: "Too Many Requests (global limit)",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if !paramLimiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			errMsg := "Rate limit exceeded: max 2 requests per minute per unique param per user/IP"
			resp := model.Response{
				Error:   &errMsg,
				Message: "Too Many Requests (per-param limit)",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		next.ServeHTTP(w, r)
	})
}
