package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client reports server state back to the platform control-plane.
type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func New(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{},
	}
}

// ReportStatus tells the platform the current status of serverID (e.g. "rolled_back", "succeeded").
func (c *Client) ReportStatus(ctx context.Context, serverID, status string) error {
	body, _ := json.Marshal(map[string]string{"status": status})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/internal/deployments/%s/status", c.baseURL, serverID),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("report status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("report status: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ReportAddress tells the platform the host:port address for serverID.
func (c *Client) ReportAddress(ctx context.Context, serverID, address string) error {
	body, _ := json.Marshal(map[string]string{"address": address})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/internal/deployments/%s/address", c.baseURL, serverID),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("report address: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("report address: unexpected status %d", resp.StatusCode)
	}
	return nil
}
