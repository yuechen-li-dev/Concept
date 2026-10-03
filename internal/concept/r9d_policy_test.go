package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const r9dManifest = `module R9d.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept PreferMatchOverElseIfLadder<declaration F> { requires compiler.NoMatchShapedElseIfLadder(F); }
comptime LintPolicy MatchPreference = LintPolicy{"PreferMatchOverElseIfLadder", "warning", "FunctionDeclaration", ""};
`

func TestR9dMatchPreference(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         int
		finite       bool
	}{
		{"negative integer", `int F(int x) { if (x == -1) {} else if (x == -2) {} else if (x == -3) {} return 0; }`, 1, false},
		{"comptime", `comptime int F(int x) { if (x == 1) { return 1; } else if (x == 2) { return 2; } else if (x == 3) { return 3; } return 0; }`, 1, false},
		{"template body is not caller body", `template <typename T> int Hidden(int x) { if (x == 1) { return 1; } else if (x == 2) { return 2; } else if (x == 3) { return 3; } return 0; } int F(int x) { return Hidden<int>(x); }`, 0, false},
		{"callable body is not enclosing body", `int F(int x) { const auto choose = callback(int y) { if (y == 1) { return 1; } else if (y == 2) { return 2; } else if (y == 3) { return 3; } return 0; }; return choose(x); }`, 0, false},
		{"integer", `int F(int x) { if (x == 1) {} else if (x == 2) {} else if (x == 3) {} return 0; }`, 1, false},
		{"reversed parenthesized", `int F(int x) { if (1 == (x)) {} else if ((x) == 2) {} else if (3 == ((x))) {} return 0; }`, 1, false},
		{"enum", `enum Kind { A, B, C } int F(Kind x) { if (x == Kind::A) {} else if (Kind::B == x) {} else if (x == Kind::C) {} return 0; }`, 1, true},
		{"field", `struct Packet { int tag; } int F(Packet p) { if (p.tag == 1) {} else if (p.tag == 2) {} else if ((p).tag == 3) {} return 0; }`, 1, false},
		{"two comparisons", `int F(int x) { if (x == 1) {} else if (x == 2) {} else {} return 0; }`, 0, false},
		{"guards", `int F(int x, int y) { if (x == 1) { return 1; } else if (y == 2) { return 2; } else if (x < y) { return 3; } return 0; }`, 0, false},
		{"early returns categorical", `int F(int x) { if (x == 1) { return 1; } else if (x == 2) { return 2; } else if (x == 3) { return 3; } return 0; }`, 1, false},
		{"duplicate values", `int F(int x) { if (x == 1) {} else if (x == 0x1) {} else if (x == 3) {} return 0; }`, 0, false},
		{"side effects", `int Next() { return 1; } int F() { if (Next() == 1) {} else if (Next() == 2) {} else if (Next() == 3) {} return 0; }`, 0, false},
		{"effectful case", `int Next() { return 1; } int F(int x) { if (x == 1) {} else if (x == Next()) {} else if (x == 3) {} return 0; }`, 0, false},
		{"existing match", `int F(int x) { return match (x) { 1 => 1, 2 => 2, 3 => 3, _ => 0 }; }`, 0, false},
		{"suffix of guards", `int F(int x, int y) { if (y == 0) {} else if (x == 1) {} else if (x == 2) {} else if (x == 3) {} return 0; }`, 0, false},
		{"two ladders one function", `int F(int x) { if (x == 1) {} else if (x == 2) {} else if (x == 3) {} if (x == 4) {} else if (x == 5) {} else if (x == 6) {} return 0; }`, 2, false},
		{"shadowed nested binding", `int F(int x) { if (x == 1) { int x = 9; if (x == 4) {} else if (x == 5) {} else if (x == 6) {} } else if (x == 2) {} else if (x == 3) {} return 0; }`, 2, false},
		{"explicit else block", `int F(int x) { if (x == 1) {} else { if (x == 2) {} else if (x == 3) {} } return 0; }`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "program.concept")
			source := "module R9d.Program; profile Core;\n" + tc.source
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			// No manifest: syntax and semantics are valid, no preference finding.
			findings, err := LintPath(path, nil)
			if err != nil || len(findings) != 0 {
				t.Fatalf("absent policy: %+v %v", findings, err)
			}
			if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(r9dManifest), 0600); err != nil {
				t.Fatal(err)
			}
			findings, err = LintPath(path, nil)
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings: %+v %v; want %d", findings, err, tc.want)
			}
			for _, finding := range findings {
				if finding.Severity != LintWarning || finding.Outcome != FactDisproven || !strings.Contains(finding.Message, "manifest.concept") || strings.Contains(finding.Message, "exhaustiveness") != tc.finite {
					t.Fatalf("finding: %+v", finding)
				}
			}
			if tc.name == "shadowed nested binding" {
				module, err := Parse(path, source)
				if err != nil {
					t.Fatal(err)
				}
				_, concepts, err := LoadProjectPolicies(filepath.Join(root, "manifest.concept"), nil)
				if err != nil {
					t.Fatal(err)
				}
				module.Concepts = concepts
				env, err := analyzeModule(module)
				if err != nil {
					t.Fatal(err)
				}
				if len(env.ifLadders) < 2 || env.ifLadders[0].SubjectID == env.ifLadders[1].SubjectID {
					t.Fatal("shadowed storage acquired same bound identity")
				}
			}
		})
	}
}

