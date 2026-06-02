package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractWithQuery(
	content []byte,
	lang *sitter.Language,
	querySrc string,
	captureKinds map[string]SymbolKind,
) ([]Symbol, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}
	if tree == nil {
		return nil, fmt.Errorf("nil syntax tree")
	}
	defer tree.Close()
	query, err := sitter.NewQuery([]byte(querySrc), lang)
	if err != nil {
		return nil, fmt.Errorf("compile query: %w", err)
	}
	defer query.Close()
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(query, tree.RootNode())

	var symbols []Symbol
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			node := capture.Node
			captureName := query.CaptureNameForId(capture.Index)
			kind, exits := captureKinds[captureName]
			if !exits {
				continue
			}
			start := node.StartPoint()
			end := node.EndPoint()
			symbols = append(symbols, Symbol{
				Kind:      kind,
				StartByte: node.StartByte(),
				EndByte:   node.EndByte(),
				StartLine: start.Row + 1,
				EndLine:   end.Row + 1,
			})
		}
	}
	return symbols, nil
}
