package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"resumer/internal/session"
)

func TestDiscoverClaudeFromIndexReturnsNonSidechainByDefault(t *testing.T) {
	cards, diagnostics := DiscoverClaude(ClaudeOptions{
		ProjectsPath: filepath.Join("testdata", "claude", "projects"),
	})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}

	card := cards[0]
	if card.Harness != session.HarnessClaude {
		t.Fatalf("Harness = %q, want %q", card.Harness, session.HarnessClaude)
	}
	if card.ID != "claude-session-1" {
		t.Fatalf("ID = %q, want claude-session-1", card.ID)
	}
	if card.Title != "Resumer CLI design" {
		t.Fatalf("Title = %q, want Resumer CLI design", card.Title)
	}
	if card.FirstPrompt != "Plan the resume picker" {
		t.Fatalf("FirstPrompt = %q, want Plan the resume picker", card.FirstPrompt)
	}
	if card.ProjectPath != "/repo/project-a" {
		t.Fatalf("ProjectPath = %q, want /repo/project-a", card.ProjectPath)
	}
	if card.SourcePath != "/home/ada/.claude/projects/project-a/claude-session-1.jsonl" {
		t.Fatalf("SourcePath = %q, want fixture full path", card.SourcePath)
	}
	if card.Sidechain {
		t.Fatalf("Sidechain = true, want false")
	}
	assertTime(t, card.CreatedAt, "2026-04-26T10:00:00Z")
	assertTime(t, card.UpdatedAt, "2026-04-26T12:00:00Z")
}

func TestDiscoverClaudeIncludeAllReturnsSidechainEntries(t *testing.T) {
	cards, diagnostics := DiscoverClaude(ClaudeOptions{
		ProjectsPath: filepath.Join("testdata", "claude", "projects"),
		IncludeAll:   true,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}

	seen := map[string]session.SessionCard{}
	for _, card := range cards {
		seen[card.ID] = card
	}
	if !seen["claude-sidechain-1"].Sidechain {
		t.Fatalf("sidechain card not returned with Sidechain=true: %#v", seen["claude-sidechain-1"])
	}
}

func TestParseClaudeJSONLExtractsSessionMetadata(t *testing.T) {
	path := filepath.Join("testdata", "claude", "projects", "project-a", "claude-session-1.jsonl")

	card, ok, diagnostic := parseClaudeJSONL(path)
	if !ok {
		t.Fatalf("ok = false, diagnostic = %#v", diagnostic)
	}
	if diagnostic != (Diagnostic{}) {
		t.Fatalf("diagnostic = %#v, want zero", diagnostic)
	}
	if card.Harness != session.HarnessClaude {
		t.Fatalf("Harness = %q, want %q", card.Harness, session.HarnessClaude)
	}
	if card.ID != "claude-session-1" {
		t.Fatalf("ID = %q, want claude-session-1", card.ID)
	}
	if card.FirstPrompt != "Plan the resume picker" {
		t.Fatalf("FirstPrompt = %q, want Plan the resume picker", card.FirstPrompt)
	}
	if card.ProjectPath != "/repo/project-a" {
		t.Fatalf("ProjectPath = %q, want /repo/project-a", card.ProjectPath)
	}
	if card.SourcePath != path {
		t.Fatalf("SourcePath = %q, want %q", card.SourcePath, path)
	}
	assertTime(t, card.CreatedAt, "2026-04-26T10:00:00Z")
	assertTime(t, card.UpdatedAt, "2026-04-26T10:02:00Z")
}

func TestParseClaudeJSONLExtractsFirstPromptFromContentBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "content-blocks.jsonl")
	data := []byte(`{"type":"summary","sessionId":"claude-block-session","timestamp":"2026-04-26T10:00:00Z"}
{"type":"user","sessionId":"claude-block-session","timestamp":"2026-04-26T10:01:00Z","cwd":"/repo/project-a","message":{"role":"user","content":[{"type":"text","text":"Plan from content block"}]}}
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	card, ok, diagnostic := parseClaudeJSONL(path)
	if !ok {
		t.Fatalf("ok = false, diagnostic = %#v", diagnostic)
	}
	if card.FirstPrompt != "Plan from content block" {
		t.Fatalf("FirstPrompt = %q, want Plan from content block", card.FirstPrompt)
	}
}

func TestDiscoverClaudeMalformedIndexRecordsDiagnostic(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "bad-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "sessions-index.json"), []byte(`{"entries": [`), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, diagnostics := DiscoverClaude(ClaudeOptions{ProjectsPath: root})
	if len(cards) != 0 {
		t.Fatalf("cards = %#v, want none", cards)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one", diagnostics)
	}
	if diagnostics[0].Source == "" || diagnostics[0].Message == "" {
		t.Fatalf("diagnostic = %#v, want source and message", diagnostics[0])
	}
}

func TestDiscoverClaudeUnreadableIndexRecordsDiagnostic(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "bad-project", "sessions-index.json")
	if err := os.MkdirAll(indexPath, 0o755); err != nil {
		t.Fatal(err)
	}

	cards, diagnostics := DiscoverClaude(ClaudeOptions{ProjectsPath: root})
	if len(cards) != 0 {
		t.Fatalf("cards = %#v, want none", cards)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one", diagnostics)
	}
	if diagnostics[0].Source != indexPath || diagnostics[0].Message == "" {
		t.Fatalf("diagnostic = %#v, want source %q and message", diagnostics[0], indexPath)
	}
}

func TestDiscoverClaudeDeduplicatesSessionIDs(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	index := `{
  "entries": [
    {
      "sessionId": "duplicate-session",
      "summary": "Same session",
      "firstPrompt": "Help me",
      "created": "2026-04-26T10:00:00Z",
      "modified": "2026-04-26T10:30:00Z",
      "projectPath": "/repo/project-a",
      "fullPath": "/sanitized/duplicate.jsonl",
      "isSidechain": false
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(first, "sessions-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "sessions-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, diagnostics := DiscoverClaude(ClaudeOptions{ProjectsPath: root})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
	if cards[0].ID != "duplicate-session" {
		t.Fatalf("ID = %q, want duplicate-session", cards[0].ID)
	}
}

func assertTime(t *testing.T, got time.Time, want string) {
	t.Helper()

	wantTime, err := time.Parse(time.RFC3339, want)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(wantTime) {
		t.Fatalf("time = %s, want %s", got.Format(time.RFC3339), want)
	}
}
