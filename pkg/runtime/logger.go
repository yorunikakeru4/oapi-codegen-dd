package runtime

import (
	"context"
	"net/http"
)

// Logger is a function type that defines the signature for logging HTTP requests and responses.
type Logger func(ctx context.Context, msg string, headers http.Header, body []byte, fields map[string]any)
