package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func r8fSource(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "libraries", "Golden", "Differentiators"}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDifferentiatorStrictC11BothCompilers(t *testing.T) {
	irSource := r8fSource(t, "Compiler", "IR.concept")
	irArtifact, err := CompileSemanticModule("IR.concept", irSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		folder, file string
		dependencies map[string][]byte
	}{
		{"Agents", "Squad.concept", nil},
		{"Async", "Journal.concept", nil},
		{"Hpc", "Dot.concept", nil},
		{"Mechanics", "Stress.concept", nil},
		{"Compiler", "IRTools.concept", map[string][]byte{"Golden.Differentiators.Compiler.IR": irArtifact}},
	} {
		t.Run(target.folder, func(t *testing.T) {
			source := r8fSource(t, target.folder, target.file)
			module, err := ParseWithSemanticModules(target.file, source, target.dependencies)
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if target.folder == "Compiler" {
				irModule, err := Parse("IR.concept", irSource)
				if err != nil {
					t.Fatal(err)
				}
				irOutputs, err := Generate(irModule, []byte(irSource))
				if err != nil {
					t.Fatal(err)
				}
				for name, data := range irOutputs {
					outputs[name] = data
				}
			}
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			var generated []string
			for name := range outputs {
				if strings.HasSuffix(name, ".generated.c") {
					generated = append(generated, filepath.Join(dir, name))
				}
			}
			harness := filepath.Join(dir, "harness.c")
			if err := os.WriteFile(harness, []byte("int main(void) { return 0; }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			for _, compilerName := range []string{"gcc", "clang"} {
				compiler, err := exec.LookPath(compilerName)
				if err != nil {
					t.Fatalf("%s unavailable: %v", compilerName, err)
				}
				binary := filepath.Join(dir, compilerName+"-strict-c11.exe")
				args := append([]string{"-std=c11", "-pedantic-errors", "-I", dir}, generated...)
				args = append(args, harness, "-o", binary)
				if output, err := nativeCommand(t, compiler, args...).CombinedOutput(); err != nil {
					t.Fatalf("%s strict C11: %v\n%s", compilerName, err, output)
				}
				if output, err := nativeCommand(t, binary).CombinedOutput(); err != nil {
					t.Fatalf("%s native execution: %v\n%s", compilerName, err, output)
				}
			}
		})
	}
}

func TestDifferentiatorDeterminism100(t *testing.T) {
	if testing.Short() {
		t.Skip("100-run artifact and C identity is a full gate")
	}
	irSource := r8fSource(t, "Compiler", "IR.concept")
	irArtifact, err := CompileSemanticModule("IR.concept", irSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, representative := range []struct {
		folder, file, module string
		dependencies         map[string][]byte
	}{
		{"Hpc", "Dot.concept", "Golden.Differentiators.Hpc.Dot", nil},
		{"Agents", "Squad.concept", "Golden.Differentiators.Agents.Squad", nil},
		{"Async", "Journal.concept", "Golden.Differentiators.Async.Journal", nil},
		{"Compiler", "IRTools.concept", "Golden.Differentiators.Compiler.IRTools", map[string][]byte{"Golden.Differentiators.Compiler.IR": irArtifact}},
	} {
		t.Run(representative.folder, func(t *testing.T) {
			source := r8fSource(t, representative.folder, representative.file)
			var baselineArtifact []byte
			var baselineC []byte
			for i := 0; i < 100; i++ {
				artifact, err := CompileSemanticModule(representative.file, source, representative.dependencies)
				if err != nil {
					t.Fatal(err)
				}
				module, err := ParseWithSemanticModules(representative.file, source, representative.dependencies)
				if err != nil {
					t.Fatal(err)
				}
				outputs, err := Generate(module, []byte(source))
				if err != nil {
					t.Fatal(err)
				}
				var generatedC []byte
				for name, body := range outputs {
					if strings.HasSuffix(name, ".generated.c") {
						generatedC = body
					}
				}
				if i == 0 {
					baselineArtifact, baselineC = artifact, generatedC
				} else if !bytes.Equal(artifact, baselineArtifact) || !bytes.Equal(generatedC, baselineC) {
					t.Fatalf("semantic artifact or C drifted on run %d", i+1)
				}
			}
		})
	}
}

func TestDifferentiatorClosedGenericSpanHasNativeDeclarations(t *testing.T) {
	source := r8fSource(t, "Hpc", "Dot.concept")
	module, err := Parse("Dot.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["dot.generated.h"])
	generatedC := string(outputs["dot.generated.c"])
	if strings.Contains(generatedC, "if (true)") || strings.Contains(generatedC, "if (false)") {
		t.Fatal("closed representation choice remains in generated C")
	}
	for _, name := range []string{"concept_readonly_span_float", "concept_readonly_span_double"} {
		if !strings.Contains(header, "} "+name+";") {
			t.Fatalf("closed generic span %s lacks a C declaration", name)
		}
	}
	runFoundationNativeHarness(t, outputs, "dot_harness.c", `#include "dot.generated.h"
int main(void) { return concept_golden__differentiators__hpc__dot_dot_float() == 20.0f && concept_golden__differentiators__hpc__dot_dot_double() == 20.0 ? 0 : 1; }`)
}

func TestDifferentiatorHotPathPolicyProvesDotKernel(t *testing.T) {
	project := filepath.Join("..", "..", "libraries", "Golden", "Differentiators")
	policies, concepts, err := LoadProjectPolicies(filepath.Join(project, "manifest.concept"), []string{filepath.Join("..", "..", "libraries")})
	if err != nil {
		t.Fatal(err)
	}
	source := r8fSource(t, "Hpc", "Dot.concept")
	module, err := Parse("Dot.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	bound := module
	bound.Concepts = append(append([]ConceptDecl{}, module.Concepts...), concepts...)
	env, err := analyzeModule(bound)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range policies {
		if policy.Identity != "HotPathPolicy" || policy.Name != "DotFloat" {
			continue
		}
		for _, subject := range ProjectDeclarationSubjects(module) {
			if subject.Name == "DotFloat" {
				proof, err := buildPolicyProof(env, policy, subject)
				if err != nil {
					t.Fatal(err)
				}
				if proof.Outcome != FactProven {
					t.Fatalf("DotFloat policy is %s", proof.Outcome)
				}
				return
			}
		}
	}
	t.Fatal("configured DotFloat hot-path policy did not bind")
}

func TestDifferentiatorGeneratedIRSurvivesArtifactOnlyConsumer(t *testing.T) {
	ir := r8fSource(t, "Compiler", "IR.concept")
	irArtifact, err := CompileSemanticModule("IR.concept", ir, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{"Golden.Differentiators.Compiler.IR": irArtifact}
	toolsSource := r8fSource(t, "Compiler", "IRTools.concept")
	toolsArtifact, err := CompileSemanticModule("IRTools.concept", toolsSource, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["Golden.Differentiators.Compiler.IRTools"] = toolsArtifact
	const consumer = `module Differentiator.IRConsumer; profile Core;
import Golden.Differentiators.Compiler.IR;
import Golden.Differentiators.Compiler.IRTools;
bool Check() {
    Instruction instruction = Instruction{Opcode::Add, NodeId{0}, NodeId{1}, 0};
    return ValidArity(ref const instruction) and OperandCount(ref const instruction) == 2;
}`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outputs["consumer.generated.c"]), "unresolved_call") {
		t.Fatal("artifact-only generated verifier call is unresolved")
	}
	found := false
	for _, subject := range DeclarationSubjects(module) {
		if subject.Name == "OperandCount" && subject.Provenance == DeclarationGenerated {
			found = true
		}
	}
	if !found {
		t.Fatal("generated verifier provenance was lost through artifacts")
	}
}

func TestDifferentiatorArtifactOnlyConsumers(t *testing.T) {
	for _, target := range []struct {
		folder, file, module, consumer string
	}{
		{"Hpc", "Dot.concept", "Golden.Differentiators.Hpc.Dot", `module Differentiator.HpcConsumer; profile Core; import Golden.Differentiators.Hpc.Dot;
double Main() { return DotDouble(); }`},
		{"Agents", "Squad.concept", "Golden.Differentiators.Agents.Squad", `module Differentiator.AgentConsumer; profile Core; import Golden.Differentiators.Agents.Squad;
int Main() { instance SquadAgent scout(Observation{0.0, 1.0, 0.0, 0}, Policy::Engage, 0, 0, 0); Step(scout, Act); Step(scout, Act); return scout.state.patrolSteps; }`},
		{"Async", "Journal.concept", "Golden.Differentiators.Async.Journal", `module Differentiator.AsyncConsumer; profile Core; import Golden.Differentiators.Async.Journal;
int Main() { return Drive(4)!; }`},
	} {
		t.Run(target.folder, func(t *testing.T) {
			source := r8fSource(t, target.folder, target.file)
			artifact, err := CompileSemanticModule(target.file, source, nil)
			if err != nil {
				t.Fatal(err)
			}
			module, err := ParseWithSemanticModules("consumer.concept", target.consumer, map[string][]byte{target.module: artifact})
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, []byte(target.consumer))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(outputs["consumer.generated.c"]), "unresolved_call") {
				t.Fatal("artifact-only call is unresolved")
			}
		})
	}
}

func TestDifferentiatorAsyncEarlyReturnFinishesOperation(t *testing.T) {
	source := r8fSource(t, "Async", "Journal.concept")
	module, err := Parse("Journal.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["journal.generated.c"])
	if !strings.Contains(generated, "concept_async_finish(async_operation") {
		t.Fatal("async error path did not finish its operation")
	}
	runFoundationNativeHarness(t, outputs, "journal_harness.c", `#include "journal.generated.h"
int main(void) { return 0; }`)
}

func TestDifferentiatorStaticRejections(t *testing.T) {
	for _, invalid := range []struct {
		folder, file, diagnostic string
	}{
		{"Mechanics", "dimensional-mismatch.concept.txt", "CV4615"},
		{"Mechanics", "shape-mismatch.concept.txt", "CV4620"},
		{"Async", "borrow-across-await.concept.txt", "ASYNC_PERSISTENT_REF_ESCAPE"},
		{"Async", "ignored-must-use.concept.txt", "MUST_USE_RESULT_IGNORED"},
		{"Compiler", "invalid-operand.concept.txt", "CV4025"},
		{"Compiler", "invalid-authored-pass.concept.txt", "CONCEPT_ASSERT_DISPROVEN"},
		{"Agents", "invalid-policy-target.concept.txt", "MACHINE_UNKNOWN_STATE"},
		{"Hpc", "unsupported-scalar.concept.txt", "CV4640"},
	} {
		t.Run(invalid.file, func(t *testing.T) {
			_, err := Parse(invalid.file, r8fSource(t, invalid.folder, invalid.file))
			if err == nil || !strings.Contains(err.Error(), invalid.diagnostic) {
				t.Fatalf("want %s, got %v", invalid.diagnostic, err)
			}
		})
	}
}
