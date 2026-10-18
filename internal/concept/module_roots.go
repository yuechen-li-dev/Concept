package concept

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// A module's trailing namespace components may share a filename, e.g.
// Library/AMD64/Encoding.concept or Library/AMD64.Encoding.concept.
// Enumerate every layout so an alias never silently shadows another source.
func evt1ModuleSourceMatches(roots []string, name string) []string {
	parts := strings.Split(name, ".")
	seen := map[string]bool{}
	var matches []string
	for _, root := range roots {
		for split := len(parts) - 1; split >= 0; split-- {
			segments := append([]string{root}, parts[:split]...)
			segments = append(segments, strings.Join(parts[split:], ".")+".concept")
			candidate := filepath.Clean(filepath.Join(segments...))
			key := candidate
			if runtime.GOOS == "windows" {
				key = strings.ToLower(key)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				matches = append(matches, candidate)
			}
		}
	}
	sort.Strings(matches)
	return matches
}

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
