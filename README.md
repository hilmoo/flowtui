# flowtui

A terminal app (Bubble Tea v2) that lists runnable **flow** scripts, runs the
one you pick, then shows every HTTP request/response that flow made in a
scrollable viewer. It is not a test framework: no assertions, just run and
inspect.

flowtui does not care how flows are written. It runs a command and reads the
requests the flow appended to a JSONL file.

## Install

```sh
go install github.com/hilmoo/flowtui/cmd/flowtui@latest   # or, from a checkout:
task build                              # -> bin/flowtui
```

Requires Go 1.22+ (developed on 1.26). Works on Linux, macOS and Windows
(on Windows, Ctrl+C kills the child tree with `taskkill /T`).

Try the bundled sample project (two shell-script flows plus a nested one and
an extra action; needs `sh` on PATH):

```sh
task run     # runs flowtui inside testdata/sample
```

Other tasks: `task test`, `task lint` (golangci-lint), `task build`.

`flowtui --version` prints the release version (`dev` for local builds).

## Config: `flowtui.json`

Read from the working directory. If missing, a default file is created.
Fields that are absent keep their default value.

| Field          | Meaning |
|----------------|---------|
| `flowsDir`     | Directory (relative to the working dir) searched for flows. A missing directory just means "no flows". |
| `glob`         | Pattern relative to `flowsDir`; `**` is supported (doublestar). The literal tail after the last wildcard (e.g. `.flow.ts`) is stripped from the display name. |
| `command`      | Argv used to run a flow. `{file}` is replaced by the flow path (`flowsDir/<match>`). |
| `historyFile`  | JSONL file the flows append requests to (see below). |
| `extraActions` | List of `{ "label", "command" }`. Shown first in the picker, run without a file. |

```json
{
  "flowsDir": "flows",
  "glob": "**/*.flow.ts",
  "command": ["pnpm", "exec", "tsx", "{file}"],
  "historyFile": "out/history.jsonl",
  "extraActions": [
    { "label": "[regenerate client]", "command": ["pnpm", "exec", "orval"] }
  ]
}
```

Flow names are the path relative to `flowsDir` with the glob suffix removed,
e.g. `flows/auth/login.flow.ts` -> `auth/login`. Subfolders are allowed.

## The history contract (any language can implement it)

A flow records a request by appending **one JSON object per line** to
`historyFile`:

```json
{"time":"2025-01-01T10:00:00Z","method":"POST","url":"http://host/api/x","request":{"a":1},"status":200,"durationMs":12,"response":{"ok":true}}
```

| Field        | Type | Notes |
|--------------|------|-------|
| `time`       | string | RFC 3339 timestamp. Shown as local `HH:MM:SS`; shown verbatim if unparsable. |
| `method`     | string | HTTP method. |
| `url`       | string | Full URL. The path is the title; the last path segment is the row label. |
| `request`    | any, optional | Request body. Absent = "(none)". |
| `status`     | number | HTTP status. `0`/absent is shown as `---`. |
| `durationMs` | number | Duration in milliseconds. |
| `response`   | any or string, optional | A JSON value, or a plain string (shown as text; a string that itself holds JSON is pretty-printed). |

How runs are matched to lines: before starting the process flowtui records the
**byte size** of `historyFile` (0 if it does not exist); after the process
exits it parses only what was appended after that offset.

- Missing file: no requests, no error.
- Malformed or blank-JSON lines are skipped and reported as warnings (shown
  at the top of the console output and as `⚠ n warning(s)` in the header).
- If the file became *smaller* than the recorded offset (truncated or rotated
  during the run) it is read from the start, with a warning.
- Limitation: a file truncated and regrown past the old offset during one run
  cannot be detected.

Minimal shell example (see `testdata/sample/flows/hello.flow.sh`):

```sh
mkdir -p out
echo '{"time":"'"$(date -u +%FT%TZ)"'","method":"GET","url":"http://h/api/Ping","status":200,"durationMs":3,"response":{"pong":true}}' >> out/history.jsonl
```

