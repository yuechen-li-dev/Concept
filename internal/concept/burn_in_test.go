package concept

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7d3IntegerLiteralBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		literal    string
		valid      bool
		cFragment  string
		diagnostic string
	}{
		{"int max", "int", "2147483647", true, "INT32_C(2147483647)", ""},
		{"int min", "int", "-2147483648", true, "(-INT32_C(2147483647) - INT32_C(1))", ""},
		{"int positive overflow", "int", "2147483648", false, "", "CV4644"},
		{"int negative overflow", "int", "-2147483649", false, "", "CV4644"},
		{"uint crosses signed max", "uint", "2147483648", true, "UINT32_C(2147483648)", ""},
		{"uint max", "uint", "4294967295", true, "UINT32_C(4294967295)", ""},
		{"uint overflow", "uint", "4294967296", false, "", "CV4644"},
		{"hex signed max", "int", "0x7FFFFFFF", true, "INT32_C(2147483647)", ""},
		{"hex uint high", "uint", "0x80000000", true, "UINT32_C(2147483648)", ""},
		{"hex uint max", "uint", "0xFFFFFFFF", true, "UINT32_C(4294967295)", ""},
		{"hex uint overflow", "uint", "0x100000000", false, "", "CV4644"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := fmt.Sprintf("profile Core; %s Value() { %s X = %s; return X; }", test.typeName, test.typeName, test.literal)
			module, err := Parse("boundary.concept", source)
			if !test.valid {
				if err == nil || !strings.Contains(err.Error(), test.diagnostic) {
					t.Fatalf("Parse() err=%v, want %s", err, test.diagnostic)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(): %v", err)
			}
			outputs, err := Generate(module, []byte(source))
			if err != nil {
				t.Fatalf("Generate(): %v", err)
			}
			body := string(outputs["boundary.generated.c"])
			if !strings.Contains(body, test.cFragment) {
				t.Fatalf("generated C omitted %q:\n%s", test.cFragment, body)
			}
		})
	}
}

func TestR7d3CallSiteContextualIntegerLiteral(t *testing.T) {
	source := `profile Core;
uint Select(uint X) { return X; }
uint Use() { return Select(4294967295); }
`
	module, err := Parse("call_literal.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if body := string(outputs["call_literal.generated.c"]); !strings.Contains(body, "UINT32_C(4294967295)") {
		t.Fatalf("call literal was not typed as uint:\n%s", body)
	}
}

func TestR7d3CPrecedenceBands(t *testing.T) {
	cases := map[string]string{
		"A ^ B & C":  "(A ^ (B & C))",
		"A | B & C":  "(A | (B & C))",
		"A | B ^ C":  "(A | (B ^ C))",
		"A << B + C": "u64_shl((uint64_t)(A), (uint64_t)((B + C))",
		"A & B + C":  "(A & (B + C))",
	}
	for expression, expected := range cases {
		source := fmt.Sprintf("profile Core; uint Read(uint A, uint B, uint C) { return %s; }", expression)
		module, err := Parse("precedence.concept", source)
		if err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
		if body := string(outputs["precedence.generated.c"]); !strings.Contains(body, expected) {
			t.Fatalf("%s did not lower as %s:\n%s", expression, expected, body)
		}
	}
}

func TestR7d3ComparisonEqualityAndBitwisePrecedence(t *testing.T) {
	var shape func(Expr) string
	shape = func(expr Expr) string {
		switch value := expr.(type) {
		case *BinaryExpr:
			return "(" + shape(value.Left) + " " + value.Op + " " + shape(value.Right) + ")"
		case *NameExpr:
			return value.Name
		default:
			return fmt.Sprintf("%T", expr)
		}
	}
	cases := map[string]string{
		"A == B & C": "((A == B) & C)",
		"A & B == C": "(A & (B == C))",
		"A < B == C": "((A < B) == C)",
		"A == B < C": "(A == (B < C))",
	}
	for expression, expected := range cases {
		source := fmt.Sprintf("profile Core; int Read(int A, int B, int C) { return %s; }", expression)
		module, err := parseSyntaxModule("precedence_syntax.concept", source)
		if err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
		actual := shape(module.Functions[0].Body.Statements[0].(*ReturnStmt).Value)
		if actual != expected {
			t.Fatalf("%s parsed as %s, want %s", expression, actual, expected)
		}
	}
}

func TestR7d3ConstantUBDiagnostics(t *testing.T) {
	tests := []struct {
		source string
		code   string
	}{
		{"profile Core; int Bad() { return 1 / 0; }", "CV4645"},
		{"profile Core; uint Bad(uint X) { return X << 32; }", "CV4646"},
		{"profile Core; int Bad() { return 2147483647 + 1; }", "CV4649"},
	}
	for _, test := range tests {
		if _, err := Parse("ub.concept", test.source); err == nil || !strings.Contains(err.Error(), test.code) {
			t.Fatalf("Parse() err=%v, want %s", err, test.code)
		}
	}
}

