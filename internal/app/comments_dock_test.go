package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
	"github.com/matt-wright86/mardi-gras/internal/views"
)

func testDockModel(t *testing.T) Model {
	t.Helper()
	seenStoreDir = t.TempDir()
	t.Cleanup(func() { seenStoreDir = "" })
	issue := data.Issue{ID: "j7k.9", Title: "S4-prep", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeTask}
	m := New([]data.Issue{issue}, data.Source{Mode: data.SourceJSONL, ProjectDir: t.TempDir()}, data.DefaultBlockingTypes)
	m.seenCommentIDs = map[string]map[string]bool{}
	m.ready = true
	m.width, m.height = 80, 24
	m.layout()
	m.detail.SetIssue(&issue)
	return m
}

func TestDockInsetLeavesBodyScrollable(t *testing.T) {
	m := testDockModel(t)
	m.applyDockInset()
	dock := m.commentsDockHeight()
	vis := m.detail.Viewport.Height()
	if vis+dock > m.detail.Height {
		t.Fatalf("viewport %d + dock %d exceeds pane %d", vis, dock, m.detail.Height)
	}
	if vis >= m.detail.Height-1 {
		t.Fatal("open dock must shrink the body viewport so covered lines can scroll up")
	}
}

func TestCommentsDockCoversWithoutChangingViewport(t *testing.T) {
	m := testDockModel(t)
	before := m.detail.Viewport.Height()
	out := ansi.Strip(m.stampCommentsDock(m.detail.View()))
	if m.detail.Viewport.Height() != before {
		t.Fatalf("viewport height moved %d -> %d", before, m.detail.Viewport.Height())
	}
	if strings.Contains(out, "COMMENTS") {
		t.Fatalf("header still shouts COMMENTS:\n%s", out)
	}
	if !strings.Contains(out, "comments") {
		t.Fatalf("dock missing comments count:\n%s", out)
	}
	if strings.Contains(out, "fold") {
		t.Fatalf("fold label must be gone:\n%s", out)
	}
	if !strings.Contains(out, "no comments yet") {
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

func TestCommentsDockHoverDoesNotMarkSeen(t *testing.T) {
	m := testDockModel(t)
	m.detail.SetComments("j7k.9", []gastown.Comment{
		{ID: "63", Author: "midi", Body: "Dispatch"},
		{ID: "64", Author: "midi", Body: "ACK"},
	})
	paint := m.paintCommentsDock(m.detail.Width, m.commentsDockHeight())
	row := -1
	for i, hit := range paint.hits {
		if hit == 0 {
			row = m.commentsDockTopRow() + i
			break
		}
	}
	if row < 0 {
		t.Fatalf("no first comment row in hits %v", paint.hits)
	}
	m.hoverCommentsDock(2, row)
	if m.commentIsSeen("j7k.9", "63") {
		t.Fatal("hover must not mark comments read")
	}
}

func TestCommentsDockClickMarksSeen(t *testing.T) {
	m := testDockModel(t)
	m.detail.SetComments("j7k.9", []gastown.Comment{
		{ID: "63", Author: "midi", Body: "Dispatch"},
	})
	paint := m.paintCommentsDock(m.detail.Width, m.commentsDockHeight())
	row := -1
	for i, hit := range paint.hits {
		if hit == 0 {
			row = m.commentsDockTopRow() + i
			break
		}
	}
	if row < 0 {
		t.Fatal("no thread row")
	}
	if !m.selectCommentAt(row) {
		t.Fatal("click must select")
	}
	if !m.commentIsSeen("j7k.9", "63") {
		t.Fatal("click must mark read")
	}
}

func TestCommentsDockUsesNerdGlyph(t *testing.T) {
	if ui.SymCommentNerd == "" || ui.SymCommentNerd == ui.SymComment {
		t.Fatal("dock must use a nerd-font comment glyph, not the emoji")
	}
	_ = views.Detail{}
}

func TestFormatCommentBodyBreaksHcom(t *testing.T) {
	got := formatCommentBody("testing add comment by ata.</hcom>")
	if !strings.Contains(got, "ata.\n</hcom>") {
		t.Fatalf("expected </hcom> on its own line, got %q", got)
	}
}

func TestEncodeSplitReply(t *testing.T) {
	body := encodeReply("21", "testing a reply")
	c := gastown.Comment{ID: "22", Body: body}
	parent, rest := splitReply(c)
	if parent != "21" || rest != "testing a reply" {
		t.Fatalf("parent=%q rest=%q", parent, rest)
	}
}

func TestCommentThreadsNestReplies(t *testing.T) {
	m := testDockModel(t)
	m.detail.SetComments("j7k.9", []gastown.Comment{
		{ID: "21", Author: "ata", Body: "root"},
		{ID: "22", Author: "zime", Body: encodeReply("21", "child")},
	})
	threads := m.commentThreads()
	if len(threads) != 1 {
		t.Fatalf("threads=%d want 1", len(threads))
	}
	if threads[0].root.ID != "21" || len(threads[0].replies) != 1 || threads[0].replies[0].ID != "22" {
		t.Fatalf("bad nest: %+v", threads)
	}
}

func TestAuthorColorAtaDistinct(t *testing.T) {
	ata := fmt.Sprintf("%v", authorColor("ata", false))
	midi := fmt.Sprintf("%v", authorColor("midi", false))
	zime := fmt.Sprintf("%v", authorColor("zime", false))
	if ata == midi || midi == zime || ata == zime {
		t.Fatalf("authors must not share a color: ata=%s midi=%s zime=%s", ata, midi, zime)
	}
}

func TestDockKeepsCommentsWhenIDCleared(t *testing.T) {
	m := testDockModel(t)
	m.detail.SetComments("j7k.9", []gastown.Comment{{ID: "1", Author: "ata", Body: "hi"}})
	m.detail.CommentsIssueID = ""
	if n := len(m.dockComments()); n != 1 {
		t.Fatalf("cleared id must still show cached comments, got %d", n)
	}
}

func TestThreadIndexFor(t *testing.T) {
	threads := []commentThread{
		{root: gastown.Comment{ID: "a"}},
		{root: gastown.Comment{ID: "b"}, replies: []gastown.Comment{{ID: "c"}}},
	}
	i, ok := threadIndexFor(threads, "c")
	if !ok || i != 1 {
		t.Fatalf("reply parent i=%d ok=%v", i, ok)
	}
}

func TestKeyCTogglesCommentsDock(t *testing.T) {
	m := testDockModel(t)
	if !m.commentsDockOpen {
		t.Fatal("dock starts open")
	}
	result, _ := m.handleKey(tea.KeyPressMsg{Code: 'C', Text: "C"})
	got := result.(Model)
	if got.commentsDockOpen {
		t.Fatal("C should fold the comments dock")
	}
	result, _ = got.handleKey(tea.KeyPressMsg{Code: 'C', Text: "C"})
	got = result.(Model)
	if !got.commentsDockOpen {
		t.Fatal("C again should unfold")
	}
}

func TestAfterSelectFetchesComments(t *testing.T) {
	m := setupModel(t)
	m.detail.CommentsIssueID = ""
	m.detail.Comments = nil
	if cmd := m.afterSelect(); cmd == nil {
		t.Fatal("selecting a bead must fetch comments without a body click")
	}
}
