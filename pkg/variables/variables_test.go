package variables_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"httpee/pkg/variables"
)

func TestParseDefinitions_Basic(t *testing.T) {
	lines := []string{
		"@host = https://example.com",
		"@token = abc123",
		"# comment line",
		"",
		"GET {{host}}/users",
	}
	vars := variables.ParseDefinitions(lines, ".")
	if vars["host"] != "https://example.com" {
		t.Errorf("expected host to be 'https://example.com', got %q", vars["host"])
	}
	if vars["token"] != "abc123" {
		t.Errorf("expected token to be 'abc123', got %q", vars["token"])
	}
}

func TestParseDefinitions_Dotenv(t *testing.T) {
	// Write a temp .env file.
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	err := os.WriteFile(envFile, []byte("API_KEY=supersecret\nDB_HOST=\"localhost\"\n# comment\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to create temp .env: %v", err)
	}

	lines := []string{
		`@dotenv = dotenv(".env")`,
		"@override = manual",
	}
	vars := variables.ParseDefinitions(lines, dir)
	if vars["API_KEY"] != "supersecret" {
		t.Errorf("expected API_KEY='supersecret', got %q", vars["API_KEY"])
	}
	if vars["DB_HOST"] != "localhost" {
		t.Errorf("expected DB_HOST='localhost', got %q", vars["DB_HOST"])
	}
	if vars["override"] != "manual" {
		t.Errorf("expected override='manual', got %q", vars["override"])
	}
}

func TestParseDefinitions_DotenvDoesNotOverrideExplicit(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	err := os.WriteFile(envFile, []byte("host=from-dotenv\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to create temp .env: %v", err)
	}

	lines := []string{
		"@host = explicit-value",
		`@dotenv = dotenv(".env")`,
	}
	vars := variables.ParseDefinitions(lines, dir)
	if vars["host"] != "explicit-value" {
		t.Errorf("expected explicit @name to win; got %q", vars["host"])
	}
}

func TestSubstitute_UserVariable(t *testing.T) {
	vars := map[string]string{"host": "https://api.example.com", "id": "42"}
	result := variables.Substitute("GET {{host}}/users/{{id}}", vars, nil)
	want := "GET https://api.example.com/users/42"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestSubstitute_EnvVariable(t *testing.T) {
	t.Setenv("MY_SECRET", "hunter2")
	result := variables.Substitute("{{$MY_SECRET}}", nil, nil)
	if result != "hunter2" {
		t.Errorf("expected env var substitution, got %q", result)
	}
}

func TestSubstitute_Datetime(t *testing.T) {
	result := variables.Substitute("{{$datetime iso8601}}", nil, nil)
	if len(result) < 10 {
		t.Errorf("expected a date string, got %q", result)
	}
	if strings.Contains(result, "{{") {
		t.Errorf("token was not substituted: %q", result)
	}
}

func TestSubstitute_Timestamp(t *testing.T) {
	result := variables.Substitute("{{$timestamp}}", nil, nil)
	if len(result) < 9 {
		t.Errorf("expected unix timestamp, got %q", result)
	}
}

func TestSubstitute_Guid(t *testing.T) {
	result := variables.Substitute("{{$guid}}", nil, nil)
	parts := strings.Split(result, "-")
	if len(parts) != 5 {
		t.Errorf("expected UUID format, got %q", result)
	}
}

func TestSubstitute_UnknownTokenLeftAsIs(t *testing.T) {
	result := variables.Substitute("{{unknown}}", nil, nil)
	if result != "{{unknown}}" {
		t.Errorf("expected token to be unchanged, got %q", result)
	}
}
