package gastown

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCommentParsing(t *testing.T) {
	raw := `[
		{"id":"c-1","author":"claude (Toast)","text":"JWT validation needs refresh token handling","created_at":"2025-02-22T10:30:00Z"},
		{"id":"c-2","author":"overseer","text":"Approved, ship it","created_at":"2025-02-22T11:15:00Z"}
	]`

	var comments []Comment
	if err := json.Unmarshal([]byte(raw), &comments); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	if comments[0].Author != "claude (Toast)" {
		t.Errorf("expected author 'claude (Toast)', got %q", comments[0].Author)
	}
	if comments[0].Body != "JWT validation needs refresh token handling" {
		t.Errorf("unexpected body: %q", comments[0].Body)
	}
	if comments[1].ID != "c-2" {
		t.Errorf("expected ID 'c-2', got %q", comments[1].ID)
	}
	if comments[1].Time != "2025-02-22T11:15:00Z" {
		t.Errorf("expected created_at time, got %q", comments[1].Time)
	}
}

func TestCommentIDAcceptsIntegerJSON(t *testing.T) {
	// br 0.5.11 emits integer comment ids (`63`). A string-only field
	// rejects that payload, so the dock would stay empty on a live board.
	raw := `[{"id":63,"author":"sf","text":"Dispatch flip_usb","created_at":"2026-10-23T04:46:51Z"}]`
	var comments []Comment
	if err := json.Unmarshal([]byte(raw), &comments); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].ID != "63" {
		t.Errorf("ID = %q, want %q", comments[0].ID, "63")
	}
	if comments[0].Author != "sf" {
		t.Errorf("Author = %q, want sf", comments[0].Author)
	}
	if comments[0].Body != "Dispatch flip_usb" {
		t.Errorf("Body = %q, want Dispatch flip_usb", comments[0].Body)
	}
}


func TestCommentParsingEmpty(t *testing.T) {
	raw := `[]`
	var comments []Comment
	if err := json.Unmarshal([]byte(raw), &comments); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments, got %d", len(comments))
	}
}

func TestFetchCommentsHappy(t *testing.T) {
	defer mockRun([]byte(gtCommentsJSON), nil)()
	comments, err := FetchComments("mg-10")
	if err != nil {
		t.Fatalf("FetchComments() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].Author != "alice" {
		t.Errorf("Author = %q, want alice", comments[0].Author)
	}
	// bd serializes the comment body as "text", not "body". The struct tag
	// pointed at "body" for a long time and nothing caught it because this
	// test only checked Author, so the body silently parsed as empty.
	if comments[0].Body != "Starting work on this." {
		t.Errorf("Body = %q, want %q", comments[0].Body, "Starting work on this.")
	}
}

func TestFetchCommentsExecError(t *testing.T) {
	defer mockRun(nil, errors.New("bd not found"))()
	_, err := FetchComments("mg-10")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchCommentsArgs(t *testing.T) {
	calls, restore := mockRunCapture([]byte(`[]`), nil)
	defer restore()
	_, _ = FetchComments("mg-42")
	args := (*calls)[0]
	if len(args) != 4 || args[0] != "br" || args[1] != "comments" || args[2] != "mg-42" || args[3] != "--json" {
		t.Errorf("args = %v", args)
	}
}

func TestFetchCommentsUsesBr(t *testing.T) {
	calls, restore := mockRunCapture([]byte(`[]`), nil)
	defer restore()
	_, _ = FetchComments("j7k.9")
	if len(*calls) != 1 {
		t.Fatalf("made %d calls, want 1", len(*calls))
	}
	args := (*calls)[0]
	if len(args) != 4 || args[0] != "br" || args[1] != "comments" || args[2] != "j7k.9" || args[3] != "--json" {
		t.Errorf("args = %v, want [br comments j7k.9 --json]", args)
	}
}


func TestCommentParsingIgnoresBodyKey(t *testing.T) {
	// A "body" key is not what bd emits; it must not populate Body. This pins
	// the field name so the tag cannot quietly drift back.
	raw := `[{"id":"c-1","author":"alice","body":"wrong key","created_at":"2025-02-22T10:30:00Z"}]`
	var comments []Comment
	if err := json.Unmarshal([]byte(raw), &comments); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if comments[0].Body != "" {
		t.Errorf("Body = %q, want empty: only the \"text\" key carries the comment body", comments[0].Body)
	}
}
