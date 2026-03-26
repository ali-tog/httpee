package exporter

import (
	"strings"
	"testing"

	"httpee/pkg/parser"
)

func TestGenerateCurl(t *testing.T) {
	req := parser.Request{
		Method: "POST",
		URL:    "http://example.com/api",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"key": "value"}`,
	}

	curl := GenerateCurl(req)
	
	if !strings.Contains(curl, "curl -X POST 'http://example.com/api'") {
		t.Errorf("expected curl to contain base command, got %s", curl)
	}
	if !strings.Contains(curl, "-H 'Content-Type: application/json'") {
		t.Errorf("expected curl to contain header, got %s", curl)
	}
}

func TestGenerateFetch(t *testing.T) {
	req := parser.Request{
		Method: "POST",
		URL:    "http://example.com/api",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		Body: `{"foo": "bar"}`,
	}

	fetchStr := GenerateFetch(req)
	
	if !strings.Contains(fetchStr, "fetch('http://example.com/api', {") {
		t.Errorf("expected fetch code, got %s", fetchStr)
	}
}
