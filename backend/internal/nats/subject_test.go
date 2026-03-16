package nats

import "testing"

func TestParseSubject(t *testing.T) {
	s, err := ParseSubject("agent.node-1.todo.progress")
	if err != nil {
		t.Fatalf("parse subject: %v", err)
	}
	if s.Namespace != "agent" || s.NodeID != "node-1" || s.Domain != "todo" || s.Action != "progress" {
		t.Fatalf("unexpected parse result: %+v", s)
	}
}

func TestParseSubjectInvalid(t *testing.T) {
	if _, err := ParseSubject("agent.node-1.todo"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestDecodeEnvelopeRequiresID(t *testing.T) {
	_, err := decodeEnvelope([]byte(`{"node_id":"node-1","timestamp":"2026-03-16T10:30:00Z","payload":{}}`), "node-1")
	if err == nil {
		t.Fatal("expected decodeEnvelope to reject missing id")
	}
}
