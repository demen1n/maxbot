package maxbot

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{"code only", &APIError{Code: 404}, "api error 404"},
		{"with error text", &APIError{Code: 400, ErrorText: "bad_request"}, "api error 400: bad_request"},
		{
			"all fields",
			&APIError{Code: 400, ErrorText: "bad_request", Message: "invalid.chat_id", Details: "chat not found"},
			"api error 400: bad_request (invalid.chat_id): chat not found",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAPIErrorIsAttachmentNotReady(t *testing.T) {
	if (&APIError{ErrorText: "attachment.not.ready"}).IsAttachmentNotReady() != true {
		t.Error("expected true when ErrorText matches")
	}
	if (&APIError{Message: "attachment.not.ready"}).IsAttachmentNotReady() != true {
		t.Error("expected true when Message matches")
	}
	if (&APIError{ErrorText: "other"}).IsAttachmentNotReady() != false {
		t.Error("expected false for unrelated error")
	}
}

func TestNetworkErrorErrorAndUnwrap(t *testing.T) {
	inner := errors.New("connection refused")
	e := &NetworkError{Op: "sendMessage", Err: inner}
	if !strings.Contains(e.Error(), "sendMessage") || !strings.Contains(e.Error(), "connection refused") {
		t.Errorf("unexpected Error(): %q", e.Error())
	}
	if errors.Unwrap(e) != inner {
		t.Error("expected Unwrap() to return the wrapped error")
	}
	if !errors.Is(e, inner) {
		t.Error("expected errors.Is to see through NetworkError")
	}
}

func TestTimeoutErrorErrorAndTimeout(t *testing.T) {
	e := &TimeoutError{Op: "getUpdates"}
	if e.Error() != "timeout during getUpdates" {
		t.Errorf("unexpected Error(): %q", e.Error())
	}
	if !e.Timeout() {
		t.Error("expected Timeout() to be true")
	}

	e2 := &TimeoutError{Op: "getUpdates", Reason: "context deadline exceeded"}
	if e2.Error() != "timeout during getUpdates: context deadline exceeded" {
		t.Errorf("unexpected Error() with reason: %q", e2.Error())
	}
}
