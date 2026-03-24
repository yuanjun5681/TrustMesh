package knowledge

import (
	"strings"
	"unicode/utf8"
)

const (
	DefaultChunkSize    = 512  // target tokens per chunk
	DefaultChunkOverlap = 64   // overlap tokens between chunks
	ApproxCharsPerToken = 4    // rough estimate for English/Chinese mixed text
)

// ChunkResult holds one chunk and its metadata.
type ChunkResult struct {
	Content    string
	TokenCount int
	Metadata   map[string]any
}

// ChunkText splits text into overlapping chunks.
// For markdown, it tries to split on headings first.
func ChunkText(text string, mimeType string, chunkSize, overlap int) []ChunkResult {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}

	if isMarkdown(mimeType) {
		chunks := chunkMarkdown(text, chunkSize, overlap)
		if len(chunks) > 0 {
			return chunks
		}
	}

	return chunkPlainText(text, chunkSize, overlap)
}

func isMarkdown(mimeType string) bool {
	return strings.Contains(mimeType, "markdown") || strings.HasSuffix(mimeType, ".md")
}

// chunkMarkdown splits on markdown headings, then sub-chunks large sections.
func chunkMarkdown(text string, chunkSize, overlap int) []ChunkResult {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}

	if len(sections) <= 1 {
		return nil // fall back to plain text
	}

	var results []ChunkResult
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		tokenCount := estimateTokens(section)
		if tokenCount <= chunkSize {
			results = append(results, ChunkResult{
				Content:    section,
				TokenCount: tokenCount,
				Metadata:   map[string]any{},
			})
		} else {
			sub := chunkPlainText(section, chunkSize, overlap)
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

		start = end - charOverlap
		if start < 0 {
			start = 0
		}
		if start >= end {
			break
		}
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
