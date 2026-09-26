package concept

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestR7j1TinyXML2ArtifactABIChainAndIdentity(t *testing.T) {
	project, err := LoadNativeProject(filepath.Join("..", "..", "tests", "dogfood", "tinyxml2"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := CompileNativeSemanticModule(project, "concept/Native.concept", nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _, err := LoadSemanticModuleArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.NativeABI == nil || len(parsed.NativeABI.Evidence) != 1 || parsed.NativeABI.Evidence[0].Origin != "NativeToolchainProbe" {
		t.Fatalf("artifact lost native evidence: %+v", parsed.NativeABI)
	}
	identity := parsed.NativeABI.Identity
	const bSource = `module B; profile Core; import Native;
requires CAbiValue<ConceptXmlStats>;
int StatsChildren(ConceptXmlStats stats) { return stats.children; }`
	b, err := CompileSemanticModuleWithNativeABI("B.concept", bSource, map[string][]byte{"Native": a}, identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	const cSource = `module C; profile Core; import B;
int UseStats() { ConceptXmlStats stats = ConceptXmlRoundTripStats(ConceptXmlStats{1, 2}); return StatsChildren(stats); }`
	artifacts := map[string][]byte{"Native": a, "B": b}
	if _, err := ParseWithSemanticModulesForNative("C.concept", cSource, artifacts, identity); err != nil {
		t.Fatal(err)
	}
	const explain = `module Explain; profile Core; import Native;
int Check() { Assert.Concept<CAbiValue>(ConceptXmlStats, "measured ABI"); return 1; }`
	proof, err := ExplainSourceWithNativeSemanticModules("Explain.concept", explain, 0, map[string][]byte{"Native": a}, identity)
	if err != nil || proof.Outcome != FactProven || !strings.Contains(RenderProofVerbose(proof), "origin=NativeToolchainProbe") || !strings.Contains(RenderProofVerbose(proof), identity.BuildInputHash) {
		t.Fatalf("matching proof lost measured provenance: %v %+v", err, proof)
	}
	if _, err := ParseWithSemanticModules("C.concept", cSource, artifacts); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("unvalidated identity reused: %v", err)
	}
	mutated := identity
	mutated.CompilerVersion += " changed"
	if _, err := ParseWithSemanticModulesForNative("C.concept", cSource, artifacts, mutated); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("toolchain mutation reused evidence: %v", err)
	}
	mutated = identity
	mutated.BuildInputHash = strings.Repeat("0", len(identity.BuildInputHash))
	if _, err := ParseWithSemanticModulesForNative("C.concept", cSource, artifacts, mutated); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("input mutation reused evidence: %v", err)
	}
	mutated = identity
	mutated.TargetTriple += "-other"
	if _, err := ParseWithSemanticModulesForNative("C.concept", cSource, artifacts, mutated); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("target mutation reused evidence: %v", err)
	}
	var serialized SemanticModuleArtifact
	if err := json.Unmarshal(a, &serialized); err != nil {
		t.Fatal(err)
	}
	contradictory := serialized
	copyReport := *serialized.NativeABI
	copyReport.Evidence = append([]NativeABIEvidence{}, serialized.NativeABI.Evidence...)
	copyReport.Evidence[0].Size = 12
	contradictory.NativeABI = &copyReport
	contradictory.ContentSHA256, err = semanticArtifactHash(contradictory)
	if err != nil {
		t.Fatal(err)
	}
	badArtifact, err := json.Marshal(contradictory)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadSemanticModuleArtifact(badArtifact); err == nil || !strings.Contains(err.Error(), "native probe size 12") || !strings.Contains(err.Error(), "Concept declaration size 8") {
		t.Fatalf("contradictory origins were not diagnosed: %v", err)
	}
	serialized.NativeABI = nil
	serialized.ContentSHA256, err = semanticArtifactHash(serialized)
	if err != nil {
		t.Fatal(err)
	}
	old, err := json.Marshal(serialized)
	if err != nil {
		t.Fatal(err)
	}
	const call = `module C; profile Core; import Native; int UseStats() { ConceptXmlStats stats = ConceptXmlRoundTripStats(ConceptXmlStats{1, 2}); return stats.children; }`
	if _, err := ParseWithSemanticModulesForNative("C.concept", call, map[string][]byte{"Native": old}, identity); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_MISSING") {
		t.Fatalf("old artifact fabricated ABI proof: %v", err)
	}
	const redeclared = `module C; profile Core; import Native;
extern "C" ConceptXmlStats LocalRoundTrip(ConceptXmlStats stats);
int Use() { ConceptXmlStats value = LocalRoundTrip(ConceptXmlStats{1, 2}); return value.children; }`
	if _, err := ParseWithSemanticModulesForNative("C.concept", redeclared, map[string][]byte{"Native": old}, identity); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_MISSING") {
		t.Fatalf("local extern declaration bypassed missing imported evidence: %v", err)
	}
	unknown, err := ExplainSourceWithNativeSemanticModules("Explain.concept", explain, 0, map[string][]byte{"Native": old}, identity)
	if err != nil || unknown.Outcome != FactUnknown {
		t.Fatalf("old artifact proof must be Unknown: %v %+v", err, unknown)
	}
	if bytes.Contains(a, []byte("#include")) {
		t.Fatal("semantic artifact transported probe source")
	}
	body, err := os.ReadFile(filepath.Join(project.Root, ".native-build", "abi.json"))
	if err != nil {
		t.Fatal(err)
	}
	var projected NativeABIReport
	if err := json.Unmarshal(body, &projected); err != nil || !reflect.DeepEqual(projected, *parsed.NativeABI) {
		t.Fatalf("abi.json and semantic artifact report diverged: %v", err)
	}
}

