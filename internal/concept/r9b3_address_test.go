package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestR9b3AddressResearchBoundaries(t *testing.T) {
	path := "testdata/r9b3/Research/Address.concept"
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{"Research.Address": buildSemanticArtifact(t, path, string(body), nil)}
	source := `module Research.AddressSubjects;
profile Core;
import Research.Address;
immovable struct Stable
{
    int value;
}
record struct MovableData
{
    int value;
}
class Resource
{
    public:
    int id;
}
void Drop(owned Resource resource)
{
}
record struct Owner
{
    owned Resource resource;
}
int Check()
{
    Assert.Concept<ResearchRequiresStableAddress>(Stable, "research intrinsic requirement");
    return 0;
}
`
	var first []byte
	for run := 0; run < 100; run++ {
		module, err := ParseWithSemanticModules("address.concept", source, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		results := []PredicateVerdict{}
		for _, tc := range []struct {
			name    string
			outcome SemanticFactCertainty
		}{
			{"Stable", FactDisproven}, {"MovableData", FactUnknown}, {"Resource", FactUnknown}, {"Owner", FactUnknown},
		} {
			result, err := evt1InvokePredicateOnMeasured(env, env, "RelocationProbe", []Value{evt1TypenameValue(Type{Name: tc.name, Kind: TypeStruct})}, Span{}, nil)
			if err != nil || result.Outcome != tc.outcome || len(result.FactAuthority) != 0 {
				t.Fatalf("%s: %+v %v", tc.name, result, err)
			}
			results = append(results, result)
		}
		encoded, err := json.Marshal(results)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = encoded
		}
		if !bytes.Equal(encoded, first) {
			t.Fatalf("address research drift %d", run)
		}
	}
	_, err = ParseWithSemanticModules("address.concept", strings.Replace(source, "<ResearchRequiresStableAddress>(Stable", "<ResearchRequiresStableAddress>(MovableData", 1), artifacts)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_UNKNOWN" || !strings.Contains(diagnostic.Message, "external address invariants") {
		t.Fatalf("absence of requirement: %v", err)
	}
	t.Log("100 typed relocation results: immovable Disproven; movable, Drop and owned Unknown; no pin or relocation authority")
}

func TestR9b3UnknownRendererContract(t *testing.T) {
	for _, tc := range []struct{ helper, code string }{
		{`comptime int JudgeUnknownDescribe(typename subject)
{
    return 1;
}
`, "VERDICT_UNKNOWN_DESCRIBE_INVALID"},
		{`comptime string JudgeUnknownDescribe(declaration subject)
{
    return "missing";
}
`, "VERDICT_UNKNOWN_DESCRIBE_INVALID"},
	} {
		_, err := Parse("unknown-renderer.concept", r9b2Protocol+"\n"+tc.helper)
		var d Diagnostic
		if !errors.As(err, &d) || d.Code != tc.code {
			t.Fatalf("renderer admission: %v", err)
		}
	}
	source := strings.Replace(r9b2Protocol, "(int, \"typed proof\")", "(float, \"typed proof\")", 1) + `
comptime string JudgeUnknownDescribe(typename subject)
{
    return "no representation summary for " + compiler.TypeName(subject);
}
`
	_, err := Parse("unknown-renderer.concept", source)
	var d Diagnostic
	if !errors.As(err, &d) || d.Code != "CONCEPT_ASSERT_UNKNOWN" || !strings.Contains(d.Message, "no representation summary for float") {
		t.Fatal(err)
	}
}

func TestR9b3UnknownDescriptionBound(t *testing.T) {
	source := strings.Replace(r9b2Protocol, "(int, \"typed proof\")", "(float, \"typed proof\")", 1) + `
comptime string JudgeUnknownDescribe(typename subject)
{
    return "` + strings.Repeat("x", evt1ComptimeMaxStringBytes+1) + `";
}
`
	_, err := Parse("unknown-limit.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_UNKNOWN" || !strings.Contains(err.Error(), "VERDICT_DESCRIPTION_LIMIT") {
		t.Fatalf("literal description bound: %v", err)
	}
}
