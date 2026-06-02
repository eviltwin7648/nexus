package languages

import (
	"github.com/eviltwin7648/nexus/internal/parser"
	"github.com/smacker/go-tree-sitter/java"
)

var javaQuery string

type JavaParser struct{}

func NewJavaParser() *JavaParser {
	return &JavaParser{}
}
func (p *JavaParser) Language() string {
	return "java"
}

func (p *JavaParser) ExtractSymbols(content []byte) ([]parser.Symbol, error) {
	return parser.ExtractWithQuery(content, java.GetLanguage(), javaQuery, map[string]parser.SymbolKind{
		"class":     parser.SymbolClass,
		"interface": parser.SymbolFunction,
		"enum":      parser.SymbolEnum,
		"method":    parser.SymbolMethod,
	})
}