func TestR9dElseIfWin64NativeExecution(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("executable-memory qualification requires Windows AMD64")
	}
	const source = `module R9d.Native; profile Core;
int Decode(int x) { if (x == 1) { return 10; } else if (x == 2) { return 20; } else if (x == 3) { return 30; } else { return 40; } }
int CallDecode(int x) { return Decode(x); }
`
	checked, err := Parse("native_ladder.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(checked)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	data.WriteString("static const unsigned char artifact[]={")
	for _, b := range artifact {
		fmt.Fprintf(&data, "%d,", b)
	}
	data.WriteString("};\n")
	for i, fn := range machine.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", fn.Name, i)
	}
	harness := `#include "amd64.generated.h"
#include "oracle.c"
#include <windows.h>
#include <string.h>
#include <stdio.h>
/* ARTIFACT */
static unsigned char code[65536];
static concept_native_image image;
int main(void) {
  concept_readonly_span_byte input = {artifact, sizeof artifact};
  concept_span_byte output = {code, sizeof code};
  if (concept_standard__backend__amd64_emit_module(input, output, &image).tag) return 1;
  void *base = VirtualAlloc(NULL, (size_t)image.byteCount, MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE);
  if (!base) return 2;
  memcpy(base, code, (size_t)image.byteCount);
  DWORD old;
  if (!VirtualProtect(base, (size_t)image.byteCount, PAGE_EXECUTE_READ, &old) || !FlushInstructionCache(GetCurrentProcess(), base, (size_t)image.byteCount)) return 3;
  typedef int (*Fn)(int);
  void *entry = (unsigned char *)base + image.symbols.data[ORD_CallDecode].offset;
  Fn call = NULL; memcpy(&call, &entry, sizeof call);
  for (int x = -2; x <= 5; ++x) if (call(x) != concept_r9d__native_call_decode(x)) return 4;
  for (int i = 0; i < image.byteCount; ++i) printf("%02x", code[i]);
  puts("");
  return VirtualFree(base, 0, MEM_RELEASE) ? 0 : 5;
}`
	harness = strings.Replace(harness, "/* ARTIFACT */", data.String(), 1)
	var normal string
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		outputs := backendTestOutputs(t, policy.Verify)
		oracle, err := GenerateForTargetWithPolicy(checked, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		for name, body := range oracle {
			if strings.HasSuffix(name, ".generated.c") {
				outputs["oracle.c"] = body
			} else {
				outputs[name] = body
			}
		}
		result := runFoundationNativeHarnessOutput(t, outputs, "r9d_native_host.c", harness, "-pedantic-errors", "-O2")
		if normal == "" {
			normal = result
		} else if result != normal {
			t.Fatal("Normal/Verify native bytes differ")
		}
	}
}