func TestR7d3DirectedLogicalOperatorDiagnostic(t *testing.T) {
	for _, test := range []struct{ token, replacement string }{{"&&", "and"}, {"||", "or"}} {
		source := "profile Core; bool Bad(bool A, bool B) { return A " + test.token + " B; }"
		if _, err := Parse("logical.concept", source); err == nil || !strings.Contains(err.Error(), "use '"+test.replacement+"'") {
			t.Fatalf("token %s: %v", test.token, err)
		}
	}
}

func TestR7d3SemanticSymbolIdentityAndCase(t *testing.T) {
	for _, moduleName := range []string{"A", "B"} {
		source := "module " + moduleName + "; profile Core; int Value() { return 1; }"
		module, err := Parse(filepath.ToSlash(filepath.Join(strings.ToLower(moduleName), "shared.concept")), source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		body := string(outputs["shared.generated.c"])
		want := "concept_" + strings.ToLower(moduleName) + "_value"
		if !strings.Contains(body, want) {
			t.Fatalf("module %s missing semantic symbol %s:\n%s", moduleName, want, body)
		}
	}

	source := `module Case; profile Core;
int Value() { return 1; }
int value() { return 2; }
`
	module, err := Parse("case.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["case.generated.c"])
	if !strings.Contains(body, "__n_56616c7565") || !strings.Contains(body, "__n_76616c7565") {
		t.Fatalf("case-distinct declarations did not receive unique symbols:\n%s", body)
	}
}

func TestR7d3DeterministicLiteralOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(0x7d3))
	values := []uint32{0, 1, 2147483647, 2147483648, 2654435761, 4294967295}
	for i := 0; i < 24; i++ {
		values = append(values, rng.Uint32())
	}
	for _, value := range values {
		for _, literal := range []string{fmt.Sprintf("%d", value), fmt.Sprintf("0x%X", value)} {
			source := fmt.Sprintf("profile Core; uint Value() { return %s; }", literal)
			module, err := Parse("oracle.concept", source)
			if err != nil {
				t.Fatalf("%s: %v", literal, err)
			}
			first, err := Generate(module, []byte(source))
			if err != nil {
				t.Fatalf("%s: %v", literal, err)
			}
			second, err := Generate(module, []byte(source))
			if err != nil {
				t.Fatalf("%s second generation: %v", literal, err)
			}
			if string(first["oracle.generated.c"]) != string(second["oracle.generated.c"]) {
				t.Fatalf("generated C changed for %s", literal)
			}
			want := fmt.Sprintf("UINT32_C(%d)", value)
			if !strings.Contains(string(first["oracle.generated.c"]), want) {
				t.Fatalf("%s omitted oracle value %s:\n%s", literal, want, first["oracle.generated.c"])
			}
		}
	}
}

func TestR7d3HundredRunArtifactCAndMIRDeterminism(t *testing.T) {
	source := []byte(`module BurnIn.Determinism; profile Core;
uint Mix(uint X, uint Shift) { return (X ^ 0xFF000000) + (2654435761 << Shift); }
int Boundary() { return -2147483648; }
`)
	module, err := Parse("different/location/shared.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	want, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		got, err := Generate(module, source)
		if err != nil {
			t.Fatalf("generation run %d: %v", run, err)
		}
		if len(got) != len(want) {
			t.Fatalf("generation run %d produced %d artifacts, want %d", run, len(got), len(want))
		}
		for name, expected := range want {
			if !bytes.Equal(got[name], expected) {
				t.Fatalf("artifact %s changed on generation run %d", name, run)
			}
		}
	}
}

func TestR7d3GeneratedCStrictC11(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is not installed")
	}
	source := `module BurnIn; profile Core;
uint Mix(uint X, uint Shift) { return (X + 2654435761) ^ (X << Shift); }
int Checked(int X, int Y) { return (X * 3) / Y; }
`
	module, err := Parse("burn_in.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	object := filepath.Join(dir, "burn_in.o")
	command := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-Werror", "-c", filepath.Join(dir, "burn_in.generated.c"), "-o", object)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 compile failed: %v\n%s", err, output)
	}
}

