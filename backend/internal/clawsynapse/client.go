package clawsynapse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

type TrustPendingItem struct {
	RequestID    string `json:"requestId"`
	From         string `json:"from"`
	To           string `json:"to"`
	Direction    string `json:"direction"`
	Reason       string `json:"reason"`
	ReceivedAtMs int64  `json:"receivedAtMs"`
}

type trustPendingResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []TrustPendingItem `json:"items"`
	} `json:"data"`
	TS int64 `json:"ts"`
}

type trustActionRequest struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

type trustActionResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	TS      int64  `json:"ts"`
}

type transfersResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []map[string]any `json:"items"`
	} `json:"data"`
	TS int64 `json:"ts"`
}

type transferResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Transfer map[string]any `json:"transfer"`
	} `json:"data"`
	TS int64 `json:"ts"`
}

type HealthSelf struct {
	NodeID              string `json:"nodeId"`
	DID                 string `json:"did"`
	IdentityFingerprint string `json:"identityFingerprint"`
	TrustMode           string `json:"trustMode"`
}

type HealthAdapter struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

type HealthNATS struct {
	Name             string `json:"name"`
	ServerURL        string `json:"serverUrl"`
	Connected        bool   `json:"connected"`
	Status           string `json:"status"`
	ConnectedAt      int64  `json:"connectedAt"`
	LastDisconnectAt int64  `json:"lastDisconnectAt"`
	LastReconnectAt  int64  `json:"lastReconnectAt"`
	Disconnects      int64  `json:"disconnects"`
	Reconnects       int64  `json:"reconnects"`
	LastError        string `json:"lastError"`
	InMsgs           uint64 `json:"inMsgs"`
	OutMsgs          uint64 `json:"outMsgs"`
	InBytes          uint64 `json:"inBytes"`
	OutBytes         uint64 `json:"outBytes"`
}

type HealthData struct {
	Self       HealthSelf    `json:"self"`
	PeersCount int           `json:"peersCount"`
	Adapter    HealthAdapter `json:"adapter"`
	NATS       HealthNATS    `json:"nats"`
}

type healthResponse struct {
	OK      bool       `json:"ok"`
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Data    HealthData `json:"data"`
	TS      int64      `json:"ts"`
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

func (c *Client) GetTransfer(ctx context.Context, transferID string) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}
	transferID = strings.TrimSpace(transferID)
	if transferID == "" {
		return nil, fmt.Errorf("transfer id is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/transfer/"+url.PathEscape(transferID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get transfer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("get transfer returned status 404")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get transfer returned status %d", resp.StatusCode)
	}

	var out transferResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode transfer response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("get transfer rejected: %s", out.Code)
	}
	if out.Data.Transfer == nil {
		return map[string]any{}, nil
	}
	return out.Data.Transfer, nil
}

func (c *Client) GetHealth(ctx context.Context) (*HealthData, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get health returned status %d", resp.StatusCode)
	}

	var out healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("get health rejected: %s", out.Code)
	}

	return &out.Data, nil
}

func (c *Client) GetSelfNodeID(ctx context.Context) (string, error) {
	health, err := c.GetHealth(ctx)
	if err != nil {
		return "", err
	}

	nodeID := strings.TrimSpace(health.Self.NodeID)
	if nodeID == "" {
		return "", fmt.Errorf("get health missing data.self.nodeId")
	}

	return nodeID, nil
}

func (c *Client) ListTransfers(ctx context.Context) ([]map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/transfers", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list transfers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list transfers returned status %d", resp.StatusCode)
	}

	var out transfersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode transfers response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("list transfers rejected: %s", out.Code)
	}
	if out.Data.Items == nil {
		return []map[string]any{}, nil
	}
	return out.Data.Items, nil
}

func (c *Client) GetPendingTrustRequests(ctx context.Context) ([]TrustPendingItem, error) {
	if c == nil {
		return nil, fmt.Errorf("clawsynapse client is disabled")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/trust/pending", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get trust pending request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get trust pending returned status %d", resp.StatusCode)
	}

	var out trustPendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode trust pending response: %w", err)
	}
	if !out.OK {
		return nil, fmt.Errorf("get trust pending rejected: %s", out.Code)
	}

	// Only return incoming requests
	incoming := make([]TrustPendingItem, 0, len(out.Data.Items))
	for _, item := range out.Data.Items {
		if item.Direction == "inbound" {
			incoming = append(incoming, item)
		}
	}
	return incoming, nil
}

func (c *Client) AuthChallenge(ctx context.Context, targetNode string) error {
	if c == nil {
		return fmt.Errorf("clawsynapse client is disabled")
	}
	body, err := json.Marshal(map[string]string{"targetNode": targetNode})
	if err != nil {
		return fmt.Errorf("marshal auth challenge request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/challenge", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth challenge request failed: %w", err)
	}
	defer resp.Body.Close()

	var out trustActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode auth challenge response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("auth challenge failed: %s — %s", out.Code, out.Message)
	}
	return nil
}

func (c *Client) ApproveTrustRequest(ctx context.Context, requestID, reason string) error {
	return c.trustAction(ctx, "/v1/trust/approve", requestID, reason)
}

func (c *Client) RejectTrustRequest(ctx context.Context, requestID, reason string) error {
	return c.trustAction(ctx, "/v1/trust/reject", requestID, reason)
}

func (c *Client) RevokeTrust(ctx context.Context, targetNode, reason string) error {
	if c == nil {
		return fmt.Errorf("clawsynapse client is disabled")
	}
	body, err := json.Marshal(map[string]string{"targetNode": targetNode, "reason": reason})
	if err != nil {
		return fmt.Errorf("marshal revoke trust request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/trust/revoke", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoke trust request failed: %w", err)
	}
	defer resp.Body.Close()

	var out trustActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode revoke trust response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("revoke trust failed: %s — %s", out.Code, out.Message)
	}
	return nil
}

func (c *Client) trustAction(ctx context.Context, path, requestID, reason string) error {
	if c == nil {
		return fmt.Errorf("clawsynapse client is disabled")
	}
	body, err := json.Marshal(trustActionRequest{RequestID: requestID, Reason: reason})
	if err != nil {
		return fmt.Errorf("marshal trust action request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("trust action request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("trust action returned status %d", resp.StatusCode)
	}

	var out trustActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode trust action response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("trust action rejected: %s", out.Code)
	}
	return nil
}
