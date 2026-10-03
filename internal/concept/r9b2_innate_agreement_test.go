package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// A test-only snapshot of the retired legacy protocol. Production has one
// ownership predicate; this oracle qualifies the switch over the corpus.
const r9b2LegacyOwnership = "comptime Verdict DroppableFieldIsOwnedHolds(declaration field)\n{\n    typename held = compiler.TypeOf(field);\n    string heldName = compiler.TypeName(held);\n    string template = compiler.TemplateName(compiler.Parent(field));\n    string problem = \"field \" + compiler.QualifiedName(field) + \" holds \" + heldName + \", which has a Drop; \";\n    return match\n    {\n        when compiler.Owned(field) or compiler.IsBorrowLike(held) or not compiler.HasDrop(held) => Verdict::Holds,\n        when template != \"\" => Verdict::Refuted(field, problem + \"declare it `owned \" + compiler.DeclaredTypeName(field) + \" \" + compiler.Name(field) + \";` in \" + template),\n        otherwise => Verdict::Refuted(field, problem + \"declare it `owned \" + heldName + \" \" + compiler.Name(field) + \";`\"),\n    };\n}\n\n"

func TestR9b2OwnershipProtocolAgreement(t *testing.T) {
	start := strings.Index(evt1InnateSource, "enum OwnershipEvidence")
	end := strings.Index(evt1InnateSource, "// A movable aggregate")
	legacy := evt1InnateSource[:start] + r9b2LegacyOwnership + evt1InnateSource[end:]
	_, previous, err := evt1CompileInnateSource("legacy-innate.concept", legacy)
	if err != nil {
		t.Fatal(err)
	}
	current, err := evt1LoadInnate()
	if err != nil {
		t.Fatal(err)
	}
	cases := innateAgreementCorpus(t)
	cases = append(cases,
		innateAgreementCase{"plain.concept", cv4653Header + "struct Holder { Resource resource; }"},
		innateAgreementCase{"generic.concept", cv4653Header + "template <typename T> struct Box { T value; } int Use() { owned Box<Resource> b = Box<Resource>{Resource{1}}; return b.value.id; }"},
	)
	compared, refuted := 0, 0
	for _, tc := range cases {
		module, err := parseSyntaxModule(tc.path, tc.source)
		if err != nil {
			continue
		}
		if err := evt1MaterializeGeneratedDeclarations(&module); err != nil {
			continue
		}
		if err := resolveNamespaceSymbols(&module); err != nil {
			continue
		}
		env, err := evt1AnalyzeModule(module, evt1AnalysisOptions{innateOff: map[string]bool{"CV4653": true}})
		if err != nil {
			continue
		}
		for _, subject := range evt1InnateSubjects(env, module) {
			if subject.Kind != FieldDeclaration {
				continue
			}
			args := []Value{evt1DeclarationValue(subject)}
			before, beforeErr := evt1InvokeInnatePredicate(env, previous, "DroppableFieldIsOwnedHolds", args, subject, subject.Site)
			after, afterErr := evt1InvokeInnatePredicate(env, current.env, "DroppableFieldIsOwnedHolds", args, subject, subject.Site)
			if beforeErr != nil || afterErr != nil || before.Outcome != after.Outcome || before.At != after.At {
				t.Fatalf("%s %s: before=%+v %v after=%+v %v", tc.path, subject.qualifiedName(), before, beforeErr, after, afterErr)
			}
			compared++
			if after.Outcome == FactDisproven {
				refuted++
				if before.Message != after.Message || after.Refutation == nil {
					t.Fatalf("diagnostic changed: before=%q after=%+v", before.Message, after)
				}
			}
		}
	}
	if compared == 0 || refuted < 2 {
		t.Fatalf("missing shadow coverage: compared=%d refuted=%d", compared, refuted)
	}
	t.Logf("ownership shadow corpus: %d field decisions, %d typed refutations; truth/site/message agree; CV4653 remains the concept's diagnostic", compared, refuted)
}

func TestR9b2InnateTypedExplanationDeterminism(t *testing.T) {
	source := cv4653Header + "struct Holder { owned Resource resource; }\n"
	line := strings.Count(cv4653Header, "\n") + 1
	graph, err := ExplainSource("ownership.concept", source, line)
	if err != nil {
		t.Fatal(err)
	}
	// Query the field rather than the enclosing type when they share a line.
	module, err := Parse("ownership.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, evaluation := range env.innateEvaluations {
		if evaluation.Concept == "DroppableFieldIsOwned" && evaluation.Subject.Parent == "Holder" {
			graph = evt1InnateProofGraph("ownership.concept", evaluation.Subject, []evt1InnateEvaluation{evaluation})
		}
	}
	if !strings.Contains(RenderProofVerbose(graph), "evidence:") {
		t.Fatalf("missing evidence: %s", RenderProofVerbose(graph))
	}
	baseline, _ := json.Marshal(graph)
	for run := 0; run < 100; run++ {
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		for _, evaluation := range env.innateEvaluations {
			if evaluation.Concept == "DroppableFieldIsOwned" && evaluation.Subject.Parent == "Holder" {
				again := evt1InnateProofGraph("ownership.concept", evaluation.Subject, []evt1InnateEvaluation{evaluation})
				encoded, _ := json.Marshal(again)
				if !bytes.Equal(encoded, baseline) {
					t.Fatalf("innate proof drift at %d", run)
				}
			}
		}
	}
}
