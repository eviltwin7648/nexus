package languages

import (
	"github.com/eviltwin7648/nexus/internal/parser"
	ts "github.com/smacker/go-tree-sitter/typescript/typescript"
)

var tsQuery string

type TypeScriptParser struct{}

func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{}
}

func (p *TypeScriptParser) Language() string {
	return "typescript"
}

func (p *TypeScriptParser) ExtractSymbols(content []byte) ([]parser.Symbol, error) {
	return parser.ExtractWithQuery(content, ts.GetLanguage(), tsQuery, map[string]parser.SymbolKind{
		"function":  parser.SymbolFunction,
		"method":    parser.SymbolMethod,
		"class":     parser.SymbolClass,
		"enum":      parser.SymbolEnum,
		"interface": parser.SymbolInterface,
	})
}
