package concept

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The strangler protocol for a rule moving from Go into an innate concept
// (EVT2-INNATE-CONCEPTS.md): shadow, switch, delete. During shadow and switch,
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
	goOnly := evt1AnalysisOptions{goRulesOn: map[string]bool{code: true}, innateOff: map[string]bool{code: true}}
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
			t.Logf("%s: %v", tc.path, innateErr)
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

// CV4653 completed the protocol (shadow, switch, delete); its Go rule is
// gone and DroppableFieldIsOwned in the innate module is the only
// implementation. These are the rule's cases, now checked directly.
func TestCV4653IsTheInnateConcept(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		body, message string
		line          int
	}{
		"plain":            {"class Holder\n{\npublic:\n    Resource resource;\n};\n", "field Holder.resource holds Resource, which has a Drop; declare it `owned Resource resource;`", 17},
		"record":           {"record struct Holder\n{\n    int count;\n    Resource resource;\n};\n", "field Holder.resource holds Resource, which has a Drop; declare it `owned Resource resource;`", 17},
		"generic instance": {"template <typename T>\nstruct Box\n{\n    T value;\n}\n\nint Use()\n{\n    owned Resource resource = Resource{7};\n    owned Box<Resource> box = Box<Resource>{move resource};\n    return box.value.id;\n}\n", "field Box<Resource>.value holds Resource, which has a Drop; declare it `owned T value;` in Box", 17},
		"owned":            {"class Holder\n{\npublic:\n    owned Resource resource;\n};\n", "", 0},
		"borrowed":         {"ref struct Holder\n{\n    ref const Resource resource;\n}\n", "", 0},
		"owned generic":    {"template <typename T>\nstruct Box\n{\n    owned T value;\n}\n\nint Use()\n{\n    owned Box<Resource> box = Box<Resource>{Resource{1}};\n    owned Box<int> count = Box<int>{2};\n    return box.value.id + count.value;\n}\n", "", 0},
		"plain generic":    {"template <typename T>\nstruct Box\n{\n    T value;\n}\n\nint Use()\n{\n    Box<int> count = Box<int>{2};\n    return count.value;\n}\n", "", 0},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse("cv4653.concept", cv4653Header+tc.body)
			if tc.message == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4653" {
				t.Fatalf("expected CV4653, got %v", err)
			}
			if diagnostic.Message != tc.message || diagnostic.Span.Line != tc.line {
				t.Fatalf("got %q at line %d", diagnostic.Message, diagnostic.Span.Line)
			}
			if diagnostic.Proof == nil || !strings.Contains(diagnostic.Proof.Reason, "DroppableFieldIsOwned (CV4653)") {
				t.Fatalf("the diagnostic carries the innate concept's proof: %+v", diagnostic.Proof)
			}
		})
	}
}

func TestCReprGoAndInnateAgree(t *testing.T) {
	t.Parallel()
	cases := innateAgreementCorpus(t)
	header := "module CRepr;\nprofile Core;\n\nextern \"C\" handle Device;\n\n"
	for name, body := range map[string]string{
		"scalars":        "[[repr(C)]]\nrecord struct Plain\n{\n    int a;\n    uint64 b;\n    double c;\n    byte d;\n};\n",
		"handle":         "[[repr(C)]]\nrecord struct Holder\n{\n    Device device;\n    uint32 index;\n};\n",
		"nested":         "[[repr(C)]]\nrecord struct Inner\n{\n    int a;\n};\n\n[[repr(C)]]\nrecord struct Outer\n{\n    Inner inner;\n    Inner<array>[4] many;\n};\n",
		"nested no repr": "record struct Inner\n{\n    int a;\n};\n\n[[repr(C)]]\nrecord struct Outer\n{\n    int b;\n    Inner inner;\n};\n",
		"deep path":      "record struct Leaf\n{\n    int a;\n};\n\n[[repr(C)]]\nrecord struct Middle\n{\n    Leaf<array>[2] leaves;\n};\n\n[[repr(C)]]\nrecord struct Outer\n{\n    Middle middle;\n};\n",
		"bool":           "[[repr(C)]]\nrecord struct Flags\n{\n    bool on;\n};\n",
		"string":         "[[repr(C)]]\nrecord struct Named\n{\n    string name;\n};\n",
		"enum field":     "enum Mode { A, B, }\n\n[[repr(C)]]\nrecord struct Tagged\n{\n    Mode mode;\n};\n",
		"class":          "[[repr(C)]]\nclass Thing\n{\npublic:\n    int a;\n};\n",
		"plain struct":   "[[repr(C)]]\nstruct Thing\n{\n    int a;\n}\n",
		"drop":           "[[repr(C)]]\nrecord struct Owner\n{\n    int id;\n};\n\nvoid Drop(owned Owner owner)\n{\n}\n",
		"nested drop":    "[[repr(C)]]\nrecord struct Owner\n{\n    int id;\n};\n\nvoid Drop(owned Owner owner)\n{\n}\n\n[[repr(C)]]\nrecord struct Outer\n{\n    int a;\n};\n",
		"no repr":        "record struct Free\n{\n    bool on;\n    string name;\n};\n",
		"deep path late": "[[repr(C)]]\nrecord struct Outer\n{\n    Middle middle;\n};\n\n[[repr(C)]]\nrecord struct Middle\n{\n    int count;\n    Leaf<array>[2] leaves;\n};\n\nrecord struct Leaf\n{\n    int a;\n};\n",
		"deep valid":     "[[repr(C)]]\nrecord struct Outer\n{\n    Middle middle;\n};\n\n[[repr(C)]]\nrecord struct Middle\n{\n    Leaf<array>[2] leaves;\n};\n\n[[repr(C)]]\nrecord struct Leaf\n{\n    Device device;\n};\n",
	} {
		cases = append(cases, innateAgreementCase{"crepr/" + name + ".concept", header + body})
	}
	assertInnateAgreement(t, "C_ABI_REPR_INVALID", cases)
}
