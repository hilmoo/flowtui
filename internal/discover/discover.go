// Package discover finds runnable flows and the extra actions.
package discover

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/hilmoo/flowtui/internal/config"
)

// Item is one entry of the picker.
type Item struct {
	Name    string   // display name
	Path    string   // flow file path (empty for extra actions)
	Command []string // fully expanded command
}

// IsExtra reports whether the item is an extra action (no file).
func (i Item) IsExtra() bool { return i.Path == "" }

// Suffix returns the literal tail of a glob after its last wildcard, e.g.
// "**/*.flow.ts" -> ".flow.ts". It is stripped from flow names.
func Suffix(glob string) string {
	i := strings.LastIndexAny(glob, "*?]}")
	if i < 0 {
		return path.Ext(glob)
	}
	return glob[i+1:]
}

// Name builds the display name for a match relative to flowsDir.
func Name(rel, glob string) string {
	rel = filepath.ToSlash(rel)
	if s := Suffix(glob); s != "" {
		rel = strings.TrimSuffix(rel, s)
	}
	return rel
}

// Expand replaces {file} in every command element.
func Expand(command []string, file string) []string {
	out := make([]string, len(command))
	for i, c := range command {
		out[i] = strings.ReplaceAll(c, "{file}", file)
	}
	return out
}

// Discover returns the extra actions followed by the sorted flows found
// under root/flowsDir. A missing flowsDir yields only the extra actions.
func Discover(cfg config.Config, root string) ([]Item, error) {
	var items []Item
	for _, a := range cfg.ExtraActions {
		items = append(items, Item{Name: a.Label, Command: a.Command})
	}
	dir := filepath.Join(root, cfg.FlowsDir)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return items, nil
		}
		return items, err
	}
	matches, err := doublestar.Glob(os.DirFS(dir), cfg.Glob, doublestar.WithFilesOnly())
	if err != nil {
		return items, err
	}
	sort.Strings(matches)
	for _, m := range matches {
		if d, err := fs.Stat(os.DirFS(dir), m); err != nil || d.IsDir() {
			continue
		}
		file := filepath.Join(cfg.FlowsDir, filepath.FromSlash(m))
		items = append(items, Item{
			Name:    Name(m, cfg.Glob),
			Path:    file,
			Command: Expand(cfg.Command, file),
		})
	}
	return items, nil
}
