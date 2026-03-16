package nats

import (
	"errors"
	"fmt"
	"strings"
)

type Subject struct {
	Namespace string
	NodeID    string
	Domain    string
	Action    string
}

func ParseSubject(value string) (Subject, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 4 {
		return Subject{}, fmt.Errorf("invalid subject %q", value)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return Subject{}, errors.New("subject has empty segment")
	}
	return Subject{
		Namespace: parts[0],
		NodeID:    parts[1],
		Domain:    parts[2],
		Action:    parts[3],
	}, nil
}
