// Package executor provides functionality for actually firing parsed HTTP requests
// and retrieving the resulting status, headers, and payload.
package executor

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"httpee/pkg/parser"
)

// Response holds the metadata and payload from an executed HTTP query.
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
}

// Client holds the underlying net/http.Client wrapper used for outgoing connections.
type Client struct {
	httpClient *http.Client
}

// NewClient initializes a fresh Client with a standard 30-second timeout.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute triggers the HTTP request built by the parser package and 
// meticulously measures the time it takes to process the total response.
func (c *Client) Execute(req parser.Request) (*Response, error) {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewReader([]byte(req.Body))
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       bodyBytes,
		Duration:   duration,
	}, nil
}
