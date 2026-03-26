package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"httpee/pkg/executor"
	"httpee/pkg/parser"
)

// HistoryEntry strictly structures a historic request mapping to a response.
type HistoryEntry struct {
	Timestamp   time.Time           `json:"timestamp"`
	DurationMs  int64               `json:"duration_ms"`
	RequestName string              `json:"request_name"`
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	StatusCode  int                 `json:"status_code"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"`
}

func getHistoryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".httpee", "history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// Save writes an entry both to a chronological log and caches the latest response by name.
func Save(req parser.Request, resp *executor.Response) error {
	dir, err := getHistoryDir()
	if err != nil {
		return err
	}

	entry := HistoryEntry{
		Timestamp:   time.Now().UTC(),
		DurationMs:  resp.Duration.Milliseconds(),
		RequestName: req.Name,
		Method:      req.Method,
		URL:         req.URL,
		StatusCode:  resp.StatusCode,
		Headers:     resp.Headers,
		Body:        string(resp.Body),
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Append to chronological log
	logFile := filepath.Join(dir, "history.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	f.Write(b)
	f.WriteString("\n")
	f.Close()

	// Update latest cache for the query name safely
	if req.Name != "" {
		safeName := SanitizeFileName(req.Name)
		latestFile := filepath.Join(dir, fmt.Sprintf("%s.json", safeName))
		if err := os.WriteFile(latestFile, b, 0644); err != nil {
			return err
		}
	}
	return nil
}

// GetLatest retrieves the most recent response for a given request name.
func GetLatest(reqName string) (*HistoryEntry, error) {
	dir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}
	safeName := SanitizeFileName(reqName)
	latestFile := filepath.Join(dir, fmt.Sprintf("%s.json", safeName))
	b, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}
	var entry HistoryEntry
	if err := json.Unmarshal(b, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// SanitizeFileName cleans up the parsed request name so it's a safe UNIX path.
func SanitizeFileName(name string) string {
	b := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b[i] = c
		} else {
			b[i] = '_'
		}
	}
	return string(b)
}

// ReadLog returns up to the last `limit` history entries from the chronological log, newest first.
func ReadLog(limit int) ([]HistoryEntry, error) {
	dir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}
	logFile := filepath.Join(dir, "history.log")
	b, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No history yet
		}
		return nil, err
	}

	lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
	var entries []HistoryEntry
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) == 0 {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(lines[i], &entry); err == nil {
			entries = append(entries, entry)
			if limit > 0 && len(entries) >= limit {
				break
			}
		}
	}
	return entries, nil
}
