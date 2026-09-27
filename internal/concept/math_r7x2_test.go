package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7x2StandardMathGeneration(t *testing.T) {
	path := filepath.Join("..", "..", "libraries", "Standard", "Math.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	base := evt1SemanticSymbolBase(module)
	call := func(name string) string { return evt1FunctionSymbol(base, name) }
	harness := "#include <math.h>\n#include \"math.generated.h\"\nint main(void) {\n" +
		"if (" + call("Abs") + "__int(-7) != 7 || " + call("Min") + "__int_int(2, 3) != 2 || " + call("Max") + "__int_int(2, 3) != 3 || " + call("Clamp") + "__int_int_int(8, 1, 5) != 5) return 11;\n" +
		"if (" + call("Abs") + "__float(-2.0f) != 2.0f || " + call("Min") + "__float_float(2.0f, 3.0f) != 2.0f || " + call("Max") + "__float_float(2.0f, 3.0f) != 3.0f || " + call("Clamp") + "__float_float_float(8.0f, 1.0f, 5.0f) != 5.0f) return 12;\n" +
		"if (fabsf(" + call("Sqrt") + "(9.0f) - 3.0f) > 0.0001f) return 1;\n" +
		"if (fabsf(" + call("Pow") + "(2.0f, 3.0f) - 8.0f) > 0.0001f) return 2;\n" +
		"if (fabsf(" + call("Exp") + "(0.0f) - 1.0f) > 0.0001f) return 3;\n" +
		"if (fabsf(" + call("Log") + "(1.0f)) > 0.0001f) return 4;\n" +
		"if (fabsf(" + call("Log2") + "(8.0f) - 3.0f) > 0.0001f) return 5;\n" +
		"if (fabsf(" + call("Log10") + "(100.0f) - 2.0f) > 0.0001f) return 6;\n" +
		"if (" + call("Floor") + "(2.8f) != 2.0f || " + call("Ceil") + "(2.2f) != 3.0f || " + call("Round") + "(2.5f) != 3.0f || " + call("Trunc") + "(-2.8f) != -2.0f) return 7;\n" +
		"if (fabsf(" + call("Sin") + "(0.0f)) > 0.0001f || fabsf(" + call("Cos") + "(0.0f) - 1.0f) > 0.0001f || fabsf(" + call("Tan") + "(0.0f)) > 0.0001f) return 8;\n" +
		"if (fabsf(" + call("Asin") + "(0.0f)) > 0.0001f || fabsf(" + call("Acos") + "(1.0f)) > 0.0001f || fabsf(" + call("Atan") + "(0.0f)) > 0.0001f || fabsf(" + call("Atan2") + "(0.0f, 1.0f)) > 0.0001f) return 9;\n" +
		"if (!" + call("IsFinite") + "(1.0f) || !" + call("IsInf") + "(INFINITY) || !" + call("IsNaN") + "(NAN)) return 10;\n" +
		"return 0; }\n"
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err == nil {
		dir := t.TempDir()
		if err := Write(dir, outputs); err != nil {
			t.Fatal(err)
		}
		harnessPath := filepath.Join(dir, "math_harness.c")
		if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
			t.Fatal(err)
		}
		executable := filepath.Join(dir, "math_harness.exe")
		cmd := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-Werror", "-I", dir, filepath.Join(dir, "math.generated.c"), harnessPath, "-lm", "-o", executable)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict C11 math compile: %v\n%s", err, output)
		}
		if output, err := exec.Command(executable).CombinedOutput(); err != nil {
			t.Fatalf("math execution: %v\n%s", err, output)
		}
	}
	var assertions strings.Builder
	assertions.WriteString(string(source))
	assertions.WriteString("\nint ProveMath() {\n")
	for _, name := range []string{"Abs", "Min", "Max", "Clamp", "Sqrt", "Pow", "Exp", "Log", "Log2", "Log10", "Floor", "Ceil", "Round", "Trunc", "Sin", "Cos", "Tan", "Asin", "Acos", "Atan", "Atan2", "IsFinite", "IsInf", "IsNaN"} {
		if name == "Abs" || name == "Min" || name == "Max" || name == "Clamp" {
			continue // overloaded subjects require a typed disambiguation in this assertion syntax
		}
		assertions.WriteString("Assert.Concept<NoAllocation>(" + name + ", \"math primitive does not allocate\");\n")
	}
	assertions.WriteString("return 0; }\n")
	if _, err := Parse(path, assertions.String()); err != nil {
		t.Fatalf("NoAllocation math proof: %v", err)
	}
}
