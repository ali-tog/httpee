package parser
import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `### Get User
GET https://api.github.com/users/octocat
Authorization: Bearer token123

### Update User
# This is a comment
PUT https://api.github.com/users/octocat
Content-Type: application/json

{
  "name": "Octocat"
}
`
	reqs, _, err := Parse(strings.NewReader(input), ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
}
