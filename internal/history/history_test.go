package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const line1 = `{"time":"2025-01-01T00:00:00Z","method":"GET","url":"http://h/a","status":200,"durationMs":3,"response":{"ok":true}}`
const line2 = `{"time":"2025-01-01T00:00:01Z","method":"POST","url":"http://h/b","request":{"x":1},"status":500,"durationMs":9,"response":"boom"}`

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "none.jsonl")
	if Offset(p) != 0 {
		t.Error("offset of a missing file must be 0")
	}
	recs, warns, err := ReadSince(p, 0)
	if err != nil || len(recs) != 0 || len(warns) != 0 {
		t.Errorf("got %v %v %v", recs, warns, err)
	}
}

func TestOnlyNewLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "h.jsonl")
	write(t, p, line1+"\n")
	off := Offset(p)
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString(line2 + "\n")
	_ = f.Close()

	recs, warns, err := ReadSince(p, off)
	if err != nil || len(warns) != 0 {
		t.Fatalf("err=%v warns=%v", err, warns)
	}
	if len(recs) != 1 || recs[0].Method != "POST" || recs[0].Status != 500 || string(recs[0].Response) != `"boom"` {
		t.Errorf("recs = %+v", recs)
	}
	if string(recs[0].Request) != `{"x":1}` {
		t.Errorf("request = %s", recs[0].Request)
	}
}

func TestMalformedAndBlankLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "h.jsonl")
	write(t, p, line1+"\n\nnot json\n{\"status\":\n"+line2) // last line lacks newline
	recs, warns, _ := ReadSince(p, 0)
	if len(recs) != 2 {
		t.Errorf("want 2 records, got %d", len(recs))
	}
	if len(warns) != 2 || !strings.Contains(warns[0], "malformed") {
		t.Errorf("warnings = %v", warns)
	}
}

func TestTruncatedBetweenRuns(t *testing.T) {
	p := filepath.Join(t.TempDir(), "h.jsonl")
	write(t, p, line1+"\n"+line2+"\n")
	off := Offset(p)
	write(t, p, line2+"\n") // shorter than offset: truncated by someone
	recs, warns, err := ReadSince(p, off)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Method != "POST" {
		t.Errorf("recs = %+v", recs)
	}
	if len(warns) != 1 {
		t.Errorf("want a truncation warning, got %v", warns)
	}
}

func TestNoNewLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "h.jsonl")
	write(t, p, line1+"\n")
	recs, _, _ := ReadSince(p, Offset(p))
	if len(recs) != 0 {
		t.Errorf("got %d records", len(recs))
	}
}
