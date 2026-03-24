package knowledge

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStorage abstracts file storage for knowledge documents.
type FileStorage interface {
	Save(ctx context.Context, userID, docID, filename string, r io.Reader) (uri string, err error)
	Get(ctx context.Context, uri string) (io.ReadCloser, error)
	Delete(ctx context.Context, uri string) error
}

// LocalFileStorage stores files on the local filesystem.
type LocalFileStorage struct {
	basePath string
}

func NewLocalFileStorage(basePath string) *LocalFileStorage {
	return &LocalFileStorage{basePath: basePath}
}

func (s *LocalFileStorage) Save(_ context.Context, userID, docID, filename string, r io.Reader) (string, error) {
	dir := filepath.Join(s.basePath, userID, docID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}

func (s *LocalFileStorage) Get(_ context.Context, uri string) (io.ReadCloser, error) {
	f, err := os.Open(uri)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *LocalFileStorage) Delete(_ context.Context, uri string) error {
	dir := filepath.Dir(uri)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove directory: %w", err)
	}
	return nil
}