stdout and stderr of the child are captured combined (ANSI escapes stripped)
and shown as the **console output** row, together with the exit code and
duration.

## Screens and keys

**Picker**: type to filter (case-insensitive substring), `Backspace` edits the
filter, `↑/↓` or the mouse wheel select, `Enter` runs, `Esc`/`Ctrl+C` quit. The
right pane previews the source with line numbers (first 64 KB, long lines are
cut at the pane edge). While a flow runs, input is ignored except `Ctrl+C`,
which kills the child process tree and returns to the picker with a
"cancelled" notice (a second `Ctrl+C` quits flowtui).

**Results**: one row per request (`status METHOD-3-chars last-path-segment`,
for example `201 POS ProjectCreate`) and a final `console output` row. Status
colors: 2xx green, 3xx and others yellow, 4xx red, 5xx magenta. The last
request is selected after a successful run, `console output` after a failure.
Extra actions and flows that record nothing show only `console output`.

| Key | Action |
|-----|--------|
| `↑/↓`, `j/k` | move between requests (list focused) / scroll (detail focused) |
| `Tab`, `←/→` | switch focus (`←` list, `→` detail) |
| `PgUp/PgDn/Space`, `Ctrl+u/Ctrl+d` | page / half page the detail pane |
| `g` / `G` | top / bottom of the detail pane |
| mouse wheel | always scrolls the detail pane |
| `t` | toggle full values (strings > 200 chars are previewed as `… (N chars)`) |
| `r` | re-run the same flow |
| `y` | copy the selected response as pretty JSON (OSC 52, works over SSH) |
| `e` | write the selected record to `out/last.json` (relative to the working dir) |
| `Esc`, `Backspace` | back to the picker |
| `Ctrl+C` | quit |

Long lines wrap by visible width (ANSI and wide-character aware). The header
shows flow name, request count and the visible line range (`lines 12-20/58`).

## Responsive layout

- Width >= 80: two panes, left pane about 33% (min 24, max 44).
- Width < 80, Results: one pane at a time, the focused one, so the detail gets
  the full width; `Tab`/`←`/`→` toggle. After a run the detail pane is focused.
- Width < 80, Picker (assumption): the list only; the source preview is hidden.

Mouse (cell-motion) and the alternate screen are enabled, and restored on
exit. Because the mouse is captured, **hold Shift while dragging** (Option on
some macOS terminals) to select text.

## Layout

```
cmd/flowtui          main (wiring only)
internal/config      flowtui.json
internal/discover    glob + naming + command expansion
internal/runner      process execution, process-tree kill (unix/windows)
internal/history     JSONL parsing by offset
internal/jsonview    colored JSON, truncation, ANSI-aware wrapping
internal/ui          Bubble Tea models (root, picker, results)
testdata/sample      sample project for `task run`
```

## Assumptions and decisions

- Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 (`charm.land/...`) and
  `teatest/v2`.
- The filter is a plain string handled by the picker (no `textinput`); fuzzy
  matching was not added.
- Request row: `status`, method cut to 3 chars, last path segment; missing
  status prints `---` in gray.
- Ctrl+C during a run returns to the picker rather than showing partial
  results.
- Children run with the working directory as cwd and inherit the environment.
- The history offset is in bytes, not lines.
- A flow that fails to start is reported as exit code -1 with the error in the
  console output.

## Releases

Pushing a tag that starts with `v` builds and publishes a GitHub release
(`.github/workflows/release.yml`, configured by `.goreleaser.yaml`):

```sh
git tag v0.1.0
git push origin v0.1.0
```

Archives are produced for linux/darwin/windows on amd64 and arm64, with a
`checksums.txt`. CI (`.github/workflows/ci.yml`) runs vet, tests on all three
OSes, golangci-lint and `goreleaser check` on every push to `main` and on pull
requests.

## License

MIT, see [LICENSE](LICENSE).
