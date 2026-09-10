package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func r6hMemoryArtifact(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile("../../libraries/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	return buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(source), nil)
}

func TestSemanticValueFactsSurviveResultReturnLocalsAndCopies(t *testing.T) {
	artifact := r6hMemoryArtifact(t)
	generic := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil)
	source := `module App;
profile Core;
import Standard.MemoryGeometry;
import Standard.Generic;

Result<int, RegionError> Verify()
{
    int<array>[4] backing = [0, 0, 0, 0];
    Address<SystemMemory> start = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = 16;
    usize<byte> alignment = 4;
    MemoryRegion<SystemMemory> parent = MemoryRegion<SystemMemory>{start, extent, alignment};
    usize<byte> firstOffset = 0;
    usize<byte> secondOffset = 8;
    usize<byte> length = 8;
    MemoryRegion<SystemMemory> first = Subregion(parent, firstOffset, length)?;
    MemoryRegion<SystemMemory> second = Subregion(parent, secondOffset, length)?;
    MemoryRegion<SystemMemory> copy = first;
    Box<MemoryRegion<SystemMemory>> boxed = Box<MemoryRegion<SystemMemory>>{copy};
    MemoryRegion<SystemMemory> unwrapped = boxed.value;
    Assert.Concept<Aligned<4>>(unwrapped, "copy and generic field retain derived alignment");
    Assert.Concept<SameRegion>(unwrapped, second, "returned subregions retain origin");
    Assert.Concept<Disjoint>(unwrapped, second, "returned subregions retain intervals");
    const auto inspect = callback()
    with (unwrapped)
    {
        Assert.Concept<Aligned<4>>(unwrapped, "capture field retains alignment");
        return 1;
    };
    int observed = inspect();
    return Result::Ok(observed);
}`
	module, err := ParseWithSemanticModules("App.concept", source, map[string][]byte{"Standard.MemoryGeometry": artifact, "Standard.Generic": generic})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	for _, want := range []string{"\"value_fact_summaries\"", "\"RegionOrigin\"", "\"ByteInterval\"", "\"CommonAlignment\""} {
		if !strings.Contains(string(artifact)+mir, want) {
			t.Fatalf("fact transport evidence omitted %s", want)
		}
	}
}

func TestSemanticValueFactsDisproveKnownOverlapAndKeepUnknownOrigin(t *testing.T) {
	artifact := r6hMemoryArtifact(t)
	overlap := `module App; profile Core; import Standard.MemoryGeometry;
Result<int, RegionError> Verify()
{
    int<array>[4] backing = [0, 0, 0, 0];
    Address<SystemMemory> start = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = 16;
    usize<byte> alignment = 4;
    MemoryRegion<SystemMemory> parent = MemoryRegion<SystemMemory>{start, extent, alignment};
    usize<byte> leftOffset = 0;
    usize<byte> rightOffset = 4;
    usize<byte> length = 8;
    MemoryRegion<SystemMemory> left = Subregion(parent, leftOffset, length)?;
    MemoryRegion<SystemMemory> right = Subregion(parent, rightOffset, length)?;
    Assert.Concept<Disjoint>(left, right, "known overlap is disproven");
    return Result::Ok(0);
}`
	_, err := ParseWithSemanticModules("overlap.concept", overlap, map[string][]byte{"Standard.MemoryGeometry": artifact})
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
		t.Fatalf("expected known overlap to disprove Disjoint, got %v", err)
	}
	unknown := `profile Core; struct SystemMemory {} template <typename T> struct Box { T value; };
int Verify(usize bits)
{
    Address<SystemMemory> raw = AddressFromBits<SystemMemory>(bits);
    Box<Address<SystemMemory>> boxed = Box<Address<SystemMemory>>{raw};
    Address<SystemMemory> again = boxed.value;
    Assert.Concept<SameRegion>(raw, again, "reconstructed bits remain unknown");
    return 0;
}`
	_, err = Parse("unknown.concept", unknown)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("expected AddressFromBits region identity to remain Unknown, got %v", err)
	}
}

