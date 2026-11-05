package concept

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/package/*
var packageTemplates embed.FS

// CompilerLibraryRoot locates the installed library payload, with an explicit
// override and a source-checkout fallback for the Stage-0 development driver.
func CompilerLibraryRoot() string {
	if root := os.Getenv("CONCEPT_LIBRARY_ROOT"); root != "" {
		return root
	}
	var candidates []string
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "libraries"))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(file), "..", "..", "libraries"))
	}
	for _, root := range candidates {
		if _, err := os.Stat(filepath.Join(root, "Standard", "Build", "Metadata.concept")); err == nil {
			return filepath.Clean(root)
		}
	}
	return ""
}

// InitPackage creates a runnable library starter. Existing files are never
// overwritten; exclusive creation also protects against a concurrent initializer.
func InitPackage(root, name, author string) ([]string, error) {
	if !nativeIdentifier(name) {
		return nil, fmt.Errorf("PACKAGE_NAME_INVALID: %q must be an identifier", name)
	}
	if strings.TrimSpace(author) == "" {
		return nil, fmt.Errorf("PACKAGE_AUTHOR_INVALID: author must not be blank")
	}
	data := struct{ Name, QuotedName, QuotedAuthor string }{name, strconv.Quote(name), strconv.Quote(author)}
	files := []struct{ name, source string }{{"manifest.concept", "manifest.concept.tmpl"}, {"src/" + name + ".concept", "Library.concept.tmpl"}, {"tests/library.concept_test", "library.concept_test.tmpl"}}
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.name))
		if _, err := os.Lstat(path); err == nil {
			return nil, fmt.Errorf("PACKAGE_INIT_EXISTS: %s", path)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	var created []string
	for _, file := range files {
		source, err := packageTemplates.ReadFile("templates/package/" + file.source)
		if err != nil {
			return created, err
		}
		tmpl, err := template.New(file.source).Parse(string(source))
		if err != nil {
			return created, err
		}
		var output bytes.Buffer
		if err := tmpl.Execute(&output, data); err != nil {
			return created, err
		}
		path := filepath.Join(root, filepath.FromSlash(file.name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return created, err
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return created, err
		}
		created = append(created, path)
		_, writeErr := out.Write(output.Bytes())
		closeErr := out.Close()
		if writeErr != nil {
			return created, writeErr
		}
		if closeErr != nil {
			return created, closeErr
		}
	}
	return created, nil
}
