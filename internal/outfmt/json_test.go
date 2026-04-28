package outfmt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pejmanjohn/resumer/internal/session"
)

func TestWriteJSONStableFieldsAndCommandDisplayForCodexCardWithProjectPath(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          "codex-123",
		Title:       "Fix parser",
		ProjectPath: "/repo/app",
		UpdatedAt:   time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC),
		CreatedAt:   time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		FirstPrompt: "Please fix the parser",
		Model:       "gpt-5",
		SourcePath:  "/home/user/.codex/session.jsonl",
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, []session.SessionCard{card}); err != nil {
		t.Fatalf("WriteJSON() returned error: %v", err)
	}

	want := "{\n  \"sessions\": [\n    {\n      \"harness\": \"codex\",\n      \"id\": \"codex-123\",\n      \"title\": \"Fix parser\",\n      \"project_path\": \"/repo/app\",\n      \"updated_at\": \"2026-04-02T10:00:00Z\",\n      \"created_at\": \"2026-04-01T09:00:00Z\",\n      \"first_prompt\": \"Please fix the parser\",\n      \"model\": \"gpt-5\",\n      \"source_path\": \"/home/user/.codex/session.jsonl\",\n      \"command\": \"codex resume codex-123 --cd /repo/app\"\n    }\n  ]\n}\n"
	if got := buf.String(); got != want {
		t.Fatalf("JSON =\n%s\nwant =\n%s", got, want)
	}
}

func TestWriteJSONOmitsZeroEmptyOptionalFieldsAndSidechainInternal(t *testing.T) {
	card := session.SessionCard{
		Harness:   session.HarnessClaude,
		ID:        "claude-123",
		Title:     "Debug issue",
		Sidechain: true,
		Internal:  true,
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, []session.SessionCard{card}); err != nil {
		t.Fatalf("WriteJSON() returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		`"harness": "claude"`,
		`"id": "claude-123"`,
		`"title": "Debug issue"`,
		`"command": "claude --resume claude-123"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("JSON = %s, want to contain %s", got, want)
		}
	}
	for _, forbidden := range []string{
		"project_path",
		"updated_at",
		"created_at",
		"first_prompt",
		"model",
		"source_path",
		"sidechain",
		"internal",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("JSON = %s, want to omit %s", got, forbidden)
		}
	}
}

func TestWriteJSONUsesDisplayTitleFallbackWhenTitleIsBlank(t *testing.T) {
	card := session.SessionCard{
		Harness:     session.HarnessCodex,
		ID:          "codex-1234567890",
		FirstPrompt: "  Investigate failing tests  ",
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, []session.SessionCard{card}); err != nil {
		t.Fatalf("WriteJSON() returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"title": "Investigate failing tests"`) {
		t.Fatalf("JSON = %s, want DisplayTitle fallback", buf.String())
	}
}

func TestWriteJSONPreservesInputOrder(t *testing.T) {
	first := session.SessionCard{Harness: session.HarnessClaude, ID: "first", Title: "First"}
	second := session.SessionCard{Harness: session.HarnessCodex, ID: "second", Title: "Second"}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, []session.SessionCard{first, second}); err != nil {
		t.Fatalf("WriteJSON() returned error: %v", err)
	}

	var doc struct {
		Sessions []struct {
			ID string `json:"id"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v", err)
	}
	if len(doc.Sessions) != 2 || doc.Sessions[0].ID != "first" || doc.Sessions[1].ID != "second" {
		t.Fatalf("session IDs = %#v, want first then second", doc.Sessions)
	}
}
