package tui

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func HighlightJSON(input string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, input)
	if err != nil {
		return input
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return input
	}

	return buf.String()
}
