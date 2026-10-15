package concept

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Explicit roots denote physical directories, not a priority search path.
// Canonicalize aliases before conflict detection in source and artifact loaders.
func evt1NormalizeModuleRoots(roots []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, root := range roots {
		if absolute, err := filepath.Abs(root); err == nil {
			root = absolute
		}
		root = filepath.Clean(root)
		if physical, err := filepath.EvalSymlinks(root); err == nil {
			root = physical
		}
		key := root
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, root)
		}
	}
	sort.Strings(result)
	return result
}
