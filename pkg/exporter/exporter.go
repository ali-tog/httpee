// Package exporter manages IO operations corresponding to user 'actions' on the CLI,
// including dispatching to the clipboard and writing payloads natively to files.
package exporter

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"httpee/pkg/parser"
)

// CopyToClipboard forces the given text representation directly out to the system clipboard.
func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// GenerateCurl cleanly converts an internal parsed Request into an executable bash cURL command schema.
func GenerateCurl(req parser.Request) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("curl -X %s '%s'", req.Method, req.URL))
	for k, v := range req.Headers {
		sb.WriteString(fmt.Sprintf(" -H '%s: %s'", k, v))
	}
	if req.Body != "" {
		escapedBody := strings.ReplaceAll(req.Body, "'", "'\\''")
		sb.WriteString(fmt.Sprintf(" -d '%s'", escapedBody))
	}
	return sb.String()
}

// GenerateFetch generates a workable javascript fetch(...) command string utilizing the active request bindings.
func GenerateFetch(req parser.Request) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("fetch('%s', {\n", req.URL))
	sb.WriteString(fmt.Sprintf("  method: '%s',\n", req.Method))
	if len(req.Headers) > 0 {
		sb.WriteString("  headers: {\n")
		for k, v := range req.Headers {
			escapedKey := strings.ReplaceAll(k, "'", "\\'")
			escapedVal := strings.ReplaceAll(v, "'", "\\'")
			sb.WriteString(fmt.Sprintf("    '%s': '%s',\n", escapedKey, escapedVal))
		}
		sb.WriteString("  },\n")
	}
	if req.Body != "" {
		escapedBody := strings.ReplaceAll(req.Body, "`", "\\`")
		sb.WriteString(fmt.Sprintf("  body: `%s`,\n", escapedBody))
	}
	sb.WriteString("})\n.then(response => response.json())\n.then(data => console.log(data));")
	return sb.String()
}
