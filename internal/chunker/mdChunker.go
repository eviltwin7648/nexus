package chunker

import "strings"

type MarkdownChunker struct {
	BaseChunker
}

func NewMarkDownChunker() *MarkdownChunker {
	return &MarkdownChunker{
		BaseChunker: NewBaseChunker(defaultChunkSize, defaultChunkOverlap),
	}
}

func (c *MarkdownChunker) Chunk(content []byte) ([]Segment, error) {
	return c.splitMarkdown(content)
}

// to prevent important context of markdown files from title and subtitle
// every header(h1, h2) are respected and not treated as a same sentance (new chunk each header)
func (c *MarkdownChunker) splitMarkdown(content []byte) ([]Segment, error) {
	text := string(content)

	lines := strings.Split(text, "\n")

	var sections []string
	var current strings.Builder

	for _, line := range lines {

		if (strings.HasPrefix(line, "# ") ||
			strings.HasPrefix(line, "## ")) &&
			current.Len() > 0 {

			sections = append(sections, current.String())
			current.Reset()
		}

		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		sections = append(sections, current.String())
	}

	var segments []Segment

	for _, section := range sections {

		words := strings.Fields(section)

		if len(words) <= c.chunkSize {

			segments = append(segments, Segment{
				Content: section,
				Metadata: map[string]any{
					"chunk_strategy": "markdown",
				},
			})

			continue
		}

		chunks, err := c.splitBySize([]byte(section))
		if err != nil {
			return nil, err
		}

		for _, chunk := range chunks {

			if chunk.Metadata == nil {
				chunk.Metadata = make(map[string]any)
			}

			chunk.Metadata["chunk_strategy"] = "markdown"

			segments = append(segments, chunk)
		}
	}

	return segments, nil
}
