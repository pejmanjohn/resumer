package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resumer/internal/session"
)

func TestDiscoverCodexFromIndexDeduplicatesAndEnrichesTranscript(t *testing.T) {
	indexPath := filepath.Join("testdata", "codex", "session_index.jsonl")
	sessionsPath := filepath.Join("testdata", "codex", "sessions")

	cards, diagnostics := DiscoverCodex(CodexOptions{
		IndexPath:    indexPath,
		SessionsPath: sessionsPath,
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

	card := seen["codex-session-1"]
	if card.Harness != session.HarnessCodex {
		t.Fatalf("Harness = %q, want %q", card.Harness, session.HarnessCodex)
	}
	if card.Title != "Explore calculator app updated" {
		t.Fatalf("Title = %q, want newest title", card.Title)
	}
	assertTime(t, card.UpdatedAt, "2026-04-26T14:00:00Z")
	assertTime(t, card.CreatedAt, "2026-04-26T10:00:00Z")
	if card.ProjectPath != "/repo/calc" {
		t.Fatalf("ProjectPath = %q, want /repo/calc", card.ProjectPath)
	}
	if card.Model != "openai" {
		t.Fatalf("Model = %q, want openai", card.Model)
	}
	wantSource := filepath.Join(sessionsPath, "2026", "04", "26", "rollout-2026-04-26T10-00-00-codex-session-1.jsonl")
	if card.SourcePath != wantSource {
		t.Fatalf("SourcePath = %q, want %q", card.SourcePath, wantSource)
	}
}

func TestDiscoverCodexBlankThreadNameFallsBackToDisplayTitle(t *testing.T) {
	cards, diagnostics := DiscoverCodex(CodexOptions{
		IndexPath:    filepath.Join("testdata", "codex", "session_index.jsonl"),
		SessionsPath: filepath.Join("testdata", "codex", "sessions"),
	})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}

	var card session.SessionCard
	for _, candidate := range cards {
		if candidate.ID == "codex-session-2" {
			card = candidate
		}
	}
	if card.ID == "" {
		t.Fatal("codex-session-2 not found")
	}
	if card.Title != "" {
		t.Fatalf("Title = %q, want blank", card.Title)
	}
	if got, want := card.DisplayTitle(), "session ession-2"; got != want {
		t.Fatalf("DisplayTitle() = %q, want %q", got, want)
	}
}

func TestDiscoverCodexMissingIndexReturnsDiagnostic(t *testing.T) {
	cards, diagnostics := DiscoverCodex(CodexOptions{
		IndexPath:    filepath.Join(t.TempDir(), "missing.jsonl"),
		SessionsPath: filepath.Join("testdata", "codex", "sessions"),
	})
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

func TestDiscoverCodexMalformedIndexLineDoesNotAbortValidLines(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "session_index.jsonl")
	data := []byte(`{"id":"valid-session","thread_name":"Valid session","updated_at":"2026-04-26T12:00:00Z"}
{"id":
{"id":"another-valid-session","thread_name":"Another valid session","updated_at":"2026-04-26T13:00:00Z"}
`)
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cards, diagnostics := DiscoverCodex(CodexOptions{IndexPath: indexPath})
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one malformed line diagnostic", diagnostics)
	}
	if !strings.Contains(diagnostics[0].Message, "line") {
		t.Fatalf("diagnostic = %#v, want line context", diagnostics[0])
	}
}

func TestDiscoverCodexKeepsIndexSessionWithoutTranscript(t *testing.T) {
	cards, diagnostics := DiscoverCodex(CodexOptions{
		IndexPath:    filepath.Join("testdata", "codex", "session_index.jsonl"),
		SessionsPath: filepath.Join(t.TempDir(), "no-sessions"),
	})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}

	seen := map[string]session.SessionCard{}
	for _, card := range cards {
		seen[card.ID] = card
	}
	card := seen["codex-session-1"]
	if card.ID == "" {
		t.Fatal("codex-session-1 not found")
	}
	if card.ProjectPath != "" || card.Model != "" || card.SourcePath != "" || !card.CreatedAt.IsZero() {
		t.Fatalf("card = %#v, want index-only fields without transcript enrichment", card)
	}
}

func TestParseCodexSessionMetaIgnoresMismatchedIDs(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-26T10:00:00Z","type":"session_meta","payload":{"id":"other-session","cwd":"/repo/other","model_provider":"openai"}}`)

	meta, ok := parseCodexSessionMeta(line, "codex-session-1")
	if ok {
		t.Fatalf("ok = true, want false with meta %#v", meta)
	}
}
