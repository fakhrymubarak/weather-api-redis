package repository

import "net/http"

// RoundTripperFunc allows us to easily mock http.Client responses in tests.
type RoundTripperFunc func(*http.Request) *http.Response

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
