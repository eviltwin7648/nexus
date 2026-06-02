package chunker

import (
	"fmt"

	"github.com/eviltwin7648/nexus/internal/parser"
)

func BuildStrategy(ft FileType, registery *parser.Registry) (Strategy, error) {
	switch ft.ChunkingStrategy {
	case ASTChunking:
		p, err := registery.Get(ft.Language)
		if err != nil {
			return nil, err
		}
		return NewASTChunker(p), nil

	case HeadingChunking:
		return NewMarkDownChunker(), nil
	case StatementChunking:
		return NewStatementChunker(), nil
	case BasicChunking:
		return NewBasicChunker(), nil
	case SkipChunking:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported strategy")
	}
}
