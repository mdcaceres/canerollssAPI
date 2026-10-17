package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type WhatsAppClient struct {
	phoneNumberID string
	apiToken      string
	httpClient    *http.Client
}

func NewWhatsAppClient() *WhatsAppClient {
	return &WhatsAppClient{
		phoneNumberID: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		apiToken:      os.Getenv("WHATSAPP_API_TOKEN"),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

type textMessagePayload struct {
	MessagingProduct string       `json:"messaging_product"`
	RecipientType    string       `json:"recipient_type"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             textBodyData `json:"text"`
}

type textBodyData struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

func (c *WhatsAppClient) SendTextMessage(ctx context.Context, to, message string) error {
	if c.phoneNumberID == "" || c.apiToken == "" || c.apiToken == "mock_token" {
		log.Printf("[WhatsApp Client Mock] Outbound message to %s:\n%s\n", to, message)
		return nil
	}

	payload := textMessagePayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
		Text: textBodyData{
			PreviewURL: true,
			Body:       message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", c.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending whatsapp message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("whatsapp api error: status code %d", resp.StatusCode)
	}

	return nil
}
