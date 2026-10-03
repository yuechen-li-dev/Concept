package concept

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestR9b3CAbiAndVulkanBoundaries(t *testing.T) {
	artifacts := r9b3VocabularyArtifacts(t)
	plain := `module Research.Native;
profile Core;
import Standard.Semantic.Vocabulary;
record struct BoolHeader
{
    bool flag;
}
int Main()
{
    Assert.Concept<PlainData>(BoolHeader, "self-contained semantic record");
    return 0;
}
`
	if _, err := ParseWithSemanticModules("native.concept", plain, artifacts); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{
		`extern "C" BoolHeader Exchange(BoolHeader value);
`,
		`concept NativeC<T>
{
    requires compiler.CAbiLayout<T>();
}
requires NativeC<half>;
`,
	} {
		_, err := ParseWithSemanticModules("native.concept", plain+extra, artifacts)
		if err == nil {
			t.Fatal("PlainData incorrectly implied ABI compatibility")
		}
	}
	handle := `module Research.Handle;
profile Core;
import Standard.Semantic.Vocabulary;
extern "C" handle Handle;
extern "C" Handle Exchange(Handle value);
int Check()
{
    Assert.Concept<PlainData>(Handle, "ABI pointer bits do not prove self-contained data");
    return 0;
}
`
	_, err := ParseWithSemanticModules("handle.concept", handle, artifacts)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("handle ABI/PlainData: %v", err)
	}
	project, err := LoadNativeProject("../../libraries/Vulkan")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	vulkan, identity, err := BuildNativeCompanionArtifacts(project)
	if err != nil {
		t.Fatal(err)
	}
	unsafe := `module Research.UnsafeUpload;
profile Core;
import Vulkan;
record struct ReferenceData
{
    string text;
}
Result<void, VulkanError> Transfer(ref const Context context, ref const Buffer<ReferenceData> target,
ReadOnlySpan<ReferenceData> data)
{
    return Upload<ReferenceData>(ref const context, ref const target, data);
}
`
	_, err = ParseWithSemanticModulesForNative("unsafe-upload.concept", unsafe, vulkan, identity)
	if err == nil || !strings.Contains(err.Error(), "FieldProblem") || !strings.Contains(err.Error(), "ReferenceData.text") {
		t.Fatalf("unsafe byte upload: %v", err)
	}
	opaque := `module Research.UnsafeHandle;
profile Core;
import Vulkan;
extern "C" handle ForeignOpaque;
Result<void, VulkanError> Transfer(ref const Context context, ref const Buffer<ForeignOpaque> target,
ReadOnlySpan<ForeignOpaque> data)
{
    return Upload<ForeignOpaque>(ref const context, ref const target, data);
}
`
	_, err = ParseWithSemanticModulesForNative("unsafe-handle.concept", opaque, vulkan, identity)
	if err == nil || !strings.Contains(err.Error(), "PREDICATE_REQUIREMENT_UNDECIDED") || !strings.Contains(err.Error(), "no trusted self-contained representation summary") {
		t.Fatalf("unqualified opaque handle upload: %v", err)
	}
}

func TestR9b3FiniteResearchAndOrdering(t *testing.T) {
	path := "testdata/r9b3/Research/Finite.concept"
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{"Research.Finite": buildSemanticArtifact(t, path, string(body), nil)}
	source := `module Research.Cases;
profile Core;
import Research.Finite;
enum Message
{
    Empty,
    Text(string value),
}
enum Tag
{
    Start,
    Finish,
}
comptime string Ordered(typename subject)
{
    declaration sum = compiler.Declaration(subject);
    string result = "";
    for (i in 0 .. compiler.CaseCount(sum))
    {
        result = result + compiler.CaseName(sum, i);
    }
    return result;
}
comptime int TagAt(typename subject, int index)
{
    return compiler.CaseTag(compiler.Declaration(subject), index);
}
comptime int PayloadAt(typename subject, int index)
{
    return compiler.CasePayloadCount(compiler.Declaration(subject), index);
}
int Check()
{
    Assert.Concept<FiniteCases>(Message, "research case inventory");
    Assert.Concept<FiniteDomain>(Tag, "research complete tag values");
    return 0;
}
`
	var first []byte
	for run := 0; run < 100; run++ {
		module, err := ParseWithSemanticModules("cases.concept", source, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		result, err := evt1InvokeComptimeFunction(env, "Ordered", []Value{evt1TypenameValue(Type{Name: "Message", Kind: TypeEnum})}, Span{})
		if err != nil || result.StringValue != "EmptyText" {
			t.Fatalf("case order: %+v %v", result, err)
		}
		indexType, _ := evt1BuiltinType("int", Span{})
		for index := 0; index < 2; index++ {
			args := []Value{evt1TypenameValue(Type{Name: "Message", Kind: TypeEnum}), {Kind: ValueInt, Type: indexType, IntValue: index}}
			for _, observer := range []string{"TagAt", "PayloadAt"} {
				observed, err := evt1InvokeComptimeFunction(env, observer, args, Span{})
				if err != nil || observed.IntValue != index {
					t.Fatalf("%s(%d): %+v %v", observer, index, observed, err)
				}
			}
		}
		_, err = evt1InvokeComptimeFunction(env, "TagAt", []Value{evt1TypenameValue(Type{Name: "Message", Kind: TypeEnum}), {Kind: ValueInt, Type: indexType, IntValue: -1}}, Span{})
		if err == nil || !strings.Contains(err.Error(), "OBSERVATION_INVALID") {
			t.Fatalf("invalid case index: %v", err)
		}
		artifact := buildSemanticArtifact(t, "cases.concept", source, artifacts)
		if run == 0 {
			first = artifact
		}
		if !bytes.Equal(artifact, first) {
			t.Fatalf("finite research artifact drift %d", run)
		}
	}
	_, err = ParseWithSemanticModules("cases.concept", strings.Replace(source, "<FiniteDomain>(Tag", "<FiniteDomain>(Message", 1), artifacts)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("payload value domain: %v", err)
	}
	_, err = ParseWithSemanticModules("cases.concept", strings.Replace(source, "<FiniteCases>(Message", "<FiniteCases>(int", 1), artifacts)
	if err == nil || !strings.Contains(err.Error(), "NotVariantType") {
		t.Fatalf("case refutation: %v", err)
	}
	t.Logf("research-only finite artifact bytes=%d; 100 ordered case inventories", len(first))
}

func TestR9b3NegativeCorpusDiagnostics(t *testing.T) {
	for _, tc := range []struct{ file, code string }{
		{"plain_data_drop", "CONCEPT_ASSERT_DISPROVEN"},
		{"plain_data_owned", "CONCEPT_ASSERT_DISPROVEN"},
		{"plain_data_reference", "CONCEPT_ASSERT_DISPROVEN"},
		{"plain_data_opaque_unknown", "CONCEPT_ASSERT_UNKNOWN"},
		{"plain_data_payload_unknown", "CONCEPT_ASSERT_UNKNOWN"},
		{"finite_payload_domain_unknown", "CONCEPT_ASSERT_UNKNOWN"},
	} {
		path := "../../language/evt1/semantic-vocabulary/invalid/" + tc.file + ".concept"
		_, err := generateSemanticCorpusFile(path)
		var d Diagnostic
		if !errors.As(err, &d) || d.Code != tc.code {
			t.Fatalf("%s: %v", tc.file, err)
		}
	}
}
