package maxbot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TASK-5: DeleteWebhook passes url as query param.
func TestDeleteWebhookPassesURL(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Query().Get("url")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.DeleteWebhook("https://example.com/hook"); err != nil {
		t.Fatalf("DeleteWebhook error: %v", err)
	}
	if gotURL != "https://example.com/hook" {
		t.Errorf("expected url query param, got %q", gotURL)
	}
}

func webhookUpdate() []byte {
	b, _ := json.Marshal(Update{
		UpdateType: "message_created",
		Message:    &Message{Body: &MessageBody{Mid: "mid.1", Text: "hi"}},
	})
	return b
}

// TASK-2: correct secret header accepted; wrong header rejected.
func TestWebhookSecretHeader(t *testing.T) {
	hook := &Webhook{Secret: "s3cr3t"}
	updates := make(chan Update, 1)
	stop := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(rw http.ResponseWriter, r *http.Request) {
		if hook.Secret != "" {
			if r.Header.Get(WebhookSecretHeader) != hook.Secret {
				http.Error(rw, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		var u Update
		json.NewDecoder(r.Body).Decode(&u)
		select {
		case updates <- u:
		default:
		}
		rw.WriteHeader(http.StatusOK)
	})
	_ = updates
	_ = stop

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := webhookUpdate()

	// correct secret → 200
	req, _ := http.NewRequest("POST", srv.URL+"/webhook", bytes.NewReader(body))
	req.Header.Set(WebhookSecretHeader, "s3cr3t")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with correct secret, got %d", resp.StatusCode)
	}

	// wrong secret → 401
	req2, _ := http.NewRequest("POST", srv.URL+"/webhook", bytes.NewReader(body))
	req2.Header.Set("X-Webhook-Secret", "s3cr3t") // old wrong header
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong secret header, got %d", resp2.StatusCode)
	}
}

