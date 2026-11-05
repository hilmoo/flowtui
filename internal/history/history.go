// Package history reads the request records flows append to a JSONL file.
package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Record is one recorded HTTP exchange.
type Record struct {
	Time       string          `json:"time"`
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	Request    json.RawMessage `json:"request,omitempty"`
	Status     int             `json:"status"`
	DurationMs float64         `json:"durationMs"`
	Response   json.RawMessage `json:"response,omitempty"`
}

// Offset returns the current size of the file in bytes; a missing file is 0.
func Offset(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// ReadSince parses the lines appended after offset. A missing file yields no
// records. If the file is now smaller than offset (truncated or rotated
// between runs) everything is read from the start. Malformed lines are
// skipped and reported in warnings.
func ReadSince(path string, offset int64) (recs []Record, warnings []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if offset < 0 || offset > int64(len(data)) {
		offset = 0
		if len(data) > 0 {
			warnings = append(warnings, "history file shrank during the run; reading it from the start")
		}
	}
	recs, w := Parse(data[offset:])
	return recs, append(warnings, w...), nil
}

// Parse decodes JSONL content. Blank lines are ignored.
func Parse(data []byte) (recs []Record, warnings []string) {
	for i, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped malformed history line %d: %s", i+1, shorten(err.Error())))
			continue
		}
		recs = append(recs, r)
	}
	return recs, warnings
}

func shorten(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return s
}
