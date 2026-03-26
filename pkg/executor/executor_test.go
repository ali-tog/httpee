package executor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"httpee/pkg/parser"
)

func TestExecutor(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer secret" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status": "ok"}`)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"error": "unauthorized"}`)
		}
	}))
	defer ts.Close()

	client := NewClient()

	reqSuccess := parser.Request{
		Name:   "Auth Request",
		Method: "GET",
		URL:    ts.URL,
		Headers: map[string]string{
			"Authorization": "Bearer secret",
		},
	}
	resp, err := client.Execute(reqSuccess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
