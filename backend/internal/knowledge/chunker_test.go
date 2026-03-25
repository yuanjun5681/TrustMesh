package knowledge

import (
	"fmt"
	"strings"
	"testing"
)

func TestChunkText_NoOverlapStillConsumesWholeText(t *testing.T) {
	text := buildTokenText(40)

	chunks := ChunkText(text, "text/plain", 5, 0)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[len(chunks)-1].Content, "token-39") {
		t.Fatalf("expected final chunk to reach end of text, got %q", chunks[len(chunks)-1].Content)
	}
}

func TestChunkText_OverlapLargerThanChunkSizeStillMakesProgress(t *testing.T) {
	text := buildTokenText(40)

	chunks := ChunkText(text, "text/plain", 5, 9)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[len(chunks)-1].Content, "token-39") {
		t.Fatalf("expected final chunk to reach end of text, got %q", chunks[len(chunks)-1].Content)
	}
}

func TestChunkText_MarkdownIgnoresHashesInsideFencedCode(t *testing.T) {
	text := strings.Join([]string{
		"# Intro",
		"Overview paragraph.",
		"```c",
		"#include <stdio.h>",
		"# not-a-heading",
		"```",
		"## Actual Section",
		"More content.",
	}, "\n")

	chunks := ChunkText(text, "text/markdown", 512, 64)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0].Content, "#include <stdio.h>") {
		t.Fatalf("expected fenced code to stay in first chunk, got %q", chunks[0].Content)
	}
	if got := chunks[0].Metadata["heading"]; got != "Intro" {
		t.Fatalf("expected first chunk heading metadata, got %#v", got)
	}
	if got := chunks[1].Metadata["heading"]; got != "Actual Section" {
		t.Fatalf("expected second chunk heading metadata, got %#v", got)
	}
}

func TestChunkText_DetectsMarkdownFromContentWhenMimeTypeIsPlainText(t *testing.T) {
	text := strings.Join([]string{
		"# Intro",
		"Overview paragraph.",
		"## Details",
		"More content.",
	}, "\n")

	chunks := ChunkText(text, "text/plain", 512, 64)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 markdown chunks, got %d", len(chunks))
	}
	if got := chunks[0].Metadata["heading"]; got != "Intro" {
		t.Fatalf("expected first chunk heading metadata, got %#v", got)
	}
	if got := chunks[1].Metadata["heading"]; got != "Details" {
		t.Fatalf("expected second chunk heading metadata, got %#v", got)
	}
}

func buildTokenText(n int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = fmt.Sprintf("token-%02d", i)
	}
	return strings.Join(parts, " ")
}
