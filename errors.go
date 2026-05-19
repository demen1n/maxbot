package maxbot

import "fmt"

// APIError represents an error response from the MAX API.
// The API body has three fields: error (short code), code (dot-separated key), message (human text).
type APIError struct {
	Code      int    // HTTP status code
	ErrorText string // "error" field — short machine-readable description
	Message   string // "code" field — dot-separated error key
	Details   string // "message" field — human-readable description
}

func (e *APIError) Error() string {
	parts := fmt.Sprintf("api error %d", e.Code)
	if e.ErrorText != "" {
		parts += ": " + e.ErrorText
	}
	if e.Message != "" {
		parts += " (" + e.Message + ")"
	}
	if e.Details != "" {
		parts += ": " + e.Details
	}
	return parts
}

// IsAttachmentNotReady reports whether the error means the uploaded attachment
// has not been processed by MAX yet and the request should be retried.
func (e *APIError) IsAttachmentNotReady() bool {
	return e.ErrorText == "attachment.not.ready" || e.Message == "attachment.not.ready"
}

// NetworkError wraps a network-level failure.
type NetworkError struct {
	Op  string
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s: %v", e.Op, e.Err)
}

func (e *NetworkError) Unwrap() error { return e.Err }

// TimeoutError represents a timeout during an API operation.
type TimeoutError struct {
	Op     string
	Reason string
}

func (e *TimeoutError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("timeout during %s: %s", e.Op, e.Reason)
	}
	return fmt.Sprintf("timeout during %s", e.Op)
}

func (e *TimeoutError) Timeout() bool { return true }
