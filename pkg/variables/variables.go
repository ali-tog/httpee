// Package variables provides variable definition parsing and token substitution
// for .http/.rest files following the REST Client specification.
//
// Supported syntax:
//
//	@name = value                   — defines a variable
//	@dotenv = dotenv("path/.env")   — loads variables from a .env file
//	{{name}}                        — substitutes a user-defined variable
//	{{$dotenv KEY}}                 — looks up KEY in loaded dotenv/var map
//	{{$ENV_VAR}}                    — substitutes an OS environment variable
//	{{$datetime iso8601}}           — current UTC time in RFC3339 format
//	{{$datetime rfc1123}}           — current UTC time in RFC1123 format
//	{{$timestamp}}                  — unix timestamp (seconds)
//	{{$guid}}                       — random UUID v4
//	{{$randomInt}}                  — random integer 0–999
package variables

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HistoryLoader is a function that retrieves body and headers for a historical request.
type HistoryLoader func(requestName string) (body []byte, headers map[string][]string, err error)

// varLineRe matches lines like:  @name = value
var varLineRe = regexp.MustCompile(`^@(\w+)\s*=\s*(.+)$`)

// dotenvRe matches the special directive: @dotenv = dotenv("relative/path")
var dotenvRe = regexp.MustCompile(`^@dotenv\s*=\s*dotenv\("([^"]+)"\)\s*$`)

// tokenRe matches all {{...}} tokens in a string.
var tokenRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ParseDefinitions remains unchanged.
func ParseDefinitions(lines []string, fileDir string) map[string]string {
	vars := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@") {
			continue
		}
		if dotenvRe.MatchString(line) {
			continue
		}
		if m := varLineRe.FindStringSubmatch(line); m != nil {
			vars[m[1]] = strings.TrimSpace(m[2])
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := dotenvRe.FindStringSubmatch(line); m != nil {
			envPath := m[1]
			if !filepath.IsAbs(envPath) {
				envPath = filepath.Join(fileDir, envPath)
			}
			dotenvVars, err := loadDotenv(envPath)
			if err != nil {
				continue
			}
			for k, v := range dotenvVars {
				if _, alreadyDefined := vars[k]; !alreadyDefined {
					vars[k] = v
				}
			}
		}
	}

	return vars
}

func loadDotenv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
			(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			result[key] = val
		}
	}
	return result, scanner.Err()
}

// Substitute replaces every {{token}} occurrence in s using the provided variable
// map, built-in dynamic variables, and historical response values via historyLoader.
func Substitute(s string, vars map[string]string, loader HistoryLoader) string {
	maxDepth := 10
	for i := 0; i < maxDepth; i++ {
		if !tokenRe.MatchString(s) {
			break
		}
		s = tokenRe.ReplaceAllStringFunc(s, func(match string) string {
			inner := strings.TrimSpace(match[2 : len(match)-2])
			
			// Built-ins
			if strings.HasPrefix(inner, "$") {
				parts := strings.Fields(inner)
				directive := parts[0]
				switch directive {
				case "$dotenv":
					if len(parts) > 1 {
						if val, ok := vars[parts[1]]; ok {
							return val
						}
					}
					return match
				case "$datetime":
					format := "iso8601"
					if len(parts) > 1 {
						format = parts[1]
					}
					switch format {
					case "rfc1123":
						return time.Now().UTC().Format(time.RFC1123)
					default:
						return time.Now().UTC().Format(time.RFC3339)
					}
				case "$timestamp":
					return strconv.FormatInt(time.Now().Unix(), 10)
				case "$guid":
					return newUUID()
				case "$randomInt":
					n, _ := rand.Int(rand.Reader, big.NewInt(1000))
					return n.String()
				default:
					envKey := strings.TrimPrefix(directive, "$")
					if val := os.Getenv(envKey); val != "" {
						return val
					}
				}
				return match
			}

			// User-defined variable map
			if val, ok := vars[inner]; ok {
				return val
			}

			// History references: reqName.response.body.path.to.prop or reqName.response.headers.HeaderName
			if loader != nil && strings.Contains(inner, ".response.") {
				parts := strings.SplitN(inner, ".response.", 2)
				if len(parts) == 2 {
					reqName := parts[0]
					target := parts[1]
					bodyBytes, headers, err := loader(reqName)
					if err == nil {
						if strings.HasPrefix(target, "headers.") {
							headerName := strings.TrimPrefix(target, "headers.")
							if vals, ok := headers[headerName]; ok && len(vals) > 0 {
								return vals[0]
							}
							// Case-insensitive fallback
							for k, v := range headers {
								if strings.EqualFold(k, headerName) && len(v) > 0 {
									return v[0]
								}
							}
						} else if strings.HasPrefix(target, "body.") || target == "body" {
							if target == "body" {
								return string(bodyBytes)
							}
							jsonPath := strings.TrimPrefix(target, "body.")
							if val, extractErr := extractJSONPath(bodyBytes, jsonPath); extractErr == nil {
								return val
							}
						}
					}
				}
			}

			// Leave unresolved token unchanged
			return match
		})
	}
	return s
}

// extractJSONPath does simple map traversal. E.g. "user.token" -> obj["user"]["token"].
func extractJSONPath(body []byte, path string) (string, error) {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	
	parts := strings.Split(path, ".")
	current := data
	for _, p := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			var found bool
			current, found = m[p]
			if !found {
				return "", fmt.Errorf("key %s not found", p)
			}
		} else {
			return "", fmt.Errorf("cannot traverse %v at key %s", current, p)
		}
	}
	
	// Convert final value to string
	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		// Format without trailing zeros unless necessary
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "null", nil
	default:
		// Array or nested object
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

// newUUID generates a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
