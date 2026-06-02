package parser

type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolClass     SymbolKind = "class"
	SymbolInterface SymbolKind = "interface"
	SymbolStruct    SymbolKind = "struct"
	SymbolEnum      SymbolKind = "enum"
)

type Symbol struct {
	Name      string
	Kind      SymbolKind
	StartByte uint32
	EndByte   uint32
	StartLine uint32
	EndLine   uint32
}

type Parser interface {
	ExtractSymbols(content []byte) ([]Symbol, error)
	Language() string
}
