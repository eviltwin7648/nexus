package chunker

import "github.com/eviltwin7648/nexus/internal/parser"

type ASTChunker struct {
	parser parser.Parser
}

func NewASTChunker(p parser.Parser) *ASTChunker {
	return &ASTChunker{
		parser: p,
	}
}

func (c *ASTChunker) Chunk(content []byte) ([]Segment, error) {
	symbols, err := c.parser.ExtractSymbols(content)
	if err != nil {
		return nil, err
	}

	var segments []Segment
	for _, sym := range symbols {
		if sym.EndByte <= sym.StartByte {
			continue
		}

		segments = append(segments, Segment{
			Content: string(content[sym.StartByte:sym.EndByte]),
			Metadata: map[string]any{
				"symbol_kind": string(sym.Kind),
				"start_line":  sym.StartLine,
				"end_line":    sym.EndLine,
			},
		})
	}
	return segments, nil
}
