package history_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"httpee/pkg/executor"
	"httpee/pkg/history"
	"httpee/pkg/parser"
	"httpee/pkg/variables"
)

func TestSaveAndGetLatest(t *testing.T) {
	// Setup req/resp
	req := parser.Request{
		Name:   "login",
		Method: "POST",
		URL:    "https://api.example.com/v1/login",
	}

	respBody := `{"token": "secret123", "user": {"id": 42}}`
	resp := &executor.Response{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(respBody),
		Duration:   120 * time.Millisecond,
	}

	// 1. Save
	err := history.Save(req, resp)
	if err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// 2. GetLatest
	latest, err := history.GetLatest("login")
	if err != nil {
		t.Fatalf("Failed to get latest history: %v", err)
	}

	if string(latest.Body) != respBody {
		t.Errorf("expected body %q, got %q", respBody, string(latest.Body))
	}

	// 3. Test Variable Substitution using variables.Substitute
	loader := func(reqName string) ([]byte, map[string][]string, error) {
		entry, e := history.GetLatest(reqName)
		if e != nil {
			return nil, nil, e
		}
		return []byte(entry.Body), entry.Headers, nil
	}

	// Test URL replacement
	urlToken := "https://api.example.com/user/{{login.response.body.user.id}}"
	resolvedURL := variables.Substitute(urlToken, map[string]string{}, loader)
	if resolvedURL != "https://api.example.com/user/42" {
		t.Errorf("expected resolved URL to be https://api.example.com/user/42, got %q", resolvedURL)
	}

	// Test Body replacement
	authHeaderToken := "Bearer {{login.response.body.token}}"
	resolvedHeader := variables.Substitute(authHeaderToken, map[string]string{}, loader)
	if resolvedHeader != "Bearer secret123" {
		t.Errorf("expected resolved header to be Bearer secret123, got %q", resolvedHeader)
	}

	// Test Header replacement
	contentTypeToken := "{{login.response.headers.Content-Type}}"
	resolvedCT := variables.Substitute(contentTypeToken, map[string]string{}, loader)
	if resolvedCT != "application/json" {
		t.Errorf("expected resolved CT to be application/json, got %q", resolvedCT)
	}

	// Clean up: delete files
	home, _ := os.UserHomeDir()
	os.RemoveAll(filepath.Join(home, ".httpee", "history"))
}
