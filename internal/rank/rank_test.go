package rank

import (
	"reflect"
	"testing"
	"time"

	"github.com/pejmanjohn/resumer/internal/session"
)

func TestApplySortsByUpdatedAtDescendingAndLimits(t *testing.T) {
	older := card("older", session.HarnessClaude, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
	newer := card("newer", session.HarnessClaude, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	newest := card("newest", session.HarnessClaude, time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC))

	got := Apply([]session.SessionCard{older, newest, newer}, Options{Limit: 2})

	if gotIDs(got) != "newest,newer" {
		t.Fatalf("Apply() IDs = %s, want newest,newer", gotIDs(got))
	}
}

func TestApplyFiltersHarnessAndSidechainUnlessIncludeAll(t *testing.T) {
	claude := card("claude", session.HarnessClaude, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
	codex := card("codex", session.HarnessCodex, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	sidechain := card("sidechain", session.HarnessClaude, time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC))
	sidechain.Sidechain = true
	internal := card("internal", session.HarnessClaude, time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC))
	internal.Internal = true

	cards := []session.SessionCard{claude, codex, sidechain, internal}

	filtered := Apply(cards, Options{Harness: session.HarnessClaude})
	if gotIDs(filtered) != "claude" {
		t.Fatalf("filtered IDs = %s, want claude", gotIDs(filtered))
	}

	included := Apply(cards, Options{Harness: session.HarnessClaude, IncludeAll: true})
	if gotIDs(included) != "internal,sidechain,claude" {
		t.Fatalf("IncludeAll IDs = %s, want internal,sidechain,claude", gotIDs(included))
	}
}

func TestApplyFiltersCodexIndexOnlySessionsUnlessIncludeAll(t *testing.T) {
	complete := card("complete", session.HarnessCodex, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	complete.SourcePath = "/home/ada/.codex/sessions/complete.jsonl"
	indexOnly := card("index-only", session.HarnessCodex, time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC))
	indexOnly.SourcePath = ""

	filtered := Apply([]session.SessionCard{complete, indexOnly}, Options{})
	if gotIDs(filtered) != "complete" {
		t.Fatalf("filtered IDs = %s, want complete", gotIDs(filtered))
	}

	included := Apply([]session.SessionCard{complete, indexOnly}, Options{IncludeAll: true})
	if gotIDs(included) != "index-only,complete" {
		t.Fatalf("IncludeAll IDs = %s, want index-only,complete", gotIDs(included))
	}
}

func TestApplyCWDBiasPlacesMatchingProjectBeforeNewerNonMatch(t *testing.T) {
	newerNonMatch := card("newer-non-match", session.HarnessCodex, time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC))
	newerNonMatch.ProjectPath = "/other/project"
	olderExact := card("older-exact", session.HarnessCodex, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
	olderExact.ProjectPath = "/repo/app"
	olderUnder := card("older-under", session.HarnessCodex, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	olderUnder.ProjectPath = "/repo/app/service"

	got := Apply([]session.SessionCard{newerNonMatch, olderExact, olderUnder}, Options{CWDBiasPath: "/repo/app"})

	if gotIDs(got) != "older-under,older-exact,newer-non-match" {
		t.Fatalf("Apply() IDs = %s, want older-under,older-exact,newer-non-match", gotIDs(got))
	}
}

func TestApplyUsesDeterministicTieBreakerForEqualTimes(t *testing.T) {
	when := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	zulu := card("z-id", session.HarnessCodex, when)
	zulu.Title = "Zulu"
	alpha := card("a-id", session.HarnessClaude, when)
	alpha.Title = "Alpha"
	bravo := card("b-id", session.HarnessClaude, when)
	bravo.Title = "Bravo"

	got := Apply([]session.SessionCard{zulu, bravo, alpha}, Options{})

	if gotIDs(got) != "a-id,b-id,z-id" {
		t.Fatalf("Apply() IDs = %s, want a-id,b-id,z-id", gotIDs(got))
	}
}

func TestApplyDoesNotMutateInputOrdering(t *testing.T) {
	older := card("older", session.HarnessClaude, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
	newer := card("newer", session.HarnessClaude, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	input := []session.SessionCard{older, newer}
	before := append([]session.SessionCard(nil), input...)

	_ = Apply(input, Options{})

	if !reflect.DeepEqual(input, before) {
		t.Fatalf("Apply() mutated input: got %#v, want %#v", input, before)
	}
}

func card(id string, harness session.Harness, updated time.Time) session.SessionCard {
	card := session.SessionCard{
		Harness:   harness,
		ID:        id,
		Title:     id,
		UpdatedAt: updated,
	}
	if harness == session.HarnessCodex {
		card.SourcePath = "/home/ada/.codex/sessions/" + id + ".jsonl"
	}
	return card
}

func gotIDs(cards []session.SessionCard) string {
	ids := ""
	for i, card := range cards {
		if i > 0 {
			ids += ","
		}
		ids += card.ID
	}
	return ids
}
