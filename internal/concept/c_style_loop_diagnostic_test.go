package concept

import (
	"strings"
	"testing"
)

func TestCStyleForDiagnosticTeachesRange(t *testing.T) {
	source := "profile Core; int Run() { for (int i = 0; i < 8; i++) { } return 0; }"
	_, err := Parse("first-draft.concept", source)
	if err == nil || !strings.Contains(err.Error(), "FOREACH_ITERATOR_INVALID") ||
		!strings.Contains(err.Error(), "for (i in start..end step n)") {
		t.Fatalf("C-style for diagnostic should explain the Concept range spelling, got %v", err)
	}
}
