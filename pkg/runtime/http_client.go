package runtime

import "time"

// HTTPCall is a recorded http call client made.
// Method is the HTTP method used for the request.
// URL is the full URL of the request.
// Path is the masked path of the request, for example "/v1/payments/{id}".
// ResponseCode is the HTTP response code received. In case of failed request, this will be zero.
// Latency is the time it took to complete the request.
type HTTPCall struct {
	Method       string
	URL          string
	Path         string
	ResponseCode int
	Latency      time.Duration
}

type HTTPCallRecorder interface {
	Record(HTTPCall)
}
