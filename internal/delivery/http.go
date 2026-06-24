package delivery

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

type DeliveryResult struct {
	StatusCode      int
	ResponseBody    string
	ResponseHeaders map[string][]string
	Duration        time.Duration
	Error           error
}

type WebhookClient struct {
	client *http.Client
}

func NewWebhookClient() *WebhookClient {
	return &WebhookClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *WebhookClient) Deliver(ctx context.Context, url string, payload []byte, headers map[string]string) DeliveryResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return DeliveryResult{Error: err, Duration: time.Since(start)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WebhookPlatform/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Error: err, Duration: time.Since(start)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	return DeliveryResult{
		StatusCode:      resp.StatusCode,
		ResponseBody:    string(body),
		ResponseHeaders: resp.Header,
		Duration:        time.Since(start),
	}
}
