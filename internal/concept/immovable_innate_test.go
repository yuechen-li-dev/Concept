package concept

import (
	"errors"
	"strings"
	"testing"
)

const cv4138Header = "module ImmovableFields;\nprofile Core;\nimmovable struct State { int id; };\n"

func cv4138Cases() []innateAgreementCase {
	return []innateAgreementCase{
		{"plain.concept", cv4138Header + "struct Holder { State state; };\n"},
		{"array.concept", cv4138Header + "struct Holder { State<array>[2] states; };\n"},
		{"immovable.concept", cv4138Header + "immovable struct Holder { State state; };\n"},
		{"reference.concept", cv4138Header + "ref struct Holder { ref const State state; };\n"},
		{"raw.concept", cv4138Header + "struct Holder { State<raw>[2] states; };\n"},
		{"sparse.concept", cv4138Header + "struct Holder { State<sparse>[2] states; };\n"},
		{"generic.concept", cv4138Header + "template <typename T> struct Holder { T state; }\nvoid Use(ref Holder<State> holder);\n"},
	}
}

// Shadow and switch agreed across the corpus before the Go rule was removed.
// Preserve that oracle's diagnostic and site; the proof now names the owner.
func TestCV4138IsTheInnateConcept(t *testing.T) {
	for _, tc := range cv4138Cases() {
		t.Run(tc.path, func(t *testing.T) {
			_, err := Parse(tc.path, tc.source)
			message := ""
			switch tc.path {
			case "plain.concept":
				message = "struct Holder cannot embed immovable field State"
			case "array.concept":
				message = "struct Holder cannot embed immovable field State<array>[2]"
			case "generic.concept":
				message = "struct Holder<State> cannot embed immovable field State"
			}
			if message == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4138" || diagnostic.Message != message {
				t.Fatalf("expected CV4138 %q, got %v", message, err)
			}
			module, err := parseSyntaxModule(tc.path, tc.source)
			if err != nil {
				t.Fatal(err)
			}
			want := module.Structs[len(module.Structs)-1].Fields[0].Span
			if tc.path == "generic.concept" {
				want = module.GenericTypes[0].Struct.Fields[0].Span
			}
			if diagnostic.Span != want {
				t.Fatalf("diagnostic site: got %+v, want %+v", diagnostic.Span, want)
			}
			if diagnostic.Proof == nil || !strings.Contains(diagnostic.Proof.Reason, "MovableFieldDoesNotEmbedImmovable (CV4138)") {
				t.Fatalf("missing innate owner: %+v", diagnostic.Proof)
			}
		})
	}
}

func TestCV4138GenericArtifactClosure(t *testing.T) {
	producer := "module Immovable.Library;\nprofile Core;\ntemplate <typename T> struct Holder { T state; }\n"
	body := buildSemanticArtifact(t, "holder.concept", producer, nil)
	consumer := "module Immovable.Consumer;\nprofile Core;\nimport Immovable.Library;\nimmovable struct State { int id; };\nvoid Use(ref Holder<State> holder);\n"
	_, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Immovable.Library": body})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4138" || diagnostic.Proof == nil {
		t.Fatalf("artifact-only generic escaped innate field legality: %v", err)
	}
}
