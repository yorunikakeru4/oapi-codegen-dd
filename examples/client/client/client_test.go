package client

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockOptions struct {
	pathParams map[string]any
	query      map[string]any
	body       any
	headers    map[string]string
}

func (m *mockOptions) GetPathParams() (map[string]any, error) {
	return m.pathParams, nil
}

func (m *mockOptions) GetQuery() (map[string]any, error) {
	return m.query, nil
}

func (m *mockOptions) GetBody() any {
	return m.body
}

func (m *mockOptions) GetHeader() (map[string]string, error) {
	return m.headers, nil
}

func TestCreateRequest(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		url             string
		options         RequestOptions
		contentType     string
		expectedURL     string
		expectedBody    string
		expectedCT      string
		expectedHeaders http.Header
	}{
		{
			name:   "GET with query",
			method: "GET",
			url:    "https://example.com/api/resource",
			options: &mockOptions{
				query: map[string]any{
					"foo": "bar",
					"id":  123,
				},
			},
			expectedURL: "https://example.com/api/resource?foo=bar&id=123",
			expectedHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name:        "GET without query",
			method:      "GET",
			url:         "https://example.com/api/resource",
			options:     &mockOptions{},
			expectedURL: "https://example.com/api/resource",
			expectedHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name:   "POST with JSON body",
			method: "POST",
			url:    "https://example.com/api/resource",
			options: &mockOptions{
				body: map[string]any{
					"name": "Alice",
				},
				headers: map[string]string{
					"x-custom-header": "value",
					"Content-Type":    "application/json",
				},
			},
			expectedURL:  "https://example.com/api/resource",
			expectedBody: `{"name":"Alice"}`,
			expectedHeaders: http.Header{
				"Content-Type":    []string{"application/json"},
				"X-Custom-Header": []string{"value"},
			},
		},
		{
			name:   "POST with form body",
			method: "POST",
			url:    "https://example.com/api/resource",
			options: &mockOptions{
				body: struct {
					Email string `url:"email"`
					Token string `url:"token"`
				}{
					Email: "test@example.com",
					Token: "abc123",
				},
			},
			contentType:  "application/x-www-form-urlencoded",
			expectedURL:  "https://example.com/api/resource",
			expectedBody: "email=test%40example.com&token=abc123",
			expectedHeaders: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
		{
			name:   "POST without body and headers with query",
			method: "POST",
			url:    "https://example.com/api/resource",
			options: &mockOptions{
				query: map[string]any{
					"foo": "bar",
					"id":  123,
				},
			},
			expectedURL:  "https://example.com/api/resource?foo=bar&id=123",
			expectedBody: "",
			expectedHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := RequestOptionsParameters{
				reqURL:      tt.url,
				method:      tt.method,
				options:     tt.options,
				contentType: tt.contentType,
			}
			req, err := createRequest(context.Background(), params)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, req.URL.String())
			assert.Equal(t, tt.method, req.Method)
			assert.Equal(t, tt.expectedHeaders, req.Header)

			if tt.expectedBody != "" {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, string(body))
			} else {
				assert.Nil(t, req.Body)
			}
		})
	}
}
