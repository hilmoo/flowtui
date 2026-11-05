package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hilmoo/flowtui/internal/history"
	"github.com/hilmoo/flowtui/internal/jsonview"
	"github.com/hilmoo/flowtui/internal/runner"
)

func TestExpandTruncated(t *testing.T) {
	long := strings.Repeat("abcdefghij", 30)
	raw, _ := json.Marshal(map[string]string{"image": long})
	res := runner.Result{Requests: []history.Record{{Method: "POST", URL: "http://x/a", Response: raw}}}
	shown := jsonview.StripANSI(DetailContent(res, 0, false, 30)) // narrow: marker gets wrapped
	got := expandTruncated(shown, res, 0)
	if !strings.Contains(got, long) {
		t.Errorf("full string not restored: %q", got)
	}
}
