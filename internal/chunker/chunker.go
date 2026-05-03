package chunker

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/eviltwin7648/nexus/internal/domain"
)

const (
	defaultChunkSize    = 400  //words per chunk
	defaultChunkOverlap = 80   //chunk overlap to maintain context during retrirval
	maxChunkChars       = 6000 // hard cap
)

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
	chunkSize    int
	chunkOverlap int
}

func New() *Chunker {
	return &Chunker{
		chunkSize:    defaultChunkSize,
		chunkOverlap: defaultChunkOverlap,
	}
}

func (c *Chunker) Chunk(doc domain.RawDocument) []Chunk {
	var segments []string

	switch doc.SourceType {
	case domain.SOurceTypeGitHubPR, domain.SourceTypeGitHubIssue:
		segments = c.splitBySize(doc.Content)

	case domain.SourceTypeGitHubFiles:
		lang, _ := doc.Metadata["language"].(string)
		if lang == "markdown" {
			segments = c.splitMarkdown(doc.Content)
		} else {
			segments = c.splitBySize(doc.Content)
		}
	default:
		segments = c.splitBySize(doc.Content)
	}

	chunks := make([]Chunk, len(segments))
	for i, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		meta := make(map[string]any, len(doc.Metadata)+3) // chunk index, chunk total
		for k, v := range doc.Metadata {
			meta[k] = v
		}
		meta["chunk_index"] = i
		meta["chunk_total"] = len(segments)
		meta["doc_title"] = doc.Title
		meta["doc_path"] = doc.Path
		meta["doc_url"] = doc.URL

		chunks = append(chunks, Chunk{
			ID:       makeChunkId(doc.ID, i),
			DocId:    doc.ID,
			SourceId: doc.SourceId,
			Index:    i,
			Content:  seg,
			Metadata: meta,
		})
	}
	for i := range chunks {
		if len(chunks[i].Content) > maxChunkChars {
			chunks[i].Content = chunks[i].Content[:maxChunkChars]
		}
	}
	return chunks
}

func (c *Chunker) splitBySize(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) < c.chunkSize {
		return []string{text} // no splitting needed if text is < chunksize
	}
	var segments []string

	//sliding window with overlap(DSA ref)
	start := 0
	for start < len(words) {
		end := start + c.chunkSize
		// start - end = window size of fixed length (chunksize)
		if end > len(words) {
			end = len(words)
		}
		segment := strings.Join(words[start:end], " ")
		segments = append(segments, segment)
		start += c.chunkSize - c.chunkOverlap
	}
	return segments
}

// to prevent important context of markdown files from title and subtitle
// every header(h1, h2) are respected and not treated as a same sentance (new chunk each header)
func (c *Chunker) splitMarkdown(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {

		if (strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ")) &&
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

	var segments []string
	for _, section := range sections {
		words := strings.Fields(section)
		if len(words) <= c.chunkSize {
			segments = append(segments, section)
		} else {
			segments = append(segments, c.splitBySize(section)...)
		}
	}
	return segments
}

func makeChunkId(docId string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", docId, index)))
	return fmt.Sprintf("%x", h)
}
