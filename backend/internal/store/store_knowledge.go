package store

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

// CreateKnowledgeDocument creates a new knowledge document record.
func (s *Store) CreateKnowledgeDocument(userID string, doc *model.KnowledgeDocument) (*model.KnowledgeDocument, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc.ID = "kb_" + newID()
	doc.UserID = userID
	doc.Status = model.KnowledgeDocStatusProcessing
	now := time.Now().UTC()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}

	s.knowledgeDocs[doc.ID] = doc
	s.userKnowledgeDocs[userID] = append(s.userKnowledgeDocs[userID], doc.ID)

	if err := s.persistKnowledgeDocUnsafe(doc); err != nil {
		return nil, mongoWriteError(err)
	}
	return doc, nil
}

// GetKnowledgeDocument returns a document by ID, checking ownership.
func (s *Store) GetKnowledgeDocument(userID, docID string) (*model.KnowledgeDocument, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.knowledgeDocs[docID]
	if !ok {
		return nil, transport.NotFound("knowledge document not found")
	}
	if doc.UserID != userID {
		return nil, transport.Forbidden("access denied")
	}
	return doc, nil
}

// ListKnowledgeDocuments returns documents for a user with optional filters.
func (s *Store) ListKnowledgeDocuments(userID, projectID, status, tag string) []*model.KnowledgeDocument {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docIDs := s.userKnowledgeDocs[userID]
	var result []*model.KnowledgeDocument
	for _, id := range docIDs {
		doc, ok := s.knowledgeDocs[id]
		if !ok {
			continue
		}
		if projectID != "" {
			if doc.ProjectID == nil || *doc.ProjectID != projectID {
				continue
			}
		}
		if status != "" && doc.Status != status {
			continue
		}
		if tag != "" && !containsTag(doc.Tags, tag) {
			continue
		}
		result = append(result, doc)
	}
	return result
}

// UpdateKnowledgeDocument updates document metadata.
func (s *Store) UpdateKnowledgeDocument(userID, docID string, title, description *string, tags []string) (*model.KnowledgeDocument, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.knowledgeDocs[docID]
	if !ok {
		return nil, transport.NotFound("knowledge document not found")
	}
	if doc.UserID != userID {
		return nil, transport.Forbidden("access denied")
	}

	if title != nil {
		doc.Title = *title
	}
	if description != nil {
		doc.Description = *description
	}
	if tags != nil {
		doc.Tags = tags
	}
	doc.UpdatedAt = time.Now().UTC()

	if err := s.persistKnowledgeDocUnsafe(doc); err != nil {
		return nil, mongoWriteError(err)
	}
	return doc, nil
}

