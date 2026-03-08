package maxbot

import "fmt"

// APIError represents an error response from the MAX API.
type APIError struct {
	Code    int
	Message string
	Details string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("api error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}

// IsAttachmentNotReady reports whether the error means the uploaded attachment
// has not been processed by MAX yet and the request should be retried.
func (e *APIError) IsAttachmentNotReady() bool {
	return e.Message == "attachment.not.ready"
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
