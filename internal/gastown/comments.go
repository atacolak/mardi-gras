package gastown

import (
	"encoding/json"
	"fmt"
)

// Comment represents a single comment on a beads issue.
type Comment struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Body   string `json:"text"`
	Time   string `json:"created_at"`
}

func (c *Comment) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID     json.RawMessage `json:"id"`
		Author string          `json:"author"`
		Text   string          `json:"text"`
		Time   string          `json:"created_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	id, err := parseFlexibleID(raw.ID)
	if err != nil {
		return err
	}
	c.ID = id
	c.Author = raw.Author
	c.Body = raw.Text
	c.Time = raw.Time
	return nil
}

// parseFlexibleID accepts JSON string or number so br integer ids (`63`)
// and historical string ids (`cmt-001`) both land as a string key.
func parseFlexibleID(data []byte) (string, error) {
	if len(data) == 0 || string(data) == "null" {
		return "", nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return "", err
		}
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return "", fmt.Errorf("comment id: %w", err)
	}
	return n.String(), nil
}

// FetchComments runs `br comments <issueID> --json` and parses the output.
func FetchComments(issueID string) ([]Comment, error) {
	out, err := runWithTimeout(timeoutMedium, "br", "comments", issueID, "--json")
	if err != nil {
		return nil, fmt.Errorf("br comments: %w", err)
	}
	var comments []Comment
	if err := json.Unmarshal(out, &comments); err != nil {
		return nil, fmt.Errorf("br comments parse: %w", err)
	}
	return comments, nil
}
