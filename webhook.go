package maxbot

import (
	"encoding/json"
	"net/url"
)

// SetWebhook registers a webhook URL with MAX API.
func (b *Bot) SetWebhook(url string, updateTypes []string, secret string) error {
	payload := map[string]interface{}{
		"url": url,
	}

	if len(updateTypes) > 0 {
		payload["update_types"] = updateTypes
	}

	if secret != "" {
		payload["secret"] = secret
	}

	return b.rawSimple("POST", "/subscriptions", payload)
}

// DeleteWebhook removes the webhook subscription for the given URL.
func (b *Bot) DeleteWebhook(webhookURL string) error {
	endpoint := "/subscriptions?url=" + url.QueryEscape(webhookURL)
	return b.rawSimple("DELETE", endpoint, nil)
}

// GetWebhook returns all active webhook subscriptions.
func (b *Bot) GetWebhook() ([]WebhookInfo, error) {
	data, err := b.Raw("GET", "/subscriptions", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Subscriptions []WebhookInfo `json:"subscriptions"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Subscriptions, nil
}
