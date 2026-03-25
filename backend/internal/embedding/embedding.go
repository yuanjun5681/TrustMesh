package embedding

import (
	"context"
	"fmt"

	"trustmesh/backend/internal/config"
)

// Client is the interface for text embedding services.
type Client interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// NewClient creates an embedding client based on the config provider.
func NewClient(cfg config.Config) (Client, error) {
	switch cfg.EmbeddingProvider {
	case "openai":
		if cfg.EmbeddingAPIKey == "" {
			return nil, fmt.Errorf("EMBEDDING_API_KEY is required for openai provider")
		}
		return NewOpenAIClient(cfg.EmbeddingAPIURL, cfg.EmbeddingAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDimension), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.EmbeddingProvider)
	}
}
