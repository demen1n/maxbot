package maxbot

import (
	"encoding/json"
	"testing"
)

func marshalBtn(t *testing.T, btn InlineButton) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(&btn)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	return m
}

// TASK-8: MarshalJSON auto-sets "type" based on filled fields.
func TestInlineButtonMarshalJSON(t *testing.T) {
	rm := &ReplyMarkup{}

	tests := []struct {
		name     string
		btn      InlineButton
		wantType string
		check    func(t *testing.T, m map[string]interface{})
	}{
		{
			name:     "callback",
			btn:      rm.Data("Click", "my_payload"),
			wantType: "callback",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["payload"] != "my_payload" {
					t.Errorf("expected payload=my_payload, got %v", m["payload"])
				}
				if _, ok := m["callback_data"]; ok {
					t.Error("callback_data must not appear in output")
				}
			},
		},
		{
			name:     "link",
			btn:      rm.URL("Visit", "https://example.com"),
			wantType: "link",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["url"] != "https://example.com" {
					t.Errorf("expected url field, got %v", m["url"])
				}
			},
		},
		{
			name:     "request_contact",
			btn:      rm.Contact("Share phone"),
			wantType: "request_contact",
		},
		{
			name:     "request_geo_location",
			btn:      rm.Geolocation("Share location", false),
			wantType: "request_geo_location",
		},
		{
			name:     "open_app",
			btn:      rm.OpenApp("Open", "https://app.example.com", "start", 0),
			wantType: "open_app",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["web_app"] != "https://app.example.com" {
					t.Errorf("expected web_app field, got %v", m["web_app"])
				}
				if _, ok := m["app"]; ok {
					t.Error("old 'app' field must not appear in output")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := marshalBtn(t, tc.btn)
			if m["type"] != tc.wantType {
				t.Errorf("expected type=%q, got %q", tc.wantType, m["type"])
			}
			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}
