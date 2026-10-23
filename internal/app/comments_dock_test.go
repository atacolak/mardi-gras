package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
	"github.com/matt-wright86/mardi-gras/internal/views"
)

func testDockModel(t *testing.T) Model {
	t.Helper()
	issue := data.Issue{ID: "j7k.9", Title: "S4-prep", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeTask}
	m := New([]data.Issue{issue}, data.Source{Mode: data.SourceJSONL}, data.DefaultBlockingTypes)
	m.ready = true
	m.width, m.height = 80, 24
	m.layout()
	m.detail.SetIssue(&issue)
	return m
}

func TestCommentsDockCoversWithoutChangingViewport(t *testing.T) {
	m := testDockModel(t)
	before := m.detail.Viewport.Height()
	out := ansi.Strip(m.stampCommentsDock(m.detail.View()))
	if m.detail.Viewport.Height() != before {
		t.Fatalf("viewport height moved %d -> %d", before, m.detail.Viewport.Height())
	}
	if !strings.Contains(out, "COMMENTS") {
		t.Fatalf("dock missing COMMENTS header:\n%s", out)
	}
	if !strings.Contains(out, "No comments on this bead yet") {
		t.Fatalf("empty dock missing empty-state:\n%s", out)
	}
}

func TestCommentsDockFoldTogglesHeight(t *testing.T) {
	m := testDockModel(t)
	open := m.commentsDockHeight()
	m.commentsDockOpen = false
	closed := m.commentsDockHeight()
	if closed != 1 {
		t.Fatalf("collapsed height = %d, want 1", closed)
	}
	if open <= closed {
		t.Fatalf("open height %d should exceed collapsed %d", open, closed)
	}
}

func TestCommentsDockHoverSeesOneComment(t *testing.T) {
	m := testDockModel(t)
	m.detail.SetComments("j7k.9", []gastown.Comment{
		{ID: "63", Author: "midi", Body: "Dispatch"},
		{ID: "64", Author: "midi", Body: "ACK"},
	})
	top := m.commentsDockTopRow()
	m.hoverCommentsDock(2, top+1) // first comment row
	if !m.commentIsSeen("j7k.9", "63") {
		t.Fatal("hovering first comment must mark 63 seen")
	}
	if m.commentIsSeen("j7k.9", "64") {
		t.Fatal("hovering 63 must not clear-or-mark 64")
	}
}

func TestCommentsDockUsesNerdGlyph(t *testing.T) {
	if ui.SymCommentNerd == "" || ui.SymCommentNerd == ui.SymComment {
		t.Fatal("dock must use a nerd-font comment glyph, not the emoji")
	}
	_ = views.Detail{}
}
