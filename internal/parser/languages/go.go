package languages

import (
	"github.com/eviltwin7648/nexus/internal/parser"
	golang "github.com/smacker/go-tree-sitter/golang"
)

var goQuery string

type GoParser struct{}

func NewGoParser() *GoParser {
	return &GoParser{}
}

func (p *GoParser) Language() string {
	return "go"
}

func (p *GoParser) ExtractSymbols(content []byte) ([]parser.Symbol, error) {
	return parser.ExtractWithQuery(content, golang.GetLanguage(), goQuery, map[string]parser.SymbolKind{
		"function":  parser.SymbolFunction,
		"method":    parser.SymbolMethod,
		"struct":    parser.SymbolStruct,
		"interface": parser.SymbolInterface,
	})
}
