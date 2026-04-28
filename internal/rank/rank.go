package rank

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/pejmanjohn/resumer/internal/session"
)

type Options struct {
	Harness     session.Harness
	IncludeAll  bool
	CWDBiasPath string
	Limit       int
}

func Apply(cards []session.SessionCard, opts Options) []session.SessionCard {
	ranked := make([]session.SessionCard, 0, len(cards))
	for _, card := range cards {
		if opts.Harness != "" && card.Harness != opts.Harness {
			continue
		}
		if !opts.IncludeAll && (card.Sidechain || card.Internal) {
			continue
		}
		if !opts.IncludeAll && isCodexIndexOnly(card) {
			continue
		}
		ranked = append(ranked, card)
	}

	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]

		if opts.CWDBiasPath != "" {
			leftMatch := matchesCWD(left.ProjectPath, opts.CWDBiasPath)
			rightMatch := matchesCWD(right.ProjectPath, opts.CWDBiasPath)
			if leftMatch != rightMatch {
				return leftMatch
			}
		}

		leftTime := left.SortTime()
		rightTime := right.SortTime()
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}

		return stableKey(left) < stableKey(right)
	})

	if opts.Limit > 0 && len(ranked) > opts.Limit {
		ranked = ranked[:opts.Limit]
	}
	return ranked
}

func matchesCWD(projectPath string, cwd string) bool {
	if projectPath == "" || cwd == "" {
		return false
	}

	cleanProject := filepath.Clean(projectPath)
	cleanCWD := filepath.Clean(cwd)
	if cleanProject == cleanCWD {
		return true
	}
	return strings.HasPrefix(cleanProject, cleanCWD+string(filepath.Separator))
}

func isCodexIndexOnly(card session.SessionCard) bool {
	return card.Harness == session.HarnessCodex && card.SourcePath == "" && card.ProjectPath == ""
}

func stableKey(card session.SessionCard) string {
	return strings.Join([]string{
		string(card.Harness),
		card.DisplayTitle(),
		card.ID,
		card.ProjectPath,
		card.SourcePath,
	}, "\x00")
}
