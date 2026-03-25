package assistant

// ChatRequest is the incoming request from the frontend.
type ChatRequest struct {
	Message string       `json:"message"`
	Context *ChatContext `json:"context,omitempty"`
	History []HistoryMsg `json:"history,omitempty"`
}

type ChatContext struct {
	CurrentPage string `json:"current_page,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
}

type HistoryMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SSE event types sent to the frontend.

type DeltaEvent struct {
	Content string `json:"content"`
}

type ToolCallEvent struct {
	Tool string `json:"tool"`
	Args any    `json:"args"`
}

type ToolResultEvent struct {
	Tool   string `json:"tool"`
	Result any    `json:"result"`
}

type NavigateEvent struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// SSEWriter abstracts writing SSE events to the response.
type SSEWriter interface {
	WriteEvent(event string, data any)
}
