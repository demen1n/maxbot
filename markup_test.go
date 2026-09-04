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

func marshalReplyBtn(t *testing.T, btn ReplyButton) map[string]interface{} {
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

func TestReplyMarkupRow(t *testing.T) {
	rm := &ReplyMarkup{}
	rm.Row(rm.URL("a", "https://a"), rm.URL("b", "https://b"))
	rm.Row(rm.URL("c", "https://c"))

	if len(rm.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rm.InlineKeyboard))
	}
	if len(rm.InlineKeyboard[0]) != 2 || len(rm.InlineKeyboard[1]) != 1 {
		t.Errorf("unexpected row sizes: %+v", rm.InlineKeyboard)
	}
}

func TestReplyMarkupDataWithPayload(t *testing.T) {
	rm := &ReplyMarkup{}

	// Without a payload, Data is the button's own callback data.
	btn := rm.Data("Text", "raw-data")
	if btn.Payload != "raw-data" {
		t.Errorf("expected payload=raw-data, got %q", btn.Payload)
	}

	// With a payload value, it's JSON-marshaled into Payload.
	btn = rm.Data("Text", "raw-data", map[string]int{"n": 1})
	if btn.Payload != `{"n":1}` {
		t.Errorf("expected marshaled payload, got %q", btn.Payload)
	}

	// An unmarshalable payload falls back to the raw data string.
	btn = rm.Data("Text", "raw-data", make(chan int))
	if btn.Payload != "raw-data" {
		t.Errorf("expected fallback to raw-data on marshal error, got %q", btn.Payload)
	}
}

func TestReplyMarkupOpenAppContactID(t *testing.T) {
	rm := &ReplyMarkup{}

	btn := rm.OpenApp("Open", "app-1", "payload", 0)
	if btn.ContactID != 0 {
		t.Errorf("expected ContactID=0 when unset, got %d", btn.ContactID)
	}

	btn = rm.OpenApp("Open", "app-1", "payload", 42)
	if btn.ContactID != 42 {
		t.Errorf("expected ContactID=42, got %d", btn.ContactID)
	}
}

// ChatButton.uuid is a JSON integer per the MAX API schema, not a string --
// reused across message edits so the button doesn't spawn a new chat.
func TestChatButtonUUIDMarshalsAsInteger(t *testing.T) {
	rm := &ReplyMarkup{}
	btn := rm.Chat("New Chat", "My Group", "desc", "start")
	btn.ChatUUID = 123456789

	m := marshalBtn(t, btn)
	uuid, ok := m["uuid"].(float64)
	if !ok {
		t.Fatalf("expected uuid to decode as a JSON number, got %T (%v)", m["uuid"], m["uuid"])
	}
	if uuid != 123456789 {
		t.Errorf("expected uuid=123456789, got %v", uuid)
	}
}

// ReplyButton.MarshalJSON auto-sets "type" based on filled fields.
func TestReplyButtonMarshalJSON(t *testing.T) {
	kb := &ReplyKeyboard{}

	tests := []struct {
		name     string
		btn      ReplyButton
		wantType string
		check    func(t *testing.T, m map[string]interface{})
	}{
		{
			name:     "message",
			btn:      kb.Message("Hi", "hi-payload"),
			wantType: "message",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["payload"] != "hi-payload" {
					t.Errorf("expected payload=hi-payload, got %v", m["payload"])
				}
			},
		},
		{
			name:     "user_contact",
			btn:      kb.Contact("Share phone"),
			wantType: "user_contact",
		},
		{
			name:     "user_geo_location",
			btn:      kb.Geolocation("Share location", true),
			wantType: "user_geo_location",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["quick"] != true {
					t.Errorf("expected quick=true, got %v", m["quick"])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := marshalReplyBtn(t, tc.btn)
			if m["type"] != tc.wantType {
				t.Errorf("expected type=%q, got %q", tc.wantType, m["type"])
			}
			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

// ReplyKeyboard marshals as an attachment payload with buttons/direct/direct_user_id.
func TestReplyKeyboardAttachmentPayload(t *testing.T) {
	kb := &ReplyKeyboard{Direct: true, DirectUserID: 42}
	kb.Row(kb.Message("Hi", "hi"), kb.Contact("Share phone"))

	data, err := json.Marshal(kb.Buttons)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var rows [][]map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("expected 1 row of 2 buttons, got %+v", rows)
	}
	if rows[0][0]["type"] != "message" || rows[0][1]["type"] != "user_contact" {
		t.Errorf("unexpected button types: %+v", rows[0])
	}
}

// TASK-14: clipboard, chat, message button types.
func TestNewButtonTypesMarshal(t *testing.T) {
	rm := &ReplyMarkup{}

	tests := []struct {
		name     string
		btn      InlineButton
		wantType string
		check    func(t *testing.T, m map[string]interface{})
	}{
		{
			name:     "clipboard",
			btn:      rm.Clipboard("Copy", "copy-text"),
			wantType: "clipboard",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["payload"] != "copy-text" {
					t.Errorf("expected payload=copy-text, got %v", m["payload"])
				}
				if _, ok := m["clipboard_payload"]; ok {
					t.Error("clipboard_payload is not a real MAX API field; must not be serialized")
				}
			},
		},
		{
			name:     "chat",
			btn:      rm.Chat("New Chat", "My Group", "A group chat", "start"),
			wantType: "chat",
			check: func(t *testing.T, m map[string]interface{}) {
				if m["chat_title"] != "My Group" {
					t.Errorf("expected chat_title=My Group, got %v", m["chat_title"])
				}
			},
		},
		{
			name:     "message",
			btn:      rm.MessageBtn("Send"),
			wantType: "message",
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
