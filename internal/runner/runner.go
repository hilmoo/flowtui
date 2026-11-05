// Package runner executes a flow process and collects what it recorded.
package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hilmoo/flowtui/internal/history"
	"github.com/hilmoo/flowtui/internal/jsonview"
)

// Spec describes one run.
type Spec struct {
	Name        string
	Command     []string
	Dir         string // working directory ("" = current)
	HistoryFile string // path to the JSONL history (relative to Dir or absolute)
}

// Result is what a finished (or cancelled) run produced.
type Result struct {
	Name      string
	Requests  []history.Record
	Warnings  []string
	Log       string
	ExitCode  int
	Duration  time.Duration
	Cancelled bool
	StartErr  error
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// Run starts the command, waits for it to exit (or ctx to be cancelled, which
// kills the whole process tree) and returns the combined output plus the
// history lines appended during the run.
func Run(ctx context.Context, s Spec) Result {
	res := Result{Name: s.Name}
	histPath := resolve(s.Dir, s.HistoryFile)
	offset := history.Offset(histPath)
	start := time.Now()

	if len(s.Command) == 0 {
		res.StartErr = errors.New("empty command")
		res.ExitCode = -1
		res.Log = res.StartErr.Error()
		return res
	}

	var out lockedBuffer
	cmd := exec.Command(s.Command[0], s.Command[1:]...)
	cmd.Dir = s.Dir
	cmd.Stdout = &out
	cmd.Stderr = &out
	configure(cmd)
	// If a killed process left grandchildren holding the pipe, don't hang.
	cmd.WaitDelay = 2 * time.Second

	if err := cmd.Start(); err != nil {
		res.StartErr = err
		res.ExitCode = -1
		res.Log = "failed to start: " + err.Error()
		res.Duration = time.Since(start)
		return res
	}

	done := make(chan struct{})
	var cancelled bool
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			cancelled = true
			killTree(cmd)
		case <-done:
		}
	}()

	err := cmd.Wait()
	close(done)
	wg.Wait()

	res.Duration = time.Since(start)
	res.Cancelled = cancelled
	res.ExitCode = 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = -1
		}
	}
	res.Log = Clean(out.String())
	if err != nil && res.ExitCode == -1 && !cancelled {
		res.Log += "\n" + err.Error()
	}

	recs, warns, herr := history.ReadSince(histPath, offset)
	res.Requests, res.Warnings = recs, warns
	if herr != nil {
		res.Warnings = append(res.Warnings, "could not read history: "+herr.Error())
	}
	return res
}

// Clean strips ANSI escapes and normalises line endings of captured output.
func Clean(s string) string {
	s = jsonview.StripANSI(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n")
}
