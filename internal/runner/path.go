package runner

import "path/filepath"

func resolve(dir, p string) string {
	if filepath.IsAbs(p) || dir == "" {
		return p
	}
	return filepath.Join(dir, p)
}