func TestR7d3LiteralRuntimeOracle(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is not installed")
	}
	source := `module Oracle; profile Core;
uint Decimal() { return 2654435761; }
uint HexMask() { return 0xFF000000; }
uint AllBits() { return 0xFFFFFFFF; }
uint Hash(uint X) { return (X ^ 0xFF000000) + 2654435761; }
`
	module, err := Parse("oracle_runtime.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	base := "concept_oracle"
	x := uint32(0x12345678)
	wantHash := (x ^ uint32(0xFF000000)) + uint32(2654435761)
	harness := fmt.Sprintf(`#include <stdint.h>
extern uint32_t %[1]s_decimal(void);
extern uint32_t %[1]s_hex_mask(void);
extern uint32_t %[1]s_all_bits(void);
extern uint32_t %[1]s_hash(uint32_t);
int main(void) {
  return %[1]s_decimal() == UINT32_C(2654435761)
      && %[1]s_hex_mask() == UINT32_C(4278190080)
      && %[1]s_all_bits() == UINT32_C(4294967295)
      && %[1]s_hash(UINT32_C(%[2]d)) == UINT32_C(%[3]d) ? 0 : 1;
}
`, base, x, wantHash)
	harnessPath := filepath.Join(dir, "harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "oracle.exe")
	command := exec.Command(compiler, "-std=c11", "-O3", "-Wall", "-Wextra", "-Werror", filepath.Join(dir, "oracle_runtime.generated.c"), harnessPath, "-o", executable)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("oracle compile failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("oracle execution failed: %v\n%s", err, output)
	}
}

func TestR7d3DeterministicLiteralExecutionOracle(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is not installed")
	}
	rng := rand.New(rand.NewSource(0x7d3c11))
	values := []uint32{0, 1, 2147483647, 2147483648, 2654435761, 4294967295}
	for i := 0; i < 18; i++ {
		values = append(values, rng.Uint32())
	}
	var source, harness strings.Builder
	source.WriteString("module LiteralFuzz; profile Core;\n")
	harness.WriteString("#include <stdint.h>\n")
	for i, value := range values {
		fmt.Fprintf(&source, "uint Decimal%d() { return %d; }\n", i, value)
		fmt.Fprintf(&source, "uint Hex%d() { return 0x%X; }\n", i, value)
		fmt.Fprintf(&harness, "extern uint32_t concept_literal_fuzz_decimal%d(void);\n", i)
		fmt.Fprintf(&harness, "extern uint32_t concept_literal_fuzz_hex%d(void);\n", i)
	}
	harness.WriteString("int main(void) {\n")
	for i, value := range values {
		fmt.Fprintf(&harness, "  if (concept_literal_fuzz_decimal%d() != UINT32_C(%d)) return %d;\n", i, value, i*2+1)
		fmt.Fprintf(&harness, "  if (concept_literal_fuzz_hex%d() != UINT32_C(%d)) return %d;\n", i, value, i*2+2)
	}
	harness.WriteString("  return 0;\n}\n")
	module, err := Parse("literal_fuzz.concept", source.String())
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source.String()))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	harnessPath := filepath.Join(dir, "literal_fuzz_harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "literal_fuzz.exe")
	command := exec.Command(compiler, "-std=c11", "-O3", "-Wall", "-Wextra", "-Werror", filepath.Join(dir, "literal_fuzz.generated.c"), harnessPath, "-o", executable)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("literal oracle compile failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("literal oracle execution failed: %v\n%s", err, output)
	}
}

func TestR7d3CrossModuleSameStemLinks(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is not installed")
	}
	dir := t.TempDir()
	var objects []string
	for index, moduleName := range []string{"A", "B"} {
		source := fmt.Sprintf("module %s; profile Core; int Value() { return %d; }", moduleName, index+1)
		module, err := Parse(filepath.ToSlash(filepath.Join(moduleName, "shared.concept")), source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		moduleDir := filepath.Join(dir, moduleName)
		if err := os.Mkdir(moduleDir, 0o700); err != nil {
			t.Fatal(err)
		}
		for name, body := range outputs {
			if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
				if err := os.WriteFile(filepath.Join(moduleDir, name), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		object := filepath.Join(moduleDir, "shared.o")
		command := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-Werror", "-c", filepath.Join(moduleDir, "shared.generated.c"), "-o", object)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("module %s compile failed: %v\n%s", moduleName, err, output)
		}
		objects = append(objects, object)
	}
	harness := `#include <stdint.h>
extern int32_t concept_a_value(void);
extern int32_t concept_b_value(void);
int main(void) { return concept_a_value() == 1 && concept_b_value() == 2 ? 0 : 1; }
`
	harnessPath := filepath.Join(dir, "link_harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "linked.exe")
	args := append([]string{"-std=c11", "-Wall", "-Wextra", "-Werror", harnessPath}, objects...)
	args = append(args, "-o", executable)
	if output, err := exec.Command(compiler, args...).CombinedOutput(); err != nil {
		t.Fatalf("link failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("linked execution failed: %v\n%s", err, output)
	}
}
