package languages

import (
	"github.com/eviltwin7648/nexus/internal/parser"
	py "github.com/smacker/go-tree-sitter/python"
)

var pythonQuery string

type PythonParser struct{}

func NewPythonParser() *PythonParser {
	return &PythonParser{}
}
func (p *PythonParser) Language() string {
	return "python"
}
func (p *PythonParser) ExtractSymbols(content []byte) ([]parser.Symbol, error) {
	return parser.ExtractWithQuery(content, py.GetLanguage(), pythonQuery, map[string]parser.SymbolKind{
		"function": parser.SymbolFunction,
		"class":    parser.SymbolClass,
	})
}
