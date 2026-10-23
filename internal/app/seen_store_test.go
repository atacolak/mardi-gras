package app

import (
	"testing"

	"github.com/matt-wright86/mardi-gras/internal/data"
)

func TestSeenCommentsSurviveReload(t *testing.T) {
	dir := t.TempDir()
	seenStoreDir = dir
	t.Cleanup(func() { seenStoreDir = "" })

	m := New([]data.Issue{{ID: "a", Title: "t", Status: data.StatusOpen}}, data.Source{ProjectDir: dir}, data.DefaultBlockingTypes)
	m.markCommentSeen("epic-1", "21")
	if !m.commentIsSeen("epic-1", "21") {
		t.Fatal("just marked 21")
	}

	m2 := New([]data.Issue{{ID: "a", Title: "t", Status: data.StatusOpen}}, data.Source{ProjectDir: dir}, data.DefaultBlockingTypes)
	if !m2.commentIsSeen("epic-1", "21") {
		t.Fatal("seen cursor must survive a new Model / mg boot")
	}
	if m2.commentIsSeen("epic-1", "22") {
		t.Fatal("unmarked comment must stay unread")
	}
}
