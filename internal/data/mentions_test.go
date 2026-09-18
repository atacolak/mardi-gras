package data

import "testing"

func TestFindLoadedMentionsLongestWins(t *testing.T) {
	parent := &Issue{ID: "mard-wte"}
	child := &Issue{ID: "mard-wte.7"}
	known := map[string]*Issue{"mard-wte": parent, "mard-wte.7": child}
	got := FindLoadedMentions("see mard-wte.7 and mard-wte", known)
	if len(got) != 2 {
		t.Fatalf("mentions = %#v, want 2", got)
	}
	if got[0].ID != "mard-wte.7" || got[1].ID != "mard-wte" {
		t.Fatalf("order = %s then %s, want child then parent", got[0].ID, got[1].ID)
	}
}

func TestFindLoadedMentionsSkipsUnknownAndSubstrings(t *testing.T) {
	known := map[string]*Issue{"mard-wte": {ID: "mard-wte"}}
	got := FindLoadedMentions("mard-wte.7 and v0-32 and mard-wte here", known)
	if len(got) != 1 || got[0].ID != "mard-wte" {
		t.Fatalf("mentions = %#v, want only the standalone mard-wte", got)
	}
	if got[0].Start != 25 {
		t.Fatalf("start = %d, want 25 (the token after 'and ')", got[0].Start)
	}
}

func TestFindLoadedMentionsEmpty(t *testing.T) {
	if got := FindLoadedMentions("mard-wte", nil); got != nil {
		t.Fatalf("nil map: got %#v", got)
	}
	if got := FindLoadedMentions("", map[string]*Issue{"a-b": {}}); got != nil {
		t.Fatalf("empty text: got %#v", got)
	}
}
