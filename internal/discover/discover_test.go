package discover

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hilmoo/flowtui/internal/config"
)

func TestSuffix(t *testing.T) {
	tests := []struct{ glob, want string }{
		{"**/*.flow.ts", ".flow.ts"},
		{"*.flow.sh", ".flow.sh"},
		{"**/*_flow.go", "_flow.go"},
		{"a/b.sh", ".sh"},
		{"**/*", ""},
	}
	for _, tt := range tests {
		if got := Suffix(tt.glob); got != tt.want {
			t.Errorf("Suffix(%q) = %q, want %q", tt.glob, got, tt.want)
		}
	}
}

func TestNameAndExpand(t *testing.T) {
	if got := Name("sub/dir/a.flow.ts", "**/*.flow.ts"); got != "sub/dir/a" {
		t.Errorf("Name = %q", got)
	}
	if got := Name(`sub\a.flow.ts`, "**/*.flow.ts"); got != "sub/a" && got != `sub\a` {
		t.Errorf("Name = %q", got)
	}
	got := Expand([]string{"pnpm", "exec", "tsx", "{file}"}, "flows/a.flow.ts")
	want := []string{"pnpm", "exec", "tsx", "flows/a.flow.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand = %v", got)
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"flows/b.flow.sh", "flows/a.flow.sh", "flows/sub/deep/c.flow.sh", "flows/ignored.txt"} {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{
		FlowsDir: "flows", Glob: "**/*.flow.sh", Command: []string{"sh", "{file}"}, HistoryFile: "h",
		ExtraActions: []config.ExtraAction{{Label: "[gen]", Command: []string{"gen"}}},
	}
	items, err := Discover(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range items {
		names = append(names, it.Name)
	}
	want := []string{"[gen]", "a", "b", "sub/deep/c"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if !items[0].IsExtra() || items[1].IsExtra() {
		t.Error("extra action must be first and have no file")
	}
	if items[1].Command[0] != "sh" || items[1].Command[1] != filepath.Join("flows", "a.flow.sh") {
		t.Errorf("command = %v", items[1].Command)
	}
}

func TestDiscoverMissingDir(t *testing.T) {
	cfg := config.Default()
	items, err := Discover(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != len(cfg.ExtraActions) {
		t.Errorf("want only extra actions, got %d items", len(items))
	}
}
