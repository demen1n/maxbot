package maxbot

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Poller is an interface for receiving updates.
type Poller interface {
	Poll(b *Bot, updates chan Update, stop chan struct{})
}

// LongPoller implements long polling for receiving updates.
type LongPoller struct {
	Limit   int
	Timeout time.Duration
	Marker  *int64
}

// Poll starts the long polling loop.
func (p *LongPoller) Poll(b *Bot, updates chan Update, stop chan struct{}) {
	b.log("Long poller started with timeout: %v", p.Timeout)

	for {
		select {
		case <-stop:
			b.log("Stop signal received")
			close(updates)
			return
		default:
			upds, marker, err := b.getUpdates(p.Marker, p.Limit, int(p.Timeout.Seconds()))
			if err != nil {
				b.log("Error getting updates: %v", err)
				time.Sleep(time.Second)
				continue
			}

			if len(upds) > 0 {
				b.log("Received %d updates, new marker: %v", len(upds), marker)
			}

			for _, u := range upds {
				updates <- u
			}

			if marker != nil {
				p.Marker = marker
			}
		}
	}
}

// Webhook implements webhook receiver for getting updates.
type Webhook struct {
	Listen   string // адрес для прослушивания, например ":8443"
	Endpoint string // endpoint для webhook, например "/webhook"
	URL      string // публичный URL бота для регистрации
	Secret   string // секретный ключ для проверки

	server *http.Server
}

// Poll starts the webhook server.
func (w *Webhook) Poll(b *Bot, updates chan Update, stop chan struct{}) {
	if w.Listen == "" {
		w.Listen = ":8443"
	}
	if w.Endpoint == "" {
		w.Endpoint = "/webhook"
	}

	mux := http.NewServeMux()

	mux.HandleFunc(w.Endpoint, func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if w.Secret != "" {
			if r.Header.Get("X-Webhook-Secret") != w.Secret {
				http.Error(rw, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var update Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			b.log("Failed to decode webhook update: %v", err)
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}

		updates <- update
		rw.WriteHeader(http.StatusOK)
	})

	w.server = &http.Server{
		Addr:    w.Listen,
		Handler: mux,
	}

	go func() {
		b.log("Webhook server started on %s%s", w.Listen, w.Endpoint)
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.log("Webhook server error: %v", err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.server.Shutdown(ctx); err != nil {
		b.log("Webhook server shutdown error: %v", err)
	}

	close(updates)
}
