package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubstitutionFailureIsAnError(t *testing.T) {
	cases := map[string]string{
		"dependent_void_array":          "CV4573",
		"missing_member":                "CV4172",
		"non_type_bool_argument":        "GENERIC_NON_TYPE_ARGUMENT_INVALID",
		"required_operation_absent":     "CV4153",
		"ambiguous_operation":           "CV4182",
		"constraint_excludes_candidate": "CV4153",
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "tooling", "substitution-failure", "invalid", name+".concept")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(filepath.ToSlash(path), string(body))
			if err == nil || !strings.Contains(err.Error(), code) {
				t.Fatalf("substitution failure must remain a diagnostic containing %s, got %v", code, err)
			}
		})
	}
}
