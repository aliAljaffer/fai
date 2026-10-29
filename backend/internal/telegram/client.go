package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.telegram.org"

type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

func New(token string) *Client {
	return NewWithBaseURL(token, defaultBaseURL)
}

func NewWithBaseURL(token, baseURL string) *Client {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{token: token, baseURL: baseURL, http: &http.Client{Timeout: 70 * time.Second}}
}

type apiEnvelope struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
}

func (c *Client) redact(value string) string {
	return strings.ReplaceAll(value, c.token, "<token>")
}

func (c *Client) call(ctx context.Context, method string, query url.Values, payload any, result any) error {
	endpoint := c.baseURL + "/bot" + c.token + "/" + method
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("telegram: %s: %s", method, c.redact(err.Error()))
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: %s: %s", method, c.redact(err.Error()))
	}
	defer resp.Body.Close()

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram: %s returned status %d", method, resp.StatusCode)
	}
	if !envelope.OK {
		description := strings.TrimSpace(envelope.Description)
		if description == "" {
			description = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return fmt.Errorf("telegram: %s: %s", method, c.redact(description))
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("telegram: %s: %s", method, c.redact(err.Error()))
	}
	return nil
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	query := url.Values{}
	query.Set("offset", strconv.FormatInt(offset, 10))
	query.Set("timeout", strconv.Itoa(timeoutSeconds))
	query.Set("allowed_updates", `["message","callback_query"]`)
	var updates []Update
	if err := c.call(ctx, "getUpdates", query, nil, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

type sendMessageRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, markup *InlineKeyboardMarkup) error {
	payload := sendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	}
	return c.call(ctx, "sendMessage", nil, payload, nil)
}

type answerCallbackRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackID, text string) error {
	payload := answerCallbackRequest{CallbackQueryID: callbackID, Text: text}
	return c.call(ctx, "answerCallbackQuery", nil, payload, nil)
}

type getFileRequest struct {
	FileID string `json:"file_id"`
}

func (c *Client) GetFile(ctx context.Context, fileID string) (File, error) {
	var file File
	if err := c.call(ctx, "getFile", nil, getFileRequest{FileID: fileID}, &file); err != nil {
		return File{}, err
	}
	return file, nil
}

func (c *Client) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	endpoint := c.baseURL + "/file/bot" + c.token + "/" + strings.TrimLeft(filePath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: download: %s", c.redact(err.Error()))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: download: %s", c.redact(err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("telegram: download returned status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 25<<20))
}
