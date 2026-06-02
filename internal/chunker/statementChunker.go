package chunker

type StatementChunker struct {
	BaseChunker
}

func NewStatementChunker() *StatementChunker {
	return &StatementChunker{
		BaseChunker: NewBaseChunker(defaultChunkSize, defaultChunkOverlap),
	}
}
func (c *StatementChunker) Chunk(content []byte) ([]Segment, error) {
	return c.splitBySize(content)
}