func TestSemanticModuleValueSummariesAreDeterministicAndChecked(t *testing.T) {
	first := r6hMemoryArtifact(t)
	for run := 1; run < 100; run++ {
		if next := r6hMemoryArtifact(t); !bytes.Equal(first, next) {
			t.Fatalf("semantic value summaries changed on build %d", run)
		}
	}
	var artifact SemanticModuleArtifact
	if err := json.Unmarshal(first, &artifact); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, summary := range artifact.ValueFactSummaries {
		if summary.Operation == "Subregion" && summary.Result.RegionOrigin != nil && summary.Result.RelativeOffset != nil && summary.Result.ByteExtent != nil && summary.Result.Alignment != nil {
			found = true
			parent := &SemanticValueFacts{RegionOrigin: "R", RelativeOffset: SemanticKnownInt{Known: true}, Alignment: SemanticKnownInt{Known: true, Value: 64}}
			instantiated := instantiateSemanticFactSummary(summary.Result, []*SemanticValueFacts{parent, nil, nil}, "Subregion")
			if instantiated.RelativeOffset.Known || instantiated.ByteExtent.Known || instantiated.Alignment.Known {
				t.Fatalf("runtime arguments were strengthened by summary instantiation: %#v", instantiated)
			}
		}
	}
	if !found {
		t.Fatalf("Subregion artifact summary is incomplete: %#v", artifact.ValueFactSummaries)
	}
	artifact.ValueFactSummaries = append(artifact.ValueFactSummaries, artifact.ValueFactSummaries[0])
	hash, err := semanticArtifactHash(artifact)
	if err != nil {
		t.Fatal(err)
	}
	artifact.ContentSHA256 = hash
	corrupt, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = LoadSemanticModuleArtifact(corrupt); err == nil || !strings.Contains(err.Error(), "MODULE_FACT_SUMMARY_INVALID") {
		t.Fatalf("duplicate value summary was not rejected: %v", err)
	}
}

func TestSemanticValueFactsNeverStrengthenUnknownAndModuleCacheIsEquivalent(t *testing.T) {
	unknown := `profile Core; struct SystemMemory {} template <typename T> struct Box { T value; };
int Verify(usize bits)
{
    Address<SystemMemory> raw = AddressFromBits<SystemMemory>(bits);
    Box<Address<SystemMemory>> boxed = Box<Address<SystemMemory>>{raw};
    Address<SystemMemory> again = boxed.value;
    Assert.Concept<Aligned<64>>(again, "wrapping cannot manufacture alignment");
    return 0;
}`
	if _, err := Parse("unknown_alignment.concept", unknown); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("unknown alignment was accidentally strengthened: %v", err)
	}

	artifact := r6hMemoryArtifact(t)
	source := `module App; profile Core; import Standard.MemoryGeometry;
Result<int, RegionError> Verify()
{
    int<array>[4] backing = [0, 0, 0, 0];
    Address<SystemMemory> start = AddressOf<SystemMemory>(ref backing);
	usize<byte> extent = 16;
	usize<byte> alignment = 4;
	usize<byte> offset = 4;
    MemoryRegion<SystemMemory> parent = MemoryRegion<SystemMemory>{start, extent, alignment};
    MemoryRegion<SystemMemory> child = Subregion(parent, offset, alignment)?;
    Assert.Concept<Aligned<4>>(child, "cached summary retains alignment");
    return Result::Ok(0);
}`
	var baseline Outputs
	for run := 0; run < 2; run++ {
		module, err := ParseWithSemanticModules("App.concept", source, map[string][]byte{"Standard.MemoryGeometry": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			baseline = outputs
			continue
		}
		for name, first := range baseline {
			if !bytes.Equal(first, outputs[name]) {
				t.Fatalf("fresh/cached module output differs for %s", name)
			}
		}
	}
}

