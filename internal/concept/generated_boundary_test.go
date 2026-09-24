package concept

import (
	"strings"
	"testing"
)

func TestGeneratedPayloadEnumCaseBuilderRejectsMalformedBody(t *testing.T) {
	source := `module GeneratedEnum; profile Core;
enum Choice { None, Some(int value), }
generator <typename T> DeriveTag
int Tag(ref const T item) {
    foreach (EnumCaseInfo option in Cases<T>()) {
        foreach (FieldInfo field in Fields<T>()) { return 1; }
    }
    return 0;
}
derive DeriveTag reflect<Choice>;
`
	_, err := Parse("generated_enum.concept", source)
	if err == nil || !strings.Contains(err.Error(), "GENERATOR_CASE_BODY_INVALID") {
		t.Fatalf("expected bounded case-generation diagnostic, got %v", err)
	}
}
