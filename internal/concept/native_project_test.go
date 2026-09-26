package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeConceptArtifactsRepeatByteIdentically(t *testing.T) {
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	project, err := LoadNativeProject(filepath.Join("..", "..", "tests", "dogfood", "tinyxml2"))
	if err != nil {
		t.Fatal(err)
	}
	requireTinyXML2Submodule(t, project.Root)
	testPath := filepath.Join(project.Root, "tests", "native.concept_test")
	testSource, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, identity, err := BuildNativeCompanionArtifacts(project)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithSemanticModulesForNative(filepath.ToSlash(testPath), string(testSource), artifacts, identity)
	if err != nil {
		t.Fatal(err)
	}
	proofPath := filepath.Join(project.Root, "proofs", "declared_allocation.concept")
	proofSource, err := os.ReadFile(proofPath)
	if err != nil {
		t.Fatal(err)
	}
	unknownPath := filepath.Join(project.Root, "proofs", "unknown_effect.concept")
	unknownSource, err := os.ReadFile(unknownPath)
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := ExplainSourceWithNativeSemanticModules(filepath.ToSlash(unknownPath), string(unknownSource), 0, artifacts, identity)
	if err != nil || unknown.Outcome != FactUnknown {
		t.Fatalf("unclaimed native effect was not Unknown: %v %+v", err, unknown)
	}
	schemaPath := filepath.Join(project.Root, "proofs", "schema_disproven.concept")
	schemaSource, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := ExplainSourceWithNativeSemanticModules(filepath.ToSlash(schemaPath), string(schemaSource), 0, artifacts, identity)
	if err != nil || schema.Outcome != FactDisproven {
		t.Fatalf("missing native schema was not Disproven: %v %+v", err, schema)
	}
	abiProofPath := filepath.Join(project.Root, "proofs", "abi_value.concept")
	abiProofSource, err := os.ReadFile(abiProofPath)
	if err != nil {
		t.Fatal(err)
	}
	var planBaseline, companionBaseline, generatedBaseline, proofBaseline, abiBaseline, abiProofBaseline, probeBaseline []byte
	for run := 0; run < 100; run++ {
		plan, err := NativeBuildPlan(project)
		if err != nil {
			t.Fatal(err)
		}
		planBody, _ := MarshalNativePlan(plan)
		artifact, err := CompileNativeSemanticModule(project, "concept/Native.concept", nil)
		if err != nil {
			t.Fatal(err)
		}
		generated, err := Generate(module, testSource)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := ExplainSourceWithNativeSemanticModules(filepath.ToSlash(proofPath), string(proofSource), 0, artifacts, identity)
		if err != nil {
			t.Fatal(err)
		}
		if graph.Outcome != FactDisproven || !strings.Contains(RenderProofVerbose(graph), "origin=DeclaredForeign") {
			t.Fatal("allocation proof lost foreign provenance")
		}
		proof, err := SerializeProof(graph)
		if err != nil {
			t.Fatal(err)
		}
		abiBody, err := os.ReadFile(filepath.Join(project.Root, ".native-build", "abi.json"))
		if err != nil {
			t.Fatal(err)
		}
		abiGraph, err := ExplainSourceWithNativeSemanticModules(filepath.ToSlash(abiProofPath), string(abiProofSource), 0, artifacts, identity)
		if err != nil || abiGraph.Outcome != FactProven {
			t.Fatalf("ABI proof changed: %v %+v", err, abiGraph)
		}
		abiProof, err := SerializeProof(abiGraph)
		if err != nil {
			t.Fatal(err)
		}
		probe := []byte(nativeABIProbeSource(project.ABI[0]))
		var c []byte
		for name, body := range generated {
			if strings.HasSuffix(name, ".generated.c") {
				c = body
			}
		}
		if run == 0 {
			planBaseline, companionBaseline, generatedBaseline, proofBaseline, abiBaseline, abiProofBaseline, probeBaseline = planBody, artifact, c, proof, abiBody, abiProof, probe
			continue
		}
		if !bytes.Equal(planBody, planBaseline) || !bytes.Equal(artifact, companionBaseline) || !bytes.Equal(c, generatedBaseline) || !bytes.Equal(proof, proofBaseline) || !bytes.Equal(abiBody, abiBaseline) || !bytes.Equal(abiProof, abiProofBaseline) || !bytes.Equal(probe, probeBaseline) {
			t.Fatalf("Concept-side artifact changed on run %d", run+1)
		}
	}
}

func TestNativePlanExecutesStructuredCArgumentsWithSpaces(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang unavailable")
	}
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	root := t.TempDir()
	for _, dir := range []string{"src files", "include paths"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range map[string]string{
		"manifest.concept":       "fixture",
		"src files/main.c":       "#include \"answer.h\"\nint main(void) { return ANSWER == 42 ? 0 : 1; }\n",
		"include paths/answer.h": "#define ANSWER 42\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	project := NativeProject{Name: "space-paths", Toolchain: "Clang", Root: root, Targets: []NativeTarget{{Name: "app", Language: "C", Standard: "C11", Kind: "Executable", Output: ".native-build/app.exe", Sources: []string{"src files/main.c"}, Includes: []string{"include paths"}}}}
	first, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := MarshalNativePlan(first)
	b, _ := MarshalNativePlan(second)
	if string(a) != string(b) {
		t.Fatal("native plan is not deterministic")
	}
	if first.Commands[0].Program != "clang" || first.Commands[0].Args[2] != "src files/main.c" {
		t.Fatalf("incorrect C command: %+v", first.Commands[0])
	}
	if _, err := RunNativeBuild(project, first); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(filepath.Join(root, ".native-build", "app.exe")).CombinedOutput(); err != nil {
		t.Fatalf("C executable failed: %v\n%s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, "include paths", "answer.h"), []byte("#define ANSWER 43\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	if changed.BuildInputHash == first.BuildInputHash {
		t.Fatal("header change did not alter build input identity")
	}
}

func TestNativeABIProbeRejectsWrongLayout(t *testing.T) {
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	project, err := LoadNativeProject(filepath.Join("..", "..", "tests", "dogfood", "tinyxml2"))
	if err != nil {
		t.Fatal(err)
	}
	requireTinyXML2Submodule(t, project.Root)
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	project.ABI[0].Size = 24
	err = CheckNativeABI(project)
	if err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_MISMATCH") || !strings.Contains(err.Error(), "size 24") || !strings.Contains(err.Error(), "size 8") {
		t.Fatalf("wrong layout was not diagnosed: %v", err)
	}
}

func requireTinyXML2Submodule(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "upstream", "tinyxml2.h")); os.IsNotExist(err) {
		t.Skip("TinyXML2 submodule not initialized; run git submodule update --init")
	} else if err != nil {
		t.Fatal(err)
	}
}
