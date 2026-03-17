package clawsynapse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type PublishResult struct {
	TargetNode string `json:"targetNode"`
	MessageID  string `json:"messageId"`
}

type Peer struct {
	NodeID       string         `json:"nodeId"`
	AgentProduct string         `json:"agentProduct"`
	Version      string         `json:"version"`
	Capabilities []string       `json:"capabilities"`
	Inbox        string         `json:"inbox"`
	AuthStatus   string         `json:"authStatus"`
	TrustStatus  string         `json:"trustStatus"`
	LastSeenMs   int64          `json:"lastSeenMs"`
	Metadata     map[string]any `json:"metadata"`
}

type publishRequest struct {
	TargetNode string         `json:"targetNode"`
	Type       string         `json:"type"`
	Message    string         `json:"message"`
	SessionKey string         `json:"sessionKey,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type publishResponse struct {
	OK      bool          `json:"ok"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Data    PublishResult `json:"data"`
	TS      int64         `json:"ts"`
}

type peersResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []Peer `json:"items"`
	} `json:"data"`
	TS int64 `json:"ts"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Publish(ctx context.Context, targetNode, msgType string, payload any, sessionKey string, metadata map[string]any) (*PublishResult, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}
	messageBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal publish payload: %w", err)
	}

	reqBody, err := json.Marshal(publishRequest{
		TargetNode: strings.TrimSpace(targetNode),
		Type:       strings.TrimSpace(msgType),
		Message:    string(messageBytes),
		SessionKey: strings.TrimSpace(sessionKey),
		Metadata:   metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal publish request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/publish", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("publish request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("publish request returned status %d", resp.StatusCode)
	}

	var out publishResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode publish response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("publish rejected: %s", out.Code)
	}
	return &out.Data, nil
}

func (c *Client) GetPeers(ctx context.Context) ([]Peer, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/peers", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get peers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get peers returned status %d", resp.StatusCode)
	}

	var out peersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode peers response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("get peers rejected: %s", out.Code)
	}
	return out.Data.Items, nil
}
