package ntfy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL  string
	topic    string
	username string
	password string
	http     *http.Client
}

// New returns nil if baseURL or topic is unset, so callers can no-op gracefully
// rather than treating ntfy as a hard dependency.
func New(baseURL, topic, username, password string) *Client {
	if baseURL == "" || topic == "" {
		return nil
	}
	return &Client{baseURL: baseURL, topic: topic, username: username, password: password, http: &http.Client{}}
}

func (c *Client) Publish(title, message string) error {
	body, err := json.Marshal(map[string]string{
		"topic":   c.topic,
		"title":   title,
		"message": message,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}
