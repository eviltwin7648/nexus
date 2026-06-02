package chunker

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/eviltwin7648/nexus/internal/parser"
)

const (
	defaultChunkSize    = 400  //words per chunk
	defaultChunkOverlap = 80   //chunk overlap to maintain context during retrirval
	maxChunkChars       = 6000 // hard cap
)

type Segment struct {
	Content  string
	Metadata map[string]any
}
type Strategy interface {
	Chunk(content []byte) ([]Segment, error)
}

type Chunk struct {
	ID         string
	DocId      string
	SourceId   string
	SourceType domain.SourceType
	Index      int
	Content    string
	Metadata   map[string]any
}

type Chunker struct {
	chunkSize      int
	chunkOverlap   int
	parserRegistry *parser.Registry
}

func New(registry *parser.Registry) *Chunker {
	return &Chunker{
		chunkSize:      defaultChunkSize,
		chunkOverlap:   defaultChunkOverlap,
		parserRegistry: registry,
	}
}

func (c *Chunker) Chunk(doc domain.RawDocument) ([]Chunk, error) {
	fileType := ClassifyFile(doc.Path, []byte(doc.Content))
	strategy, err := BuildStrategy(
		fileType, c.parserRegistry,
	)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}
	segments, err := strategy.Chunk(
		[]byte(doc.Content),
	)
	if err != nil {
		return nil, err
	}
	return c.buildChunks(
		doc,
		segments,
	), nil
}

func makeChunkId(docId string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", docId, index)))
	return fmt.Sprintf("%x", h)
}

func (c *Chunker) buildChunks(
	doc domain.RawDocument,
	segments []Segment,
) []Chunk {
	chunks := make([]Chunk, 0, len(segments))
	for i, seg := range segments {
		if strings.TrimSpace(seg.Content) == "" {
			continue
		}
		content := seg.Content
		if len(content) > maxChunkChars {
			content = content[:maxChunkChars]
		}

		meta := make(map[string]any)
		//copy doc and seg metadaata
		for k, v := range doc.Metadata {
			meta[k] = v
		}
		for k, v := range seg.Metadata {
			meta[k] = v
		}

		meta["chunk_index"] = i
		meta["chunk_total"] = len(segments)

		meta["doc_title"] = doc.Title
		meta["doc_path"] = doc.Path
		meta["doc_url"] = doc.URL

		chunks = append(chunks, Chunk{
			ID:         makeChunkId(doc.ID, i),
			DocId:      doc.ID,
			SourceId:   doc.SourceId,
			SourceType: doc.SourceType,
			Index:      i,
			Content:    content,
			Metadata:   meta,
		})
	}
	return chunks
}
