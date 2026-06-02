package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eviltwin7648/nexus/internal/domain"
)

type Connector struct {
	token  string
	owner  string
	repo   string
	client *http.Client
}

func New(token, ownerRepo string) (*Connector, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q - expected owner/repo", ownerRepo)
	}
	return &Connector{
		token:  token,
		owner:  parts[0],
		repo:   parts[1],
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Connector) ID() string {
	return fmt.Sprintf("github:%s/%s", c.owner, c.repo)
}

func (c *Connector) get(ctx context.Context, path string, out any) error {
	url := fmt.Sprintf("https://api.github.com%s", path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-Github-Api-Version", "2022-11-28")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusUnauthorized && c.token != "" {
		resp.Body.Close()
		reqNoAuth, errNoAuth := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if errNoAuth != nil {
			return fmt.Errorf("build unauthenticated request: %w", errNoAuth)
		}
		reqNoAuth.Header.Set("Accept", "application/vnd.github+json")
		reqNoAuth.Header.Set("X-Github-Api-Version", "2022-11-28")
		respNoAuth, errNoAuth := c.client.Do(reqNoAuth)
		if errNoAuth != nil {
			return fmt.Errorf("http get (unauthenticated) %s: %w", path, errNoAuth)
		}
		resp = respNoAuth
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api %s returned %d: %s", path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Connector) Fetch(ctx context.Context) ([]domain.RawDocument, error) {
	var docs []domain.RawDocument
	files, err := c.fetchFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch files: %w", err)
	}

	docs = append(docs, files...)

	issues, err := c.fetchIssues(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch issues: %w", err)
	}
	docs = append(docs, issues...)
	prs, err := c.fetchPRs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch prs: %w", err)
	}
	docs = append(docs, prs...)

	return docs, nil
}

func (c *Connector) Diff(ctx context.Context, since time.Time) ([]domain.RawDocument, error) {
	var docs []domain.RawDocument
	//skipping file diff for now
	// refetching entire files for now

	issues, err := c.fetchIssuesSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("diff issues: %w", err)
	}
	docs = append(docs, issues...)

	prs, err := c.fetchPRsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("diff prs: %w", err)
	}
	docs = append(docs, prs...)

	return docs, nil
}

// fetching files
type githubTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

type githubTree struct {
	Tree []githubTreeEntry `json:"tree"`
}

func (c *Connector) fetchFiles(ctx context.Context) ([]domain.RawDocument, error) {
	var tree githubTree
	err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/git/trees/HEAD?recursive=1", c.owner, c.repo), &tree)
	if err != nil {
		return nil, err
	}
	var docs []domain.RawDocument
	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		if !isTextFile(entry.Path) {
			continue
		}
		doc, err := c.fetchFileContent(ctx, entry.Path)
		if err != nil {
			fmt.Printf("warn: skip file %s: %v\n", entry.Path, err)
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

type githubFileResponse struct {
	Content string `json:"content"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

func (c *Connector) fetchFileContent(ctx context.Context, path string) (domain.RawDocument, error) {
	var file githubFileResponse
	err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/contents/%s", c.owner, c.repo, path),
		&file)
	if err != nil {
		return domain.RawDocument{}, err
	}
	// GitHub returns content as base64 with newlines — clean and decode
	cleaned := strings.ReplaceAll(file.Content, "\n", "")
	contentBytes, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return domain.RawDocument{}, fmt.Errorf("decode file content: %w", err)
	}
	content := string(contentBytes)
	sourceID := c.ID()
	return domain.RawDocument{
		ID:         makeId(sourceID, path),
		SourceId:   sourceID,
		SourceType: domain.SourceTypeGitHubFiles,
		Path:       path,
		Title:      file.Name,
		Content:    content,
		Metadata: map[string]any{
			"language": detectLanguage(path),
		},
		URL:       file.HTMLURL,
		Checksum:  makeCheckSum(content),
		UpdatedAt: time.Now(),
	}, nil
}

// fetching issues

type githubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`

	Body      string    `json:"body"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	UpdatedAt time.Time `json:"updated_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	PullRequest *struct{} `json:"pull_request"`
}

func (c *Connector) fetchIssues(ctx context.Context) ([]domain.RawDocument, error) {
	return c.fetchIssuesSince(ctx, time.Time{})
}

func (c *Connector) fetchIssuesSince(ctx context.Context, since time.Time) ([]domain.RawDocument, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?state=all&per_page=100", c.owner, c.repo)
	if !since.IsZero() {
		path += "&since=" + since.Format(time.RFC3339)
	}
	var issues []githubIssue
	if err := c.get(ctx, path, &issues); err != nil {
		return nil, err
	}
	sourceId := c.ID()
	var docs []domain.RawDocument
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}
		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}
		content := fmt.Sprintf("# %s\n\n%s", issue.Title, issue.Body)
		issuePath := fmt.Sprintf("issues/%d", issue.Number)
		docs = append(docs, domain.RawDocument{
			ID:         makeId(sourceId, issuePath),
			SourceId:   sourceId,
			SourceType: domain.SourceTypeGitHubIssue,
			Path:       issuePath,
			Title:      issue.Title,
			Content:    content,
			Metadata: map[string]any{
				"number": issue.Number,
				"state":  issue.State,
				"author": issue.User.Login,
				"labels": labels,
			},
			URL:       issue.HTMLURL,
			Checksum:  makeCheckSum(content),
			UpdatedAt: issue.UpdatedAt,
		})
	}
	return docs, nil
}

