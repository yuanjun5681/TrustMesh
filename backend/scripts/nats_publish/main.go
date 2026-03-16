package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

type envelope struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	NodeID    string          `json:"node_id"`
	Payload   json.RawMessage `json:"payload"`
}

func main() {
	var (
		url       = flag.String("url", "nats://127.0.0.1:4222", "NATS server URL")
		subject   = flag.String("subject", "", "Subject to publish")
		nodeID    = flag.String("node-id", "", "Envelope node_id")
		messageID = flag.String("message-id", "", "Envelope id")
		payload   = flag.String("payload", "{}", "JSON object payload")
	)
	flag.Parse()

	if *subject == "" || *nodeID == "" || *messageID == "" {
		fmt.Fprintln(os.Stderr, "subject, node-id, and message-id are required")
		os.Exit(1)
	}
	if !json.Valid([]byte(*payload)) {
		fmt.Fprintln(os.Stderr, "payload must be valid JSON")
		os.Exit(1)
	}

	body, err := json.Marshal(envelope{
		ID:        *messageID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		NodeID:    *nodeID,
		Payload:   json.RawMessage(*payload),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal envelope: %v\n", err)
		os.Exit(1)
	}

	nc, err := nats.Connect(*url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect nats: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()

	if err := nc.Publish(*subject, body); err != nil {
		fmt.Fprintf(os.Stderr, "publish nats: %v\n", err)
		os.Exit(1)
	}
	if err := nc.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush nats: %v\n", err)
		os.Exit(1)
	}
}
