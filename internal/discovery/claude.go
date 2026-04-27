package discovery

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"resumer/internal/session"
)

type Diagnostic struct {
	Source  string
	Message string
}

type ClaudeOptions struct {
	ProjectsPath string
	IncludeAll   bool
}

func DiscoverClaude(opts ClaudeOptions) ([]session.SessionCard, []Diagnostic) {
	pattern := filepath.Join(opts.ProjectsPath, "*", "sessions-index.json")
	indexPaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, []Diagnostic{{Source: pattern, Message: err.Error()}}
	}

	cards := make([]session.SessionCard, 0)
	diagnostics := make([]Diagnostic, 0)
	seen := map[string]bool{}

	for _, indexPath := range indexPaths {
		index, diagnostic, ok := readClaudeIndex(indexPath)
		if !ok {
			diagnostics = append(diagnostics, diagnostic)
		} else {
			for _, entry := range index.Entries {
				card := entry.card()
				if card.ID == "" || seen[card.ID] {
					continue
				}
				if card.Sidechain && !opts.IncludeAll {
					seen[card.ID] = true
					continue
				}

				cards = append(cards, card)
				seen[card.ID] = true
			}
		}

		projectDir := filepath.Dir(indexPath)
		fallbackCards, fallbackDiagnostics := discoverClaudeJSONLInProject(projectDir, opts, seen)
		cards = append(cards, fallbackCards...)
		diagnostics = append(diagnostics, fallbackDiagnostics...)
		for _, card := range fallbackCards {
			seen[card.ID] = true
		}
	}

	transcriptPattern := filepath.Join(opts.ProjectsPath, "*", "*.jsonl")
	transcriptPaths, err := filepath.Glob(transcriptPattern)
	if err != nil {
		diagnostics = append(diagnostics, Diagnostic{Source: transcriptPattern, Message: err.Error()})
	} else {
		for _, transcriptPath := range transcriptPaths {
			if hasIndex(filepath.Dir(transcriptPath), indexPaths) {
				continue
			}
			card, ok, diagnostic := parseClaudeJSONL(transcriptPath)
			if !ok {
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			if card.ID == "" || seen[card.ID] {
				continue
			}
			if card.Sidechain && !opts.IncludeAll {
				continue
			}
			cards = append(cards, card)
			seen[card.ID] = true
		}
	}

	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].SortTime().After(cards[j].SortTime())
	})

	return cards, diagnostics
}

func discoverClaudeJSONLInProject(projectDir string, opts ClaudeOptions, seen map[string]bool) ([]session.SessionCard, []Diagnostic) {
	pattern := filepath.Join(projectDir, "*.jsonl")
	transcriptPaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, []Diagnostic{{Source: pattern, Message: err.Error()}}
	}

	cards := make([]session.SessionCard, 0)
	diagnostics := make([]Diagnostic, 0)
	for _, transcriptPath := range transcriptPaths {
		card, ok, diagnostic := parseClaudeJSONL(transcriptPath)
		if !ok {
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		if card.ID == "" || seen[card.ID] {
			continue
		}
		if card.Sidechain && !opts.IncludeAll {
			continue
		}
		cards = append(cards, card)
		seen[card.ID] = true
	}

	return cards, diagnostics
}

func hasIndex(projectDir string, indexPaths []string) bool {
	indexPath := filepath.Join(projectDir, "sessions-index.json")
	for _, existing := range indexPaths {
		if existing == indexPath {
			return true
		}
	}
	return false
}

type claudeIndex struct {
	Entries []claudeIndexEntry `json:"entries"`
}

type claudeIndexEntry struct {
	SessionID   string `json:"sessionId"`
	Summary     string `json:"summary"`
	Title       string `json:"title"`
	FirstPrompt string `json:"firstPrompt"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	ProjectPath string `json:"projectPath"`
	FullPath    string `json:"fullPath"`
	IsSidechain bool   `json:"isSidechain"`
}

func readClaudeIndex(path string) (claudeIndex, Diagnostic, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return claudeIndex{}, Diagnostic{Source: path, Message: err.Error()}, false
	}

	var index claudeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return claudeIndex{}, Diagnostic{Source: path, Message: err.Error()}, false
	}

	return index, Diagnostic{}, true
}

func (entry claudeIndexEntry) card() session.SessionCard {
	title := strings.TrimSpace(entry.Summary)
	if title == "" {
		title = strings.TrimSpace(entry.Title)
	}

	return session.SessionCard{
		Harness:     session.HarnessClaude,
		ID:          strings.TrimSpace(entry.SessionID),
		Title:       title,
		ProjectPath: strings.TrimSpace(entry.ProjectPath),
		CreatedAt:   parseClaudeTime(entry.Created),
		UpdatedAt:   parseClaudeTime(entry.Modified),
		FirstPrompt: strings.TrimSpace(entry.FirstPrompt),
		SourcePath:  strings.TrimSpace(entry.FullPath),
		Sidechain:   entry.IsSidechain,
	}
}

func parseClaudeJSONL(path string) (session.SessionCard, bool, Diagnostic) {
	file, err := os.Open(path)
	if err != nil {
		return session.SessionCard{}, false, Diagnostic{Source: path, Message: err.Error()}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	card := session.SessionCard{
		Harness:    session.HarnessClaude,
		SourcePath: path,
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event claudeJSONLEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if card.ID == "" {
			card.ID = strings.TrimSpace(event.SessionID)
		}
		if card.ProjectPath == "" {
			card.ProjectPath = strings.TrimSpace(event.CWD)
		}
		if event.Sidechain {
			card.Sidechain = true
		}

		timestamp := parseClaudeTime(event.Timestamp)
		if !timestamp.IsZero() {
			if card.CreatedAt.IsZero() || timestamp.Before(card.CreatedAt) {
				card.CreatedAt = timestamp
			}
			if card.UpdatedAt.IsZero() || timestamp.After(card.UpdatedAt) {
				card.UpdatedAt = timestamp
			}
		}

		if card.FirstPrompt == "" && event.Type == "user" && event.Message.Role == "user" {
			card.FirstPrompt = strings.TrimSpace(event.Message.contentString())
		}
	}

	if err := scanner.Err(); err != nil {
		return session.SessionCard{}, false, Diagnostic{Source: path, Message: err.Error()}
	}
	if card.ID == "" {
		return session.SessionCard{}, false, Diagnostic{Source: path, Message: "missing session ID"}
	}

	return card, true, Diagnostic{}
}

type claudeJSONLEvent struct {
	Type      string             `json:"type"`
	SessionID string             `json:"sessionId"`
	Timestamp string             `json:"timestamp"`
	CWD       string             `json:"cwd"`
	Sidechain bool               `json:"isSidechain"`
	Message   claudeJSONLMessage `json:"message"`
}

type claudeJSONLMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (message claudeJSONLMessage) contentString() string {
	var content string
	if err := json.Unmarshal(message.Content, &content); err == nil {
		return content
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(message.Content, &blocks); err == nil {
		for _, block := range blocks {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				return block.Text
			}
		}
	}
	return ""
}

func parseClaudeTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return timestamp
}
