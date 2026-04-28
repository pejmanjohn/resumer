package discovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pejmanjohn/resumer/internal/session"
)

type CodexOptions struct {
	IndexPath    string
	SessionsPath string
}

func DiscoverCodex(opts CodexOptions) ([]session.SessionCard, []Diagnostic) {
	cardsByID, diagnostics := readCodexIndex(opts.IndexPath)
	if len(cardsByID) == 0 {
		return nil, diagnostics
	}

	if opts.SessionsPath != "" {
		enrichCodexCards(cardsByID, opts.SessionsPath)
	}

	cards := make([]session.SessionCard, 0, len(cardsByID))
	for _, card := range cardsByID {
		cards = append(cards, card)
	}
	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].SortTime().After(cards[j].SortTime())
	})

	return cards, diagnostics
}

func readCodexIndex(path string) (map[string]session.SessionCard, []Diagnostic) {
	file, err := os.Open(path)
	if err != nil {
		return nil, []Diagnostic{{Source: path, Message: err.Error()}}
	}
	defer file.Close()

	cards := map[string]session.SessionCard{}
	diagnostics := make([]Diagnostic, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry codexIndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Source:  path,
				Message: fmt.Sprintf("line %d: %s", lineNumber, err.Error()),
			})
			continue
		}

		card := entry.card()
		if card.ID == "" {
			continue
		}
		existing, exists := cards[card.ID]
		if !exists || card.UpdatedAt.After(existing.UpdatedAt) {
			cards[card.ID] = card
		}
	}
	if err := scanner.Err(); err != nil {
		diagnostics = append(diagnostics, Diagnostic{Source: path, Message: err.Error()})
	}

	return cards, diagnostics
}

type codexIndexEntry struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
	UpdatedAt  string `json:"updated_at"`
}

func (entry codexIndexEntry) card() session.SessionCard {
	return session.SessionCard{
		Harness:   session.HarnessCodex,
		ID:        strings.TrimSpace(entry.ID),
		Title:     strings.TrimSpace(entry.ThreadName),
		UpdatedAt: parseClaudeTime(entry.UpdatedAt),
	}
}

func enrichCodexCards(cards map[string]session.SessionCard, sessionsPath string) {
	_ = filepath.WalkDir(sessionsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}

		for id, card := range cards {
			if card.SourcePath != "" || !strings.Contains(filepath.Base(path), id) {
				continue
			}
			if meta, ok := parseCodexSessionMetaFile(path, id); ok {
				card.ProjectPath = meta.ProjectPath
				card.Model = meta.Model
				card.CreatedAt = meta.CreatedAt
				card.Internal = meta.Internal
				card.SourcePath = path
				cards[id] = card
			}
		}
		return nil
	})
}

func parseCodexSessionMetaFile(path string, id string) (codexSessionMeta, bool) {
	file, err := os.Open(path)
	if err != nil {
		return codexSessionMeta{}, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		if meta, ok := parseCodexSessionMeta([]byte(scanner.Text()), id); ok {
			return meta, true
		}
	}
	return codexSessionMeta{}, false
}

type codexSessionMeta struct {
	ProjectPath string
	Model       string
	CreatedAt   time.Time
	Internal    bool
}

func parseCodexSessionMeta(line []byte, id string) (codexSessionMeta, bool) {
	var event struct {
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`
		Payload   struct {
			ID            string          `json:"id"`
			CWD           string          `json:"cwd"`
			ModelProvider string          `json:"model_provider"`
			Source        json.RawMessage `json:"source"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(line, &event); err != nil {
		return codexSessionMeta{}, false
	}
	if event.Type != "session_meta" || strings.TrimSpace(event.Payload.ID) != id {
		return codexSessionMeta{}, false
	}

	return codexSessionMeta{
		ProjectPath: strings.TrimSpace(event.Payload.CWD),
		Model:       strings.TrimSpace(event.Payload.ModelProvider),
		CreatedAt:   parseClaudeTime(event.Timestamp),
		Internal:    codexSourceIsSubagent(event.Payload.Source),
	}, true
}

func codexSourceIsSubagent(source json.RawMessage) bool {
	var parsed struct {
		Subagent *struct {
			ThreadSpawn json.RawMessage `json:"thread_spawn"`
		} `json:"subagent"`
	}
	if err := json.Unmarshal(source, &parsed); err != nil {
		return false
	}
	return parsed.Subagent != nil
}
