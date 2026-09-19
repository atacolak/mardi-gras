package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/digest"
)

func TestCurrentDigestIsChromeNotParade(t *testing.T) {
	m := New([]data.Issue{
		{ID: "x-1", Title: "One", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeTask},
	}, data.Source{Mode: data.SourceJSONL, Path: "/tmp/x.jsonl"}, nil)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := nm.(Model)
	if model.chromeHeight() != 1 {
		t.Fatalf("unpublished digest must not grow chrome, got %d", model.chromeHeight())
	}

	now := time.Date(2026, 9, 19, 5, 5, 0, 0, time.UTC)
	nm, _ = model.Update(digestMsg{digest: &digest.Digest{
		GeneratedAt: now.Add(-90 * time.Second),
		Sprint:      &digest.Sprint{Epic: "mard-wte", Phase: "executing"},
		Now:         &digest.Now{Count: 2, Frontier: []digest.Frontier{{ID: "mard-wte.13"}}},
		Narrative:   &digest.Narrative{Text: "Shared summary only."},
	}})
	model = nm.(Model)
	if model.chromeHeight() != 1+model.current.Height() {
		t.Fatalf("chrome = %d", model.chromeHeight())
	}
	if got := len(strings.Split(model.View().Content, "\n")); got != 40 {
		t.Fatalf("screen = %d lines, want 40 after Current landed", got)
	}
	plain := ansi.Strip(model.View().Content)
	if !strings.Contains(plain, "CURRENT") || !strings.Contains(plain, "note  Shared summary only.") {
		t.Fatalf("current strip missing:\n%s", plain)
	}
	if strings.Count(plain, "CURRENT") != 1 {
		t.Fatalf("CURRENT should appear once, not as a parade row:\n%s", plain)
	}
}
