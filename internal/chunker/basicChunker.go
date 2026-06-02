package chunker

import "strings"

type BaseChunker struct {
	chunkSize    int
	chunkOverlap int
}

func NewBaseChunker(size, overlap int) BaseChunker {
	return BaseChunker{
		chunkSize:    size,
		chunkOverlap: overlap,
	}
}

func (c *BaseChunker) splitBySize(content []byte) ([]Segment, error) {
	text := string(content)

	words := strings.Fields(text)

	if len(words) == 0 {
		return nil, nil
	}

	if len(words) <= c.chunkSize {
		return []Segment{
			{
				Content:  text,
				Metadata: map[string]any{},
			},
		}, nil
	}

	var segments []Segment

	start := 0

	for start < len(words) {
		end := start + c.chunkSize

		if end > len(words) {
			end = len(words)
		}

		segments = append(segments, Segment{
			Content: strings.Join(words[start:end], " "),
			Metadata: map[string]any{
				"chunk_strategy": "size",
			},
		})

		start += c.chunkSize - c.chunkOverlap
	}

	return segments, nil
}

type BasicChunker struct {
	BaseChunker
}

func NewBasicChunker() *BasicChunker {
	return &BasicChunker{
		BaseChunker: NewBaseChunker(defaultChunkSize, defaultChunkOverlap),
	}
}

func (c *BasicChunker) Chunk(content []byte) ([]Segment, error) {
	return c.splitBySize(content)
}
