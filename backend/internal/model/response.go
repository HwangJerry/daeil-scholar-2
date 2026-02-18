package model

// APIError is the standard error response format.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