func TestSemanticFactsProjectToAsyncAndMachinePersistentFields(t *testing.T) {
	asyncSource, err := os.ReadFile("../../language/evt1/async/basic/valid/async_tensor_live_across_await.concept")
	if err != nil {
		t.Fatal(err)
	}
	asyncModule, err := Parse("fact_async_persistence.concept", string(asyncSource))
	if err != nil {
		t.Fatal(err)
	}
	asyncOutputs, err := Generate(asyncModule, asyncSource)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(moduleOutput(t, asyncOutputs, ".mir.json"), string(SubjectAsyncPersistentField)) {
		t.Fatal("async frame did not receive a transported semantic fact subject")
	}

	machineSource := `profile Core;
automata Worker
with state
{
    Span<int> values;
}
{
    machine Main
    {
        state Running
        {
            int observed = state.values[0];
            yield;
        }
    }
}
int Main()
{
    int<array>[2] backing = [7, 9];
    Span<int> values = Span(backing);
    instance Worker worker(values);
    Step(worker, Main);
    return 0;
}`
	machineModule, err := Parse("fact_machine_persistence.concept", machineSource)
	if err != nil {
		t.Fatal(err)
	}
	machineOutputs, err := Generate(machineModule, []byte(machineSource))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, machineOutputs, ".mir.json")
	if !strings.Contains(mir, string(SubjectMachineField)) || !strings.Contains(mir, string(FactTransformMachinePersist)) {
		t.Fatal("machine persistent field omitted transported region facts")
	}
	generated := moduleOutput(t, machineOutputs, ".generated.c")
	for _, forbidden := range []string{"provenance_id", "semantic_fact", "region_table", "rtti", "malloc("} {
		if strings.Contains(strings.ToLower(generated), forbidden) {
			t.Fatalf("fact transport leaked runtime metadata %q", forbidden)
		}
	}
}

func TestSemanticFactTransportCorpusCompilesAndHasNoRuntimeProofBaggage(t *testing.T) {
	root := "../../language/evt1/tooling/semantic-facts/valid"
	paths, err := filepath.Glob(filepath.Join(root, "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 22 {
		t.Fatalf("R6h valid fixture count = %d, want 22", len(paths))
	}
	memory := r6hMemoryArtifact(t)
	genericSource, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/Generic.concept")
	if err != nil {
		t.Fatal(err)
	}
	hostSource, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/Host.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{
		"Standard.MemoryGeometry": memory,
		"Standard.Generic":        buildSemanticArtifact(t, "Standard/Generic.concept", string(genericSource), nil),
		"Standard.Host":           buildSemanticArtifact(t, "Standard/Host.concept", string(hostSource), nil),
	}
	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".concept"), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := ParseWithSemanticModules(filepath.ToSlash(path), string(source), artifacts)
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, source)
			if err != nil {
				t.Fatal(err)
			}
			for name, body := range outputs {
				if !strings.HasSuffix(name, ".generated.c") {
					continue
				}
				lower := strings.ToLower(string(body))
				for _, forbidden := range []string{"semantic_fact", "provenance_id", "region_table", "rtti", "malloc("} {
					if strings.Contains(lower, forbidden) {
						t.Fatalf("generated C contains runtime proof baggage %q", forbidden)
					}
				}
			}
			if filepath.Base(path) != "fact_module_effect_regression.concept" {
				base := evt1OutputBase(filepath.ToSlash(path))
				runFoundationNativeHarness(t, outputs, base+"_compile_harness.c", "#include \""+base+".generated.h\"\nint main(void) { return 0; }\n")
			}
		})
	}
}

func TestSemanticFactTransportUnknownCorpusStaysConservative(t *testing.T) {
	root := "../../language/evt1/tooling/semantic-facts/unknown"
	paths, err := filepath.Glob(filepath.Join(root, "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 6 {
		t.Fatalf("R6h unknown fixture count = %d, want 6", len(paths))
	}
	memory := r6hMemoryArtifact(t)
	genericSource, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/Generic.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{
		"Standard.MemoryGeometry": memory,
		"Standard.Generic":        buildSemanticArtifact(t, "Standard/Generic.concept", string(genericSource), nil),
	}
	for _, path := range paths {
		name := filepath.Base(path)
		t.Run(strings.TrimSuffix(name, ".concept"), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			selected := artifacts
			if name == "fact_module_missing_summary_unknown.concept" {
				var legacy SemanticModuleArtifact
				if err := json.Unmarshal(memory, &legacy); err != nil {
					t.Fatal(err)
				}
				legacy.ValueFactSummaries = nil
				legacy.ContentSHA256, err = semanticArtifactHash(legacy)
				if err != nil {
					t.Fatal(err)
				}
				body, err := json.Marshal(legacy)
				if err != nil {
					t.Fatal(err)
				}
				selected = map[string][]byte{"Standard.MemoryGeometry": body, "Standard.Generic": artifacts["Standard.Generic"]}
			}
			_, err = ParseWithSemanticModules(filepath.ToSlash(path), string(source), selected)
			if err == nil {
				t.Fatal("negative/Unknown fixture was accepted")
			}
			if name == "fact_ref_escape_through_wrapper.concept" {
				if !strings.Contains(err.Error(), "SEMANTIC_REGION_LIFETIME_ESCAPE") {
					t.Fatalf("wrapper escape diagnostic = %v", err)
				}
			} else if !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
				t.Fatalf("conservative boundary diagnostic = %v", err)
			}
		})
	}
}

