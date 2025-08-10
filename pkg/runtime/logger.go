package runtime

import (
	"context"
	"net/http"
)

type LogFields struct {
	Headers http.Header
	Body    []byte
	Extras  map[string]any
}

type LogEntry struct {
	Message string
	Prefix  string
	Data    *LogFields
}

// Logger is a function type that defines the signature for logging HTTP requests and responses.
type Logger func(ctx context.Context, entry LogEntry)
