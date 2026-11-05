package concept

import (
	"os"
	"testing"
)

func TestRecordUpdateNestedFailureStrictC11(t *testing.T) {
	path := "../../tests/runtime/record_failure.concept_test"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "record_failure.generated.c")
}

func TestBorrowedOwnedMatchStrictC11(t *testing.T) {
	path := "../../tests/runtime/borrowed_owned_match.concept_test"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "borrowed_owned_match.generated.c")
}
