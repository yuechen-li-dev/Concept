package concept

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// The strangler protocol for a rule moving from Go into an innate concept:
// the Go rule and the innate concept run separately over the whole corpus
// and the rule's own cases, and must agree on every file: the same
// diagnostic code, site, and message, or both silent.

func evt1ParseWithOptions(path, text string, options evt1AnalysisOptions) error {
	module, err := parseSyntaxModule(path, text)
	if err != nil {
		return err
	}
	if err := evt1MaterializeGeneratedDeclarations(&module); err != nil {
		return err
	}
	_, err = evt1AnalyzeModule(module, options)
	return err
}

type innateAgreementCase struct {
	path, source string
}

func innateAgreementCorpus(t *testing.T) []innateAgreementCase {
	t.Helper()
	_, manifest := loadSemanticCorpusManifest(t)
	root := filepath.Join("..", "..", "language", "evt1")
	var cases []innateAgreementCase
	for _, subsystem := range manifest.Subsystems {
		for _, group := range []string{"valid", "invalid"} {
			paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(subsystem.Path), group, "*.concept"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				cases = append(cases, innateAgreementCase{filepath.ToSlash(path), string(source)})
			}
		}
	}
	return cases
}

func assertInnateAgreement(t *testing.T, code string, cases []innateAgreementCase) {
	t.Helper()
	goOnly := evt1AnalysisOptions{innateOff: map[string]bool{code: true}}
	innateOnly := evt1AnalysisOptions{goRulesOff: map[string]bool{code: true}}
	fired := 0
	for _, tc := range cases {
		goErr := evt1ParseWithOptions(tc.path, tc.source, goOnly)
		innateErr := evt1ParseWithOptions(tc.path, tc.source, innateOnly)
		var goDiagnostic, innateDiagnostic Diagnostic
		goIs := errors.As(goErr, &goDiagnostic) && goDiagnostic.Code == code
		innateIs := errors.As(innateErr, &innateDiagnostic) && innateDiagnostic.Code == code
		if goIs || innateIs {
			fired++
			if !goIs || !innateIs || goDiagnostic.Span != innateDiagnostic.Span || goDiagnostic.Message != innateDiagnostic.Message {
				t.Errorf("%s disagrees on %s:\n  go:     %v\n  innate: %v", code, tc.path, goErr, innateErr)
			}
			continue
		}
		if (goErr == nil) != (innateErr == nil) {
			t.Errorf("%s: one implementation accepts and the other rejects %s:\n  go:     %v\n  innate: %v", code, tc.path, goErr, innateErr)
		}
	}
	if fired == 0 {
		t.Fatalf("no case exercises %s", code)
	}
}

const cv4653Header = "module Ownership;\nprofile Core;\n\nclass Resource\n{\npublic:\n    int id;\n};\n\nvoid Drop(owned Resource resource)\n{\n}\n\n"

func TestCV4653GoAndInnateAgree(t *testing.T) {
	t.Parallel()
	cases := innateAgreementCorpus(t)
	for name, body := range map[string]string{
		"plain":            "class Holder\n{\npublic:\n    Resource resource;\n};\n",
		"owned":            "class Holder\n{\npublic:\n    owned Resource resource;\n};\n",
		"record":           "record struct Holder\n{\n    int count;\n    Resource resource;\n};\n",
		"borrowed":         "ref struct Holder\n{\n    ref const Resource resource;\n}\n",
		"generic instance": "template <typename T>\nstruct Box\n{\n    T value;\n}\n\nint Use()\n{\n    owned Resource resource = Resource{7};\n    owned Box<Resource> box = Box<Resource>{move resource};\n    return box.value.id;\n}\n",
		"owned generic":    "template <typename T>\nstruct Box\n{\n    owned T value;\n}\n\nint Use()\n{\n    owned Box<Resource> box = Box<Resource>{Resource{1}};\n    owned Box<int> count = Box<int>{2};\n    return box.value.id + count.value;\n}\n",
		"plain generic":    "template <typename T>\nstruct Box\n{\n    T value;\n}\n\nint Use()\n{\n    Box<int> count = Box<int>{2};\n    return count.value;\n}\n",
	} {
		cases = append(cases, innateAgreementCase{"cv4653/" + name + ".concept", cv4653Header + body})
	}
	assertInnateAgreement(t, "CV4653", cases)
}