//fetch PRs

type githubPR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (c *Connector) fetchPRs(ctx context.Context) ([]domain.RawDocument, error) {
	return c.fetchPRsSince(ctx, time.Time{})
}

func (c *Connector) fetchPRsSince(ctx context.Context, since time.Time) ([]domain.RawDocument, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=all&per_page=100", c.owner, c.repo)
	//we'll need manual since filter (PRs API doesn't support it natively)
	var prs []githubPR
	if err := c.get(ctx, path, &prs); err != nil {
		return nil, err
	}
	sourceID := c.ID()
	var docs []domain.RawDocument
	for _, pr := range prs {
		if !since.IsZero() && pr.UpdatedAt.Before(since) {
			continue
		}
		content := fmt.Sprintf("# %s\n\n%s", pr.Title, pr.Body)
		prPath := fmt.Sprintf("pulls/%d", pr.Number)

		docs = append(docs, domain.RawDocument{
			ID:         makeId(sourceID, prPath),
			SourceId:   sourceID,
			SourceType: domain.SOurceTypeGitHubPR,
			Path:       prPath,
			Title:      pr.Title,
			Content:    content,
			Metadata: map[string]any{
				"number":      pr.Number,
				"state":       pr.State,
				"author":      pr.User.Login,
				"head_branch": pr.Head.Ref,
				"base_branch": pr.Base.Ref,
			},
			URL:       pr.HTMLURL,
			Checksum:  makeCheckSum(content),
			UpdatedAt: pr.UpdatedAt,
		})
	}
	return docs, nil
}
func isTextFile(path string) bool {
	skip := []string{
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot",
		".zip", ".tar", ".gz", ".jar", ".pdf",
		".lock", ".sum",
	}
	lower := strings.ToLower(path)
	for _, ext := range skip {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") {
		return false
	}
	if strings.HasPrefix(path, ".obsidian/") {
		return false
	}
	return true
}

func makeId(sourceId, path string) string {
	h := sha256.Sum256([]byte(sourceId + ":" + path))
	return fmt.Sprintf("%x", h)
}

func detectLanguage(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	ext := strings.ToLower(parts[len(parts)-1])
	langs := map[string]string{
		"go":   "go",
		"ts":   "typescript",
		"js":   "javascript",
		"py":   "python",
		"md":   "markdown",
		"sql":  "sql",
		"yaml": "yaml",
		"yml":  "yaml",
		"json": "json",
		"sh":   "shell",
	}
	if l, ok := langs[ext]; ok {
		return l
	}
	return ext
}

func makeCheckSum(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}
