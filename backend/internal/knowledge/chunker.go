package knowledge

import (
	"strings"
	"unicode/utf8"
)

const (
	DefaultChunkSize    = 512 // target tokens per chunk
	DefaultChunkOverlap = 64  // overlap tokens between chunks
	ApproxCharsPerToken = 4   // rough estimate for English/Chinese mixed text
)

// ChunkResult holds one chunk and its metadata.
type ChunkResult struct {
	Content    string
	TokenCount int
	Metadata   map[string]any
}

type markdownSection struct {
	Heading string
	Content string
}

// ChunkText splits text into overlapping chunks.
// For markdown, it tries to split on headings first.
func ChunkText(text string, mimeType string, chunkSize, overlap int) []ChunkResult {
	chunkSize, overlap = normalizeChunkParams(chunkSize, overlap)

	if isMarkdown(text, mimeType) {
		chunks := chunkMarkdown(text, chunkSize, overlap)
		if len(chunks) > 0 {
			return chunks
		}
	}

	return chunkPlainText(text, chunkSize, overlap)
}

func normalizeChunkParams(chunkSize, overlap int) (int, int) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}
	if overlap >= chunkSize {
		if chunkSize <= 1 {
			overlap = 0
		} else {
			overlap = chunkSize - 1
		}
	}
	return chunkSize, overlap
}

func isMarkdown(text, mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if strings.Contains(mimeType, "markdown") || strings.HasSuffix(mimeType, ".md") {
		return true
	}
	return looksLikeMarkdown(text)
}

func looksLikeMarkdown(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) > 32 {
		lines = lines[:32]
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isMarkdownHeading(trimmed) {
			return true
		}
		if _, ok := markdownFenceMarker(trimmed); ok {
			return true
		}
	}
	return false
}

func isMarkdownHeading(line string) bool {
	if line == "" || line[0] != '#' {
		return false
	}
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return false
	}
	return level == len(line) || line[level] == ' ' || line[level] == '\t'
}

func extractMarkdownHeading(line string) string {
	line = strings.TrimSpace(line)
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	return strings.TrimSpace(line[level:])
}

func markdownFenceMarker(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "```"):
		return "```", true
	case strings.HasPrefix(line, "~~~"):
		return "~~~", true
	default:
		return "", false
	}
}

// chunkMarkdown splits on markdown headings, then sub-chunks large sections.
func chunkMarkdown(text string, chunkSize, overlap int) []ChunkResult {
	lines := strings.Split(text, "\n")
	var sections []markdownSection
	var current strings.Builder
	currentHeading := ""
	inFence := false
	fenceMarker := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if marker, ok := markdownFenceMarker(trimmed); ok {
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			}
		}
		if !inFence && isMarkdownHeading(trimmed) && current.Len() > 0 {
			sections = append(sections, markdownSection{
				Heading: currentHeading,
				Content: current.String(),
			})
			current.Reset()
			currentHeading = extractMarkdownHeading(trimmed)
		} else if !inFence && isMarkdownHeading(trimmed) && current.Len() == 0 {
			currentHeading = extractMarkdownHeading(trimmed)
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, markdownSection{
			Heading: currentHeading,
			Content: current.String(),
		})
	}

	if len(sections) <= 1 {
		return nil // fall back to plain text
	}

	var results []ChunkResult
	for _, section := range sections {
		content := strings.TrimSpace(section.Content)
		if content == "" {
			continue
		}
		metadata := map[string]any{}
		if section.Heading != "" {
			metadata["heading"] = section.Heading
		}
		tokenCount := estimateTokens(content)
		if tokenCount <= chunkSize {
			results = append(results, ChunkResult{
				Content:    content,
				TokenCount: tokenCount,
				Metadata:   metadata,
			})
		} else {
			sub := chunkPlainText(content, chunkSize, overlap)
			for i := range sub {
				if len(metadata) == 0 {
					continue
				}
				if sub[i].Metadata == nil {
					sub[i].Metadata = map[string]any{}
				}
				for key, value := range metadata {
					sub[i].Metadata[key] = value
				}
			}
			results = append(results, sub...)
		}
	}
	return results
}

// chunkPlainText splits text by approximate token count with overlap.
func chunkPlainText(text string, chunkSize, overlap int) []ChunkResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	charChunkSize := chunkSize * ApproxCharsPerToken
	charOverlap := overlap * ApproxCharsPerToken

	runes := []rune(text)
	totalChars := len(runes)

	if totalChars <= charChunkSize {
		return []ChunkResult{{
			Content:    text,
			TokenCount: estimateTokens(text),
			Metadata:   map[string]any{},
		}}
	}

	var results []ChunkResult
	start := 0
	for start < totalChars {
		end := start + charChunkSize
		if end > totalChars {
			end = totalChars
		}

		// Try to break at paragraph or sentence boundary
		if end < totalChars {
			end = findBreakPoint(runes, start, end)
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			results = append(results, ChunkResult{
				Content:    chunk,
				TokenCount: estimateTokens(chunk),
				Metadata:   map[string]any{},
			})
		}

		nextStart := end - charOverlap
		if nextStart <= start {
			nextStart = end
		}
		if nextStart < 0 {
			nextStart = 0
		}
		start = nextStart
	}
	return results
}

// findBreakPoint looks for a good break point (paragraph, sentence, or word boundary).
func findBreakPoint(runes []rune, start, end int) int {
	// Search backwards for paragraph break
	for i := end; i > start+len(runes[start:end])/2; i-- {
		if i < len(runes) && runes[i] == '\n' && i+1 < len(runes) && runes[i+1] == '\n' {
			return i + 1
		}
	}
	// Search backwards for sentence end
	for i := end; i > start+len(runes[start:end])/2; i-- {
		if i < len(runes) {
			ch := runes[i]
			if ch == '.' || ch == '。' || ch == '!' || ch == '？' || ch == '?' {
				return i + 1
			}
		}
	}
	// Search backwards for word boundary (space)
	for i := end; i > start+len(runes[start:end])*3/4; i-- {
		if i < len(runes) && runes[i] == ' ' {
			return i + 1
		}
	}
	return end
}

func estimateTokens(text string) int {
	charCount := utf8.RuneCountInString(text)
	tokens := charCount / ApproxCharsPerToken
	if tokens == 0 && charCount > 0 {
		tokens = 1
	}
	return tokens
}