// DeleteKnowledgeDocument removes a document and its chunks from the store.
func (s *Store) DeleteKnowledgeDocument(userID, docID string) (*model.KnowledgeDocument, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.knowledgeDocs[docID]
	if !ok {
		return nil, transport.NotFound("knowledge document not found")
	}
	if doc.UserID != userID {
		return nil, transport.Forbidden("access denied")
	}

	delete(s.knowledgeDocs, docID)
	ids := s.userKnowledgeDocs[userID]
	for i, id := range ids {
		if id == docID {
			s.userKnowledgeDocs[userID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	_ = s.deleteKnowledgeDocUnsafe(docID)
	_ = s.deleteKnowledgeChunksUnsafe(docID)
	return doc, nil
}

// SaveKnowledgeChunks saves chunks to MongoDB (called by processor).
func (s *Store) SaveKnowledgeChunks(docID string, chunks []model.KnowledgeChunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = s.deleteKnowledgeChunksUnsafe(docID)
	return s.persistKnowledgeChunksUnsafe(chunks)
}

// SetKnowledgeDocSourceURI sets the source URI and persists to MongoDB.
func (s *Store) SetKnowledgeDocSourceURI(docID, uri string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.knowledgeDocs[docID]
	if !ok {
		return transport.NotFound("knowledge document not found")
	}
	doc.SourceURI = uri
	if err := s.persistKnowledgeDocUnsafe(doc); err != nil {
		return mongoWriteError(err)
	}
	return nil
}

// UpdateKnowledgeDocStatus updates document status and chunk count.
func (s *Store) UpdateKnowledgeDocStatus(docID, status string, chunkCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.knowledgeDocs[docID]
	if !ok {
		return nil
	}
	doc.Status = status
	doc.ChunkCount = chunkCount
	doc.UpdatedAt = time.Now().UTC()
	return s.persistKnowledgeDocUnsafe(doc)
}

// GetKnowledgeChunksByIDs fetches chunks from MongoDB by their IDs.
func (s *Store) GetKnowledgeChunksByIDs(chunkIDs []string) ([]model.KnowledgeChunk, error) {
	if !s.mongoEnabled || s.mongoKnowledgeChunks == nil || len(chunkIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()

	cursor, err := s.mongoKnowledgeChunks.Find(ctx, bson.M{"_id": bson.M{"$in": chunkIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chunks []model.KnowledgeChunk
	if err := cursor.All(ctx, &chunks); err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetKnowledgeChunksByDocID returns all chunks for a document.
func (s *Store) GetKnowledgeChunksByDocID(docID string) ([]model.KnowledgeChunk, error) {
	if !s.mongoEnabled || s.mongoKnowledgeChunks == nil {
		return nil, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()

	cursor, err := s.mongoKnowledgeChunks.Find(ctx, bson.M{"document_id": docID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chunks []model.KnowledgeChunk
	if err := cursor.All(ctx, &chunks); err != nil {
		return nil, err
	}
	return chunks, nil
}

// ResolveKnowledgeDocOwnerByAgentNode finds the user who owns an agent by node ID.
func (s *Store) ResolveKnowledgeDocOwnerByAgentNode(nodeID string) (string, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentID, ok := s.agentByNode[strings.TrimSpace(nodeID)]
	if !ok {
		return "", transport.NotFound("agent not found for node")
	}
	agent, ok := s.agents[agentID]
	if !ok {
		return "", transport.NotFound("agent not found")
	}
	return agent.UserID, nil
}

// ValidateProjectOwnership checks that a project belongs to a user.
func (s *Store) ValidateProjectOwnership(userID, projectID string) *transport.AppError {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return transport.NotFound("project not found")
	}
	if project.UserID != userID {
		return transport.Forbidden("access denied to project")
	}
	return nil
}

// GetKnowledgeDocTitle returns the title of a knowledge document.
func (s *Store) GetKnowledgeDocTitle(docID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if doc, ok := s.knowledgeDocs[docID]; ok {
		return doc.Title
	}
	return ""
}

// SearchKnowledgeChunks does a text search fallback (when no vector search).
func (s *Store) SearchKnowledgeChunks(ctx context.Context, userID string, projectID *string, query string, limit int) ([]model.KnowledgeChunk, error) {
	if !s.mongoEnabled || s.mongoKnowledgeChunks == nil {
		return nil, nil
	}
	mctx, cancel := s.mongoContext()
	defer cancel()
	_ = ctx

	filter := bson.M{"user_id": userID}
	if projectID != nil {
		filter["$or"] = bson.A{
			bson.M{"project_id": *projectID},
			bson.M{"project_id": nil},
		}
	}

	cursor, err := s.mongoKnowledgeChunks.Find(mctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(mctx)

	var chunks []model.KnowledgeChunk
	if err := cursor.All(mctx, &chunks); err != nil {
		return nil, err
	}

	// Simple keyword matching fallback
	queryLower := strings.ToLower(query)
	var matched []model.KnowledgeChunk
	for _, chunk := range chunks {
		if strings.Contains(strings.ToLower(chunk.Content), queryLower) {
			matched = append(matched, chunk)
			if len(matched) >= limit {
				break
			}
		}
	}
	return matched, nil
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}
