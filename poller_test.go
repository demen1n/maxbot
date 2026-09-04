package maxbot

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestLongPollerPollSendsUpdatesAndTracksMarker(t *testing.T) {
	callCount := 0
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"updates": []map[string]interface{}{{"update_type": "message_created"}},
				"marker":  5,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"updates": []interface{}{}, "marker": 5})
	}))

	p := &LongPoller{}
	updates := make(chan Update, 4)
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		p.Poll(b, updates, stop)
		close(done)
	}()

	select {
	case u := <-updates:
		if u.UpdateType != "message_created" {
			t.Errorf("unexpected update: %+v", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected an update from Poll")
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poll did not return after stop")
	}

	// Safe to read now: done's close happens-after Poll's last write to Marker.
	if p.Marker == nil || *p.Marker != 5 {
		t.Errorf("expected Marker=5, got %v", p.Marker)
	}
	if _, ok := <-updates; ok {
		t.Error("expected updates channel to be closed")
	}
}

func TestLongPollerPollRetriesOnError(t *testing.T) {
	callCount := 0
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "boom"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updates": []map[string]interface{}{{"update_type": "message_created"}},
		})
	}))

	p := &LongPoller{}
	updates := make(chan Update, 1)
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		p.Poll(b, updates, stop)
		close(done)
	}()

	// The retry path sleeps 1s after the first error before trying again.
	select {
	case <-updates:
	case <-time.After(3 * time.Second):
		t.Fatal("expected Poll to recover after a failed request")
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poll did not return after stop")
	}
}

// freeAddr reserves an ephemeral TCP port on 127.0.0.1 and releases it
// immediately so a test-owned http.Server can bind to a known address.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func TestWebhookPollEndToEnd(t *testing.T) {
	b := makeBot(t)
	hook := &Webhook{Listen: freeAddr(t), Endpoint: "/hook"}
	updates := make(chan Update, 2)
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		hook.Poll(b, updates, stop)
		close(done)
	}()

	url := "http://" + hook.Listen + "/hook"
	var resp *http.Response
	var err error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = http.Post(url, "application/json", bytes.NewReader(webhookUpdate()))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to reach webhook server: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case u := <-updates:
		if u.Message == nil || u.Message.Mid() != "mid.1" {
			t.Errorf("unexpected update: %+v", u)
		}
	case <-time.After(time.Second):
		t.Fatal("expected an update on the channel")
	}

	// Wrong HTTP method must be rejected.
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", resp2.StatusCode)
	}

	// Malformed JSON body must be rejected.
	resp3, err := http.Post(url, "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed body, got %d", resp3.StatusCode)
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poll did not return after stop")
	}
	if _, ok := <-updates; ok {
		t.Error("expected updates channel to be closed")
	}
}

func TestWebhookPollNoSecretAcceptsAnyRequest(t *testing.T) {
	b := makeBot(t)
	hook := &Webhook{Listen: freeAddr(t), Endpoint: "/hook"}
	updates := make(chan Update, 1)
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		hook.Poll(b, updates, stop)
		close(done)
	}()
	defer func() {
		close(stop)
		<-done
	}()

	url := "http://" + hook.Listen + "/hook"
	var resp *http.Response
	var err error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = http.Post(url, "application/json", bytes.NewReader(webhookUpdate()))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to reach webhook server: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with no secret configured, got %d", resp.StatusCode)
	}
}
