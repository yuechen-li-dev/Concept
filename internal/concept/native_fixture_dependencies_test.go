package concept

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var nativeCompilerIdentities sync.Map

func nativeCompilerIdentity(compiler string) string {
	if value, ok := nativeCompilerIdentities.Load(compiler); ok {
		return value.(string)
	}
	// The executable and target are immutable for one qualification process.
	// Source/bridge/configuration identity is the exact preprocessed-input set.
	body, err := os.ReadFile(compiler)
	if err != nil {
		return compiler
	}
	value := fmt.Sprintf("%s:%s/%s:%x", compiler, runtime.GOOS, runtime.GOARCH, sha256.Sum256(body))
	nativeCompilerIdentities.Store(compiler, value)
	return value
}

// Conservative include closure for generated C. Literal includes are followed
// recursively, including headers that shadow system names. Unknown directives,
// macro includes or continuations disable reuse rather than guessing.
func nativeFixtureDependencies(outputs Outputs, source string) ([]string, bool) {
	lookup := map[string]string{}
	for name := range outputs {
		key := strings.ToLower(path.Clean(filepath.ToSlash(name)))
		if prior, ok := lookup[key]; ok && prior != name {
			return nil, false
		}
		lookup[key] = name
	}
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(name string) bool {
		if seen[name] {
			return true
		}
		seen[name] = true
		text := strings.ReplaceAll(string(outputs[name]), "\r\n", "\n")
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			at := strings.Index(trimmed, "#")
			if at < 0 {
				continue
			}
			directive := strings.TrimSpace(trimmed[at+1:])
			if strings.Contains(directive, "/*") {
				return false
			}
			if strings.HasSuffix(trimmed, "\\") {
				return false
			}
			if !strings.HasPrefix(directive, "include") {
				words := strings.Fields(directive)
				if len(words) == 0 {
					continue
				}
				switch words[0] {
				case "define", "undef", "if", "ifdef", "ifndef", "elif", "else", "endif", "pragma", "line", "error", "warning":
					continue
				}
				return false
			}
			if at != 0 || !strings.HasPrefix(directive, "include ") && !strings.HasPrefix(directive, "include\t") {
				return false
			}
			rest := strings.TrimSpace(strings.TrimPrefix(directive, "include"))
			if len(rest) < 3 || rest[0] != '"' && rest[0] != '<' {
				return false
			}
			close := byte('"')
			if rest[0] == '<' {
				close = '>'
			}
			end := strings.IndexByte(rest[1:], close)
			if end < 0 {
				return false
			}
			include := rest[1 : 1+end]
			candidate := path.Clean(path.Join(path.Dir(filepath.ToSlash(name)), include))
			child, ok := lookup[strings.ToLower(candidate)]
			if !ok {
				child, ok = lookup[strings.ToLower(path.Clean(include))]
			}
			if !ok {
				if rest[0] == '"' {
					return false
				}
				continue
			}
			if !visit(child) {
				return false
			}
		}
		return true
	}
	if !visit(source) {
		return nil, false
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, true
}
func nativeFixtureKey(outputs Outputs, source, compiler string, flags []string) (string, bool) {
	for _, flag := range flags {
		if strings.HasPrefix(flag, "-I") || strings.HasPrefix(flag, "-include") || strings.HasPrefix(flag, "-imacros") || strings.HasPrefix(flag, "-iquote") || strings.HasPrefix(flag, "--sysroot") {
			return "", false
		}
	}
	names, cacheable := nativeFixtureDependencies(outputs, source)
	if !cacheable {
		return "", false
	}
	h := sha256.New()
	fmt.Fprintf(h, "%q %q %q", nativeCompilerIdentity(compiler), flags, source)
	for _, name := range names {
		fmt.Fprintf(h, "%d:%s%d:", len(name), name, len(outputs[name]))
		h.Write(outputs[name])
	}
	return fmt.Sprintf("%x", h.Sum(nil)), true
}
