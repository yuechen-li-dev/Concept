package concept

import (
	"strings"
	"testing"
)

func TestTwoGeneratorsReportBothConflictOrigins(t *testing.T) {
	source := `module GeneratedConflict; profile Core;
struct Item { int value; }
generator <typename T> First
int Read(ref const T item) { return item.value; }
generator <typename T> Second
int Read(ref const T item) { return item.value; }
derive First reflect<Item>;
derive Second reflect<Item>;
`
	_, err := Parse("two_generators.concept", source)
	if err == nil || !strings.Contains(err.Error(), "GENERATOR_DECLARATION_CONFLICT") || !strings.Contains(err.Error(), "First") || !strings.Contains(err.Error(), "Second") {
		t.Fatalf("missing both generator origins: %v", err)
	}
}