func TestSemanticFactTransportDogfoodExecutes(t *testing.T) {
	root := "../../language/evt1/tooling/semantic-facts"
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tests) != 4 {
		t.Fatalf("R6h dogfood test count = %d, want 4", len(manifest.Tests))
	}
	run, err := RunTests(manifest, TestRunOptions{ResultsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if run.Failed != 0 || run.Passed != 4 {
		t.Fatalf("R6h dogfood run = %+v", run)
	}
}

func TestSemanticFactTransportProofGoldens(t *testing.T) {
	memory := r6hMemoryArtifact(t)
	artifacts := map[string][]byte{"Standard.MemoryGeometry": memory}
	root := "../../language/evt1/tooling/semantic-facts"
	renderValid := func(name string) string {
		source, err := os.ReadFile(filepath.Join(root, "valid", name))
		if err != nil {
			t.Fatal(err)
		}
		module, err := ParseWithSemanticModules(name, string(source), artifacts)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		var mir MIR
		if err := json.Unmarshal([]byte(moduleOutput(t, outputs, ".mir.json")), &mir); err != nil {
			t.Fatal(err)
		}
		if len(mir.ProofGraphs) == 0 {
			t.Fatalf("%s emitted no proof graph", name)
		}
		return RenderProofSummary(mir.ProofGraphs[len(mir.ProofGraphs)-1])
	}
	renderInvalid := func(class, name string, transform func(string) string) string {
		source, err := os.ReadFile(filepath.Join(root, class, name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(source)
		if transform != nil {
			body = transform(body)
		}
		_, err = ParseWithSemanticModules(name, body, artifacts)
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Proof == nil {
			t.Fatalf("%s has no structured proof diagnostic: %v", name, err)
		}
		return RenderProofSummary(*diagnostic.Proof)
	}
	cases := []struct {
		golden string
		output string
	}{
		{"r6h-alignment-transport.txt", renderValid("fact_region_alignment_degrade.concept")},
		{"r6h-disjoint-transport.txt", renderValid("fact_region_disjoint.concept")},
		{"r6h-overlap-transport.txt", renderInvalid("valid", "fact_region_overlap.concept", func(body string) string {
			return strings.Replace(body, "Assert.Concept<SameRegion>(left, right, \"overlapping views retain their common origin\")", "Assert.Concept<Disjoint>(left, right, \"known overlap is disproven\")", 1)
		})},
		{"r6h-opaque-boundary.txt", renderInvalid("unknown", "fact_opaque_extern_unknown.concept", nil)},
	}
	for _, tc := range cases {
		golden, err := os.ReadFile(filepath.Join("../../language/evt1/tooling/proofs/golden", tc.golden))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(golden)), "\n") {
			if !strings.Contains(tc.output, strings.TrimSpace(line)) {
				t.Fatalf("%s missing golden line %q:\n%s", tc.golden, line, tc.output)
			}
		}
	}
}

func BenchmarkSemanticFactTransportModule(b *testing.B) {
	library, err := os.ReadFile("../../libraries/Standard/MemoryGeometry.concept")
	if err != nil {
		b.Fatal(err)
	}
	artifact, err := CompileSemanticModule("Standard/MemoryGeometry.concept", string(library), nil)
	if err != nil {
		b.Fatal(err)
	}
	source, err := os.ReadFile("../../language/evt1/tooling/semantic-facts/valid/fact_region_disjoint.concept")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for run := 0; run < b.N; run++ {
		module, err := ParseWithSemanticModules("fact_region_disjoint.concept", string(source), map[string][]byte{"Standard.MemoryGeometry": artifact})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := Generate(module, source); err != nil {
			b.Fatal(err)
		}
	}
}
