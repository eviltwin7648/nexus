package domain

import (
	"context"
	"time"
)

type SourceType string

const (
	SourceTypeGitHubFiles SourceType = "github_file"
	SourceTypeGitHubIssue SourceType = "github_issue"
	SOurceTypeGitHubPR    SourceType = "github_pr"
)

type RawDocument struct {
	ID         string
	SourceId   string
	SourceType SourceType
	Path       string
	Title      string
	Content    string
	Metadata   map[string]any
	URL        string
	Checksum   string
	UpdatedAt  time.Time
}

type Connector interface {
	Fetch(ctx context.Context) ([]RawDocument, error)
	// Watch(ctx context.Context, ch chan<- RawDocument)
	Diff(ctx context.Context, since time.Time) ([]RawDocument, error)
}
