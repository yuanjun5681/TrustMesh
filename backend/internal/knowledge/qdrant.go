package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QdrantClient wraps the Qdrant REST API for vector operations.
type QdrantClient struct {
	baseURL    string
	httpClient *http.Client
	collection string
	dimension  int
}

func NewQdrantClient(baseURL string, dimension int) *QdrantClient {
	return &QdrantClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		collection: "knowledge_chunks",
		dimension:  dimension,
	}
}

// EnsureCollection creates the collection if it does not exist.
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant check collection: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create collection
	body := map[string]any{
		"vectors": map[string]any{
			"size":     c.dimension,
			"distance": "Cosine",
		},
	}
	return c.put(ctx, fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection), body)
}

// UpsertPoints inserts or updates points in the collection.
func (c *QdrantClient) UpsertPoints(ctx context.Context, points []QdrantPoint) error {
	if len(points) == 0 {
		return nil
	}
	body := map[string]any{
		"points": points,
	}
	return c.put(ctx, fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection), body)
}

// Search performs a vector similarity search with optional filters.
func (c *QdrantClient) Search(ctx context.Context, vector []float32, filter *QdrantFilter, limit int) ([]QdrantSearchResult, error) {
	body := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	if filter != nil {
		body["filter"] = filter
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collection)
	respBody, err := c.postJSON(ctx, url, body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result []QdrantSearchResult `json:"result"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal search response: %w", err)
	}
	return resp.Result, nil
}

// DeleteByDocumentID deletes all points associated with a document.
func (c *QdrantClient) DeleteByDocumentID(ctx context.Context, documentID string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "document_id", "match": map[string]any{"value": documentID}},
			},
		},
	}
	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, c.collection)
	_, err := c.postJSON(ctx, url, body)
	return err
}

// QdrantPoint represents a point to upsert.
type QdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

// QdrantFilter represents a search filter.
type QdrantFilter struct {
	Must   []QdrantCondition `json:"must,omitempty"`
	Should []QdrantCondition `json:"should,omitempty"`
}

// QdrantCondition represents a filter condition.
type QdrantCondition struct {
	Key   string `json:"key"`
	Match any    `json:"match"`
}

// QdrantSearchResult represents a search hit.
type QdrantSearchResult struct {
	ID      string         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func (c *QdrantClient) put(ctx context.Context, url string, body any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant PUT %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	return nil
}

func (c *QdrantClient) postJSON(ctx context.Context, url string, body any) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qdrant POST %s: status %d: %s", url, resp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}
