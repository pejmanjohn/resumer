package outfmt

import (
	"encoding/json"
	"io"
	"time"

	"resumer/internal/session"
)

type jsonDocument struct {
	Sessions []jsonSession `json:"sessions"`
}

type jsonSession struct {
	Harness     session.Harness `json:"harness"`
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	ProjectPath string          `json:"project_path,omitempty"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	FirstPrompt string          `json:"first_prompt,omitempty"`
	Model       string          `json:"model,omitempty"`
	SourcePath  string          `json:"source_path,omitempty"`
	Command     string          `json:"command"`
}

func WriteJSON(w io.Writer, cards []session.SessionCard) error {
	doc := jsonDocument{
		Sessions: make([]jsonSession, 0, len(cards)),
	}
	for _, card := range cards {
		doc.Sessions = append(doc.Sessions, jsonSession{
			Harness:     card.Harness,
			ID:          card.ID,
			Title:       card.DisplayTitle(),
			ProjectPath: card.ProjectPath,
			UpdatedAt:   timePtr(card.UpdatedAt),
			CreatedAt:   timePtr(card.CreatedAt),
			FirstPrompt: card.FirstPrompt,
			Model:       card.Model,
			SourcePath:  card.SourcePath,
			Command:     card.ResumeCommand().Display(),
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(doc)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	copy := t
	return &copy
}