func TestR9dPolicyDeterminismSeverityAndExplain(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "program.concept")
	source := `module R9d.Program; profile Core;
int F(int x) {
    if (x == 1) { return 10; }
    else if (x == 2) { return 20; }
    else if (x == 3) { return 30; }
    else { return 40; }
}`
	manifestPath := filepath.Join(root, "manifest.concept")
	for path, body := range map[string]string{path: source, manifestPath: r9dManifest} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	baseline, err := LintPath(path, nil)
	if err != nil || len(baseline) != 1 {
		t.Fatalf("%+v %v", baseline, err)
	}
	if baseline[0].Site.Line != 3 {
		t.Fatalf("anchor: %+v", baseline[0].Site)
	}
	encoded, _ := json.Marshal(baseline)
	for i := 0; i < 100; i++ {
		findings, err := LintPath(path, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(findings)
		if !bytes.Equal(encoded, got) {
			t.Fatalf("drift at %d", i)
		}
	}
	graph, err := ExplainPolicy(path, "PreferMatchOverElseIfLadder", "F", nil)
	if err != nil || graph.Outcome != FactDisproven || !strings.Contains(RenderProofVerbose(graph), "3-branch") || !strings.Contains(RenderProofVerbose(graph), "manifest.concept") {
		t.Fatalf("explain: %+v %v", graph, err)
	}
	before, err := CompileSemanticModule(path, source, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(r9dManifest, `"warning"`, `"error"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(path, nil)
	if err != nil || !HasLintErrors(findings) {
		t.Fatalf("error severity: %+v %v", findings, err)
	}
	after, err := CompileSemanticModule(path, source, nil)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("policy changed semantic artifact: %v", err)
	}
	// Existing disabled surface is absence of the immutable activation value.
	if err := os.WriteFile(manifestPath, []byte(strings.Split(r9dManifest, "comptime LintPolicy")[0]), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err = LintPath(path, nil)
	if err != nil || len(findings) != 0 {
		t.Fatalf("disabled: %+v %v", findings, err)
	}
}

func TestR9dElseIfFormatting(t *testing.T) {
	source := runtimeTestSource(t, "runtime.concept_test")
	for _, style := range []string{"same-line", "allman"} {
		opts := DefaultFormatOptions()
		opts.BraceStyle = style
		formatted, err := FormatSource("runtime.concept", source, opts)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(formatted, "else if (") || strings.Contains(formatted, "else\n    if") {
			t.Fatalf("split else-if:\n%s", formatted)
		}
		if style == "allman" && !strings.Contains(formatted, "}\n    else if (x == 2)\n    {") {
			t.Fatalf("Allman ladder:\n%s", formatted)
		}
		for i := 0; i < 100; i++ {
			got, err := FormatSource("runtime.concept", formatted, opts)
			if err != nil || got != formatted {
				t.Fatalf("format drift %d: %v", i, err)
			}
		}
	}
	commented := strings.Replace(source, "else if (x == 2)", "else /* preserve */ if (x == 2)", 1)
	opts := DefaultFormatOptions()
	opts.BraceStyle = "allman"
	formatted, err := FormatSource("comments.concept", commented, opts)
	if err != nil || !strings.Contains(formatted, "/* preserve */") {
		t.Fatalf("comment lost: %v", err)
	}
	again, err := FormatSource("comments.concept", formatted, opts)
	if err != nil || again != formatted {
		t.Fatalf("comment drift: %v", err)
	}

	module, err := Parse("runtime.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "runtime.generated.c")

}

func TestR9dDogfoodProject(t *testing.T) {
	findings, err := LintPath("../../tests/lint/r9d", nil)
	if err != nil || len(findings) != 2 {
		t.Fatalf("dogfood: %+v %v", findings, err)
	}
	for _, finding := range findings {
		if finding.Name != "Decode" && finding.Name != "Opcode" {
			t.Fatalf("guard/match warned: %+v", finding)
		}
	}
}

func TestR9dObservationComposesWithExistingPolicy(t *testing.T) {
	root := t.TempDir()
	manifest := strings.Replace(r9dManifest, "requires compiler.NoMatchShapedElseIfLadder(F);", "requires compiler.NoMatchShapedElseIfLadder(F); requires compiler.CamelCase(F);", 1)
	source := "module R9d.Composed; profile Core; int F(int x) { if (x == 1) {} else if (x == 2) {} else if (x == 3) {} return 0; }"
	for name, body := range map[string]string{"manifest.concept": manifest, "program.concept": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := LintPath(root, nil)
	if err != nil || len(findings) != 2 {
		t.Fatalf("composed policy lost naming or ladder failure: %+v %v", findings, err)
	}
}

func TestR9dUnavailableBodyObservationIsUnknown(t *testing.T) {
	module, err := Parse("empty.concept", "module R9d.Empty; profile Core; int F(int x) { return x; }")
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	declaration := ProjectDeclarationSubjects(module)[0]
	for _, subject := range ProjectDeclarationSubjects(module) {
		if subject.Kind == FunctionDeclaration {
			declaration = subject
		}
	}
	var graph ProofGraph
	outcome, _ := projectControlFlowAnalysis(env, &graph, "", declaration)
	if outcome != FactUnknown {
		t.Fatalf("missing observation claimed %s", outcome)
	}
}
