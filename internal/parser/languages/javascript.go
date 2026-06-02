package languages

import (
	"github.com/eviltwin7648/nexus/internal/parser"
	js "github.com/smacker/go-tree-sitter/javascript"
)

var jsQuery string

type JavascriptParser struct{}

func NewJavaScriptParser() *JavascriptParser {
	return &JavascriptParser{}
}

func (p *JavascriptParser) Language() string {
	return "javascript"
}

func (p *JavascriptParser) ExtractSymbols(content []byte) ([]parser.Symbol, error) {
	return parser.ExtractWithQuery(content, js.GetLanguage(), jsQuery, map[string]parser.SymbolKind{
		"function": parser.SymbolFunction,
		"class":    parser.SymbolClass,
		"method":   parser.SymbolMethod,
	})
}
