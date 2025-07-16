package model

// Response is a generic struct for API responses
type Response struct {
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
	Message string      `json:"message"`
}
