package maxbot

import (
	"encoding/json"
	"fmt"
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

	data, err := b.Raw("POST", "/subscriptions", payload)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to set webhook: %s", result.Message)
	}

	return nil
}

// DeleteWebhook removes the webhook subscription.
func (b *Bot) DeleteWebhook() error {
	data, err := b.Raw("DELETE", "/subscriptions", nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to delete webhook: %s", result.Message)
	}

	return nil
}

// GetWebhook returns current webhook information.
func (b *Bot) GetWebhook() (*WebhookInfo, error) {
	data, err := b.Raw("GET", "/subscriptions", nil)
	if err != nil {
		return nil, err
	}

	var info WebhookInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}
