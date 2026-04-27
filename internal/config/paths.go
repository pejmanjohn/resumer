package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Paths struct {
	ClaudeProjectsPath string
	CodexSessionsPath  string
	CodexIndexPath     string
	DefaultTmux        bool
	TmuxHostHint       string
}

func LoadPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	paths := Paths{
		ClaudeProjectsPath: filepath.Join(home, ".claude", "projects"),
		CodexSessionsPath:  filepath.Join(home, ".codex", "sessions"),
		CodexIndexPath:     filepath.Join(home, ".codex", "session_index.jsonl"),
	}

	if value := os.Getenv("RESUMER_CLAUDE_PROJECTS_PATH"); value != "" {
		paths.ClaudeProjectsPath = value
	}
	if value := os.Getenv("RESUMER_CODEX_SESSIONS_PATH"); value != "" {
		paths.CodexSessionsPath = value
	}
	if value := os.Getenv("RESUMER_CODEX_INDEX_PATH"); value != "" {
		paths.CodexIndexPath = value
	}
	if value := os.Getenv("RESUMER_DEFAULT_TMUX"); value != "" {
		defaultTmux, err := strconv.ParseBool(value)
		if err != nil {
			return Paths{}, fmt.Errorf("invalid RESUMER_DEFAULT_TMUX: %w", err)
		}
		paths.DefaultTmux = defaultTmux
	}
	if value := os.Getenv("RESUMER_TMUX_HOST_HINT"); value != "" {
		paths.TmuxHostHint = value
	}

	return paths, nil
}
