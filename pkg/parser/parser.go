// Package parser extracts HTTP requests from standard .http and .rest files
package parser

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"httpee/pkg/variables"
)

// Request represents a single parsed HTTP request
type Request struct {
	Name     string
	Method   string
	URL      string
	Headers  map[string]string
	Body     string
	Original string
}

// Parse consumes an io.Reader (typically a file or string reader) and extracts
// a slice of all structured Request elements separated by `###`.
// It also returns a map of all variable definitions found in the file
// (both @name = value and @dotenv = dotenv("path") expansions).
// fileDir is used to resolve relative dotenv paths; pass an empty string or "."
// when loading from a non-file source.
func Parse(r io.Reader, fileDir string) ([]Request, map[string]string, error) {
	scanner := bufio.NewScanner(r)
	var allLines []string
	var requests []Request
	var currentLines []string
	var currentName string

	for scanner.Scan() {
		line := scanner.Text()
		allLines = append(allLines, line)

		if strings.HasPrefix(line, "###") {
			if len(currentLines) > 0 {
				if req := parseSingleRequest(currentLines, currentName); req != nil {
					requests = append(requests, *req)
				}
				currentLines = nil
			}
			currentName = strings.TrimSpace(strings.TrimPrefix(line, "###"))
			continue
		}
		currentLines = append(currentLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if len(currentLines) > 0 {
		if req := parseSingleRequest(currentLines, currentName); req != nil {
			requests = append(requests, *req)
		}
	}

	vars := variables.ParseDefinitions(allLines, fileDir)
	return requests, vars, nil
}

func parseSingleRequest(lines []string, name string) *Request {
	var startIndex int
	var found bool
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			startIndex = i
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	lines = lines[startIndex:]
	if len(lines) == 0 {
		return nil
	}

	var reqLine string
	var reqLineIndex int

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		// Extract name if provided via `# @name requestName`
		if strings.HasPrefix(trimmed, "# @name ") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# @name "))
		}
		// Skip comment lines and variable definition lines
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "@") {
			continue
		}
		if reqLine == "" {
			reqLine = trimmed
			reqLineIndex = i
			break
		}
	}

	if reqLine == "" {
		return nil
	}

	parts := strings.Fields(reqLine)
	method := "GET"
	url := ""

	if len(parts) >= 2 {
		method = strings.ToUpper(parts[0])
		url = parts[1]
	} else if len(parts) == 1 {
		url = parts[0]
	}

	req := &Request{
		Name:     name,
		Method:   method,
		URL:      url,
		Headers:  make(map[string]string),
		Original: strings.Join(lines, "\n"),
	}
	if req.Name == "" {
		req.Name = req.Method + " " + req.URL
	}

	var bodyBuf bytes.Buffer
	inBody := false

	for i := reqLineIndex + 1; i < len(lines); i++ {
		line := lines[i]
		if inBody {
			bodyBuf.WriteString(line + "\n")
			continue
		}
		if strings.TrimSpace(line) == "" {
			inBody = true
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx != -1 {
			key := strings.TrimSpace(line[:colonIdx])
			val := strings.TrimSpace(line[colonIdx+1:])
			req.Headers[key] = val
		}
	}
	req.Body = strings.TrimSpace(bodyBuf.String())
	return req
}