func TestR7j1ZeroArgumentGeneratedCIsStrictC11(t *testing.T) {
	const source = `module Prototype; profile Core;
enum Signal { Start, }
[[repr(C)]] record struct Pair { int x; int y; }
extern "C" Pair GetPair();
int Main() { Pair pair = GetPair(); Signal signal = Signal::Start; return pair.x; }`
	module, err := Parse("Prototype.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["prototype.generated.h"])
	body := string(outputs["prototype.generated.c"])
	for _, expected := range []string{"GetPair(void)", "concept_prototype_main(void)"} {
		if !strings.Contains(header+body, expected) {
			t.Fatalf("missing strict prototype %s", expected)
		}
	}
	if strings.Contains(body, "_make_start() {") {
		t.Fatal("zero-argument enum helper uses unspecified arguments")
	}
	dir := t.TempDir()
	for name, data := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, compiler := range []string{"clang", "gcc"} {
		if _, err := exec.LookPath(compiler); err != nil {
			continue
		}
		command := exec.Command(compiler, "-std=c11", "-pedantic", "-pedantic-errors", "-Wstrict-prototypes", "-Werror", "-fsyntax-only", filepath.Join(dir, "prototype.generated.c"))
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", compiler, err, output)
		}
	}
}

func TestR7j1NativeHeaderAndFlagMutationInvalidatesArtifact(t *testing.T) {
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	root := t.TempDir()
	files := map[string]string{
		"manifest.concept": "// native identity fixture\n",
		"bridge.h":         "#include <cstdint>\n#ifdef ABI_WIDE\ntypedef struct Pair { int32_t x; int64_t y; } Pair;\n#else\ntypedef struct Pair { int32_t x; int32_t y; } Pair;\n#endif\n",
		"bridge.cpp":       "#include \"bridge.h\"\n",
		"Native.concept":   "module Native; profile Core; [[repr(C)]] record struct Pair { int x; int y; } extern \"C\" Pair Exchange(Pair value);\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	project := NativeProject{Name: "ABIIdentity", Toolchain: "Clang", Root: root,
		Targets:    []NativeTarget{{Name: "bridge", Language: "Cpp", Standard: "Cpp17", Kind: "StaticLibrary", Output: "bridge.a", Sources: []string{"bridge.cpp"}, Includes: []string{"."}}},
		Companions: []string{"Native.concept"},
		ABI:        []NativeABIClaim{{Header: "bridge.h", TypeName: "Pair", Companion: "Native.concept", Size: 8, Alignment: 4, Fields: []string{"x", "y"}, Offsets: []int{0, 4}}},
	}
	a, err := CompileNativeSemanticModule(project, "Native.concept", nil)
	if err != nil {
		t.Fatal(err)
	}
	report, _, err := LoadSemanticModuleArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	identity := report.NativeABI.Identity
	plan, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "bridge.a")
	if err := os.WriteFile(outputPath, []byte("first archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	build := NativeBuildResult{Plan: plan, NativeOutputHashes: map[string]string{"bridge": nativeDigest([]byte("first archive"))}}
	if err := ValidateNativeBuildOutputs(project, build); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte("different archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNativeBuildOutputs(project, build); err == nil || !strings.Contains(err.Error(), "NATIVE_BUILD_OUTPUT_MISMATCH") {
		t.Fatalf("different native archive escaped validation: %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("first archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	const consumer = `module Consumer; profile Core; import Native;
Pair Use(Pair value) { return Exchange(value); }`
	if _, err := ParseWithSemanticModulesForNative("Consumer.concept", consumer, map[string][]byte{"Native": a}, identity); err != nil {
		t.Fatal(err)
	}
	changedHeader := "#include <cstdint>\ntypedef struct Pair { int32_t x; int64_t y; } Pair;\n"
	if err := os.WriteFile(filepath.Join(root, "bridge.h"), []byte(changedHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := CurrentNativeABIIdentity(project)
	if err != nil {
		t.Fatal(err)
	}
	if changed.BuildInputHash == identity.BuildInputHash {
		t.Fatal("header mutation kept native input identity")
	}
	if _, err := ParseWithSemanticModulesForNative("Consumer.concept", consumer, map[string][]byte{"Native": a}, changed); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("changed header reused evidence: %v", err)
	}
	if _, err := EnsureNativeABIReport(project); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_MISMATCH") {
		t.Fatalf("changed native layout escaped reprobe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bridge.h"), []byte(files["bridge.h"]), 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := CurrentNativeABIIdentity(project)
	if err != nil || restored != identity {
		t.Fatalf("restored input did not recover identity: %v", err)
	}
	if _, err := ParseWithSemanticModulesForNative("Consumer.concept", consumer, map[string][]byte{"Native": a}, restored); err != nil {
		t.Fatal(err)
	}
	project.Targets[0].Defines = []NativeDefine{{Name: "ABI_WIDE", Value: "1", HasValue: true}}
	flagged, err := CurrentNativeABIIdentity(project)
	if err != nil {
		t.Fatal(err)
	}
	if flagged.BuildInputHash == identity.BuildInputHash {
		t.Fatal("ABI flag mutation kept native input identity")
	}
	if _, err := ParseWithSemanticModulesForNative("Consumer.concept", consumer, map[string][]byte{"Native": a}, flagged); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_EVIDENCE_STALE") {
		t.Fatalf("changed flag reused evidence: %v", err)
	}
	if _, err := EnsureNativeABIReport(project); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_MISMATCH") {
		t.Fatalf("ABI-relevant define escaped reprobe: %v", err)
	}
}
