// Package concept implements the active Concept EVT1 Stage 0 compiler.
//
// It was extracted from Oct's Concept/Vulkan compiler at R0. Core syntax,
// semantic analysis, MIR, and C11 lowering live here. Vulkan admissions remain
// explicit through `profile Vulkan;` and the profile-specific validation and
// lowering seams; Oct is not a build-time dependency.
package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Span identifies a one-based source position.
type Span struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Token retains the lexeme and its source provenance.
type Token struct {
	Lexeme string
	Span   Span
}

// Diagnostic is a stable compiler error with a source position.
type Diagnostic struct {
	Code    string
	Message string
	Span    Span
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("concept:%d:%d: %s: %s", d.Span.Line, d.Span.Column, d.Code, d.Message)
}

// Outputs is the deterministic generated-artifact set keyed by relative path.
type Outputs map[string][]byte

func digest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Write materializes generated outputs under dir in stable filename order.
func Write(dir string, outputs Outputs) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(outputs))
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := os.WriteFile(filepath.Join(dir, key), outputs[key], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Check verifies that dir contains byte-identical generated outputs.
func Check(dir string, outputs Outputs) error {
	for name, want := range outputs {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("CV3001 stale generated output %s: %w", name, err)
		}
		if string(got) != string(want) {
			return fmt.Errorf("CV3001 stale or hand-edited generated output %s", name)
		}
	}
	return nil
}
