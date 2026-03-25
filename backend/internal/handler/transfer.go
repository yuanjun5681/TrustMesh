package handler

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type TransferHandler struct {
	store  *store.Store
	client *clawsynapse.Client
}

func NewTransferHandler(s *store.Store, client *clawsynapse.Client) *TransferHandler {
	return &TransferHandler{store: s, client: client}
}

func (h *TransferHandler) GetTaskArtifactTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if h.client == nil {
		transport.WriteError(c, transport.NewError(http.StatusServiceUnavailable, "CLAWSYNAPSE_DISABLED", "clawsynapse client is disabled"))
		return
	}

	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	artifact := findTaskArtifact(task, c.Param("artifactId"))
	if artifact == nil {
		transport.WriteError(c, transport.NotFound("artifact not found"))
		return
	}

	transferID := transferIDFromArtifact(artifact)
	if transferID == "" {
		transport.WriteError(c, transport.Validation("artifact is not backed by transfer", map[string]any{"artifact_id": artifact.ID}))
		return
	}

	transfer, err := h.client.GetTransfer(context.Background(), transferID)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			transport.WriteError(c, transport.NotFound("transfer not found"))
			return
		}
		transport.WriteError(c, transport.NewError(http.StatusBadGateway, "TRANSFER_LOOKUP_FAILED", "failed to fetch transfer details"))
		return
	}

	transport.WriteData(c, http.StatusOK, transfer)
}

func (h *TransferHandler) GetTaskArtifactContent(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	artifact := findTaskArtifact(task, c.Param("artifactId"))
	if artifact == nil {
		transport.WriteError(c, transport.NotFound("artifact not found"))
		return
	}

	localPath, fileName, mimeType, appErr := h.resolveArtifactFile(artifact)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	file, err := os.Open(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			transport.WriteError(c, transport.NotFound("artifact file not found"))
			return
		}
		transport.WriteError(c, transport.NewError(http.StatusBadGateway, "ARTIFACT_READ_FAILED", "failed to read artifact file"))
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		transport.WriteError(c, transport.NewError(http.StatusBadGateway, "ARTIFACT_READ_FAILED", "failed to stat artifact file"))
		return
	}

	if mimeType == "" {
		mimeType = detectContentType(file)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			transport.WriteError(c, transport.NewError(http.StatusBadGateway, "ARTIFACT_READ_FAILED", "failed to rewind artifact file"))
			return
		}
	}
	mimeType = normalizeTextContentType(mimeType)
	if mimeType != "" {
		c.Header("Content-Type", mimeType)
	}
	c.Header("Content-Disposition", contentDisposition("inline", fileName))
	http.ServeContent(c.Writer, c.Request, fileName, fileModTime(info), file)
}

func findTaskArtifact(task *model.TaskDetail, artifactID string) *model.TaskArtifact {
	for i := range task.Artifacts {
		if task.Artifacts[i].ID == artifactID {
			return &task.Artifacts[i]
		}
	}
	return nil
}

func transferIDFromArtifact(artifact *model.TaskArtifact) string {
	if artifact == nil {
		return ""
	}
	if transferID, ok := artifact.Metadata["transfer_id"].(string); ok && strings.TrimSpace(transferID) != "" {
		return strings.TrimSpace(transferID)
	}
	if transfer, ok := artifact.Metadata["transfer"].(map[string]any); ok {
		if transferID, ok := transfer["transfer_id"].(string); ok && strings.TrimSpace(transferID) != "" {
			return strings.TrimSpace(transferID)
		}
		if transferID, ok := transfer["transferId"].(string); ok && strings.TrimSpace(transferID) != "" {
			return strings.TrimSpace(transferID)
		}
	}
	if strings.HasPrefix(artifact.URI, "transfer://") {
		return strings.TrimSpace(strings.TrimPrefix(artifact.URI, "transfer://"))
	}
	return ""
}

func (h *TransferHandler) resolveArtifactFile(artifact *model.TaskArtifact) (string, string, string, *transport.AppError) {
	localPath := stringMetadataValue(artifact.Metadata, "local_path")
	fileName := stringMetadataValue(artifact.Metadata, "file_name")
	mimeType := ""
	if artifact.MimeType != nil {
		mimeType = strings.TrimSpace(*artifact.MimeType)
	}
	if localPath == "" || fileName == "" || mimeType == "" {
		transfer, err := h.lookupTransferDetail(artifact)
		if err == nil {
			if localPath == "" {
				localPath = firstTransferString(transfer, "localPath", "local_path")
			}
			if fileName == "" {
				fileName = firstTransferString(transfer, "fileName", "file_name")
			}
			if mimeType == "" {
				mimeType = firstTransferString(transfer, "mimeType", "mime_type")
			}
		}
	}

	if localPath == "" {
		return "", "", "", transport.NotFound("artifact file path unavailable")
	}
	if fileName == "" {
		fileName = filepath.Base(localPath)
	}
	return localPath, fileName, mimeType, nil
}

func (h *TransferHandler) lookupTransferDetail(artifact *model.TaskArtifact) (map[string]any, error) {
	if h == nil || h.client == nil {
		return nil, http.ErrServerClosed
	}
	transferID := transferIDFromArtifact(artifact)
	if transferID == "" {
		return nil, http.ErrMissingFile
	}
	return h.client.GetTransfer(context.Background(), transferID)
}

func firstTransferString(transfer map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := transfer[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func stringMetadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	transfer, ok := metadata["transfer"].(map[string]any)
	if !ok {
		return ""
	}
	if value, ok := transfer[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	switch key {
	case "local_path":
		if value, ok := transfer["localPath"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	case "file_name":
		if value, ok := transfer["fileName"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func detectContentType(file *os.File) string {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}
	if n == 0 {
		return ""
	}
	return http.DetectContentType(buf[:n])
}

func normalizeTextContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		if strings.HasPrefix(strings.ToLower(contentType), "text/") && !strings.Contains(strings.ToLower(contentType), "charset=") {
			return contentType + "; charset=utf-8"
		}
		return contentType
	}
	if !strings.HasPrefix(strings.ToLower(mediaType), "text/") {
		return contentType
	}
	if _, ok := params["charset"]; ok {
		return contentType
	}
	params["charset"] = "utf-8"
	return mime.FormatMediaType(mediaType, params)
}

func fileModTime(info os.FileInfo) time.Time {
	if info == nil {
		return time.Time{}
	}
	return info.ModTime()
}

func contentDisposition(mode, fileName string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "inline"
	}
	if fileName == "" {
		return mode
	}
	return mode + `; filename="` + strings.ReplaceAll(fileName, `"`, "") + `"`
}
