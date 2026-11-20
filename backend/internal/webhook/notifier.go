// Package webhook delivers task events to an external agent endpoint. Requests
// are signed the way Hermes' generic webhook route expects: an HMAC over
// "<unix timestamp>.<body>" in X-Webhook-Signature-V2.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	queueSize      = 256
	requestTimeout = 10 * time.Second
	// attempts are spaced 2s, then 10s apart.
	maxAttempts = 3
)

// Task is the task snapshot carried by an event.
type Task struct {
	ID        int64   `json:"id"`
	Ref       string  `json:"ref"`
	Title     string  `json:"title"`
	Kind      string  `json:"kind"`
	Priority  string  `json:"priority"`
	DueDate   *string `json:"due_date,omitempty"`
	Completed bool    `json:"completed"`
	Board     string  `json:"board"`
	Column    string  `json:"column"`
	Workspace string  `json:"workspace"`
}

// Event is one delivered payload. EventType repeats Event so a receiver that
// filters on event_type (Hermes does) sees the same value.
type Event struct {
	Event     string    `json:"event"`
	EventType string    `json:"event_type"`
	ID        string    `json:"id"`
	At        time.Time `json:"at"`
	Actor     string    `json:"actor,omitempty"`
	Task      Task      `json:"task"`
}

// Notifier queues events and delivers them in order on one worker.
type Notifier struct {
	url    string
	secret string
	queue  chan Event
	client *http.Client
}

// New returns nil when no endpoint is configured, or when the endpoint has no
// secret, since an unsigned endpoint would let anyone drive the agent.
func New(url, secret string) *Notifier {
	if url == "" {
		return nil
	}
	if secret == "" {
		log.Println("FAI_WEBHOOK_URL set without FAI_WEBHOOK_SECRET: webhook delivery disabled")
		return nil
	}
	return &Notifier{
		url:    url,
		secret: secret,
		queue:  make(chan Event, queueSize),
		client: &http.Client{Timeout: requestTimeout},
	}
}

// Publish queues an event. A full queue drops the event and logs it, so a slow
// or dead endpoint never stalls a user request.
func (n *Notifier) Publish(event Event) {
	if n == nil {
		return
	}
	if event.ID == "" {
		event.ID = newEventID()
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	event.EventType = event.Event
	select {
	case n.queue <- event:
	default:
		log.Printf("webhook: queue full, dropped %s for %s", event.Event, event.Task.Ref)
	}
}

// Run delivers queued events until ctx is cancelled.
func (n *Notifier) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-n.queue:
			if err := n.deliver(ctx, event); err != nil {
				log.Printf("webhook: %s for %s failed: %v", event.Event, event.Task.Ref, err)
			}
		}
	}
}

func (n *Notifier) deliver(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	delay := 2 * time.Second
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if lastErr = n.post(ctx, event.ID, body); lastErr == nil {
			return nil
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 5
		}
	}
	return lastErr
}

func (n *Notifier) post(ctx context.Context, id string, body []byte) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fai-webhook")
	req.Header.Set("X-Request-ID", id)
	req.Header.Set("X-Webhook-Timestamp", timestamp)
	req.Header.Set("X-Webhook-Signature-V2", sign(n.secret, timestamp, body))

	res, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		io.Copy(io.Discard, res.Body)
		return nil
	}
	snippet, _ := io.ReadAll(io.LimitReader(res.Body, 200))
	return fmt.Errorf("endpoint returned %d: %s", res.StatusCode, bytes.TrimSpace(snippet))
}

func sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newEventID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf[:])
}
