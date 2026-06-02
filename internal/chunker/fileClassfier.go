package chunker

import (
	"path/filepath"
	"strings"
)

var extMap = map[string]FileType{
	".ts": {
		Category:         "source_code",
		Language:         "typescript",
		ChunkingStrategy: ASTChunking,
	},
	".tsx": {
		Category:         "source_code",
		Language:         "tsx",
		ChunkingStrategy: ASTChunking,
	},
	".js": {
		Category:         "source_code",
		Language:         "javascript",
		ChunkingStrategy: ASTChunking,
	},
	".jsx": {
		Category:         "source_code",
		Language:         "jsx",
		ChunkingStrategy: ASTChunking,
	},
	".java": {
		Category:         "source_code",
		Language:         "java",
		ChunkingStrategy: ASTChunking,
	},
	".py": {
		Category:         "source_code",
		Language:         "python",
		ChunkingStrategy: ASTChunking,
	},
	".md": {
		Category:         "documentation",
		Language:         "markdown",
		ChunkingStrategy: HeadingChunking,
	},
	".sql": {
		Category:         "sql",
		Language:         "sql",
		ChunkingStrategy: StatementChunking,
	},
	".go": {
		Category:         "source_code",
		Language:         "go",
		ChunkingStrategy: ASTChunking,
	},
}

var fileMap = map[string]FileType{
	"dockerfile": {
		Category:         "config",
		Language:         "dockerfile",
		ChunkingStrategy: StatementChunking,
	},
}

type ChunkingStrategy string

const (
	ASTChunking       ChunkingStrategy = "ast"
	HeadingChunking   ChunkingStrategy = "heading"
	StatementChunking ChunkingStrategy = "statement"
	BasicChunking     ChunkingStrategy = "basic"
	SkipChunking      ChunkingStrategy = "skip"
)

type FileClassifier interface {
	Classify(path string, content []byte) FileType
}

type FileType struct {
	Category         string
	Language         string
	Parser           string
	ChunkingStrategy ChunkingStrategy
	SkipReason       string
}

func isBinary(content []byte) bool {
	limit := len(content)
	if limit > 8000 {
		limit = 8000
	}
	for i := 0; i < limit; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

func ClassifyFile(path string, content []byte) FileType {
	ext := strings.ToLower(filepath.Ext(path))
	if ft, ok := extMap[ext]; ok {
		return ft
	}
	base := strings.ToLower(filepath.Base(path))
	if ft, ok := fileMap[base]; ok {
		return ft
	}

	if isBinary(content) {
		return FileType{
			Category:         "binary",
			Language:         "binary",
			ChunkingStrategy: SkipChunking,
			SkipReason:       "binary file",
		}
	}

	return FileType{
		Category:         "text",
		Language:         "text",
		ChunkingStrategy: BasicChunking,
	}
}
