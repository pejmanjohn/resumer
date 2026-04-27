package runner

import (
	"path/filepath"
	"strings"

	"resumer/internal/session"
)

func TmuxSessionName(card session.SessionCard) string {
	name := card.ProjectPath
	if name != "" {
		name = filepath.Base(filepath.Clean(name))
	}
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = card.DisplayTitle()
	}

	harness := slugPart(string(card.Harness), "session")
	target := slugPart(name, "session")
	return "resumer-" + harness + "-" + target
}

func TmuxNewSessionArgv(card session.SessionCard) []string {
	return []string{
		"tmux",
		"new-session",
		"-A",
		"-s",
		TmuxSessionName(card),
		card.ResumeCommand().Display(),
	}
}

func slug(s string) string {
	return slugPart(s, "session")
}

func slugPart(s string, fallback string) string {
	var b strings.Builder
	previousHyphen := false

	for _, r := range strings.ToLower(s) {
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			b.WriteRune(r)
			previousHyphen = false
			continue
		}
		if !previousHyphen {
			b.WriteByte('-')
			previousHyphen = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}
