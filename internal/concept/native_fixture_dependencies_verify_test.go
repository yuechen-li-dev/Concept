package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeFixtureHeaderClosureKeys(t *testing.T) {
	outputs := Outputs{"module.generated.c": []byte("#include \"used.h\"\nint value(void){return VALUE;}\n"), "used.h": []byte("#include \"nested.h\"\n"), "nested.h": []byte("#define VALUE 7\n"), "oracle.generated.h": []byte("old unrelated")}
	key, ok := nativeFixtureKey(outputs, "module.generated.c", "unavailable-compiler", []string{"-O2"})
	if !ok {
		t.Fatal("literal closure not cacheable")
	}
	outputs["oracle.generated.h"] = []byte("new unrelated")
	next, ok := nativeFixtureKey(outputs, "module.generated.c", "unavailable-compiler", []string{"-O2"})
	if !ok || key != next {
		t.Fatal("unrelated header invalidated backend")
	}
	outputs["nested.h"] = []byte("#define VALUE 9\n")
	next, ok = nativeFixtureKey(outputs, "module.generated.c", "unavailable-compiler", []string{"-O2"})
	if !ok || key == next {
		t.Fatal("transitive header change ignored")
	}
	flags, _ := nativeFixtureKey(outputs, "module.generated.c", "unavailable-compiler", []string{"-O0"})
	if flags == next {
		t.Fatal("flags ignored")
	}
	for _, directive := range []string{"#include HEADER", "# include \"missing.h\"", "#/**/include \"used.h\"", "#import \"used.h\"", "#inc\\\nlude \"used.h\""} {
		outputs["module.generated.c"] = []byte(directive)
		if _, ok := nativeFixtureKey(outputs, "module.generated.c", "unavailable-compiler", nil); ok {
			t.Fatalf("unproven dependency cacheable: %s", directive)
		}
	}
}
func TestNativeFixtureHeaderClosureCompiledReuse(t *testing.T) {
	requireNativeQualification(t)
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	outputs := Outputs{"closure.generated.c": []byte("#include \"used.h\"\nint closure_value(void){return VALUE;}\n"), "used.h": []byte("#define VALUE 7\n"), "oracle.generated.h": []byte("unused first")}
	first := nativeFixtureObjects(t, outputs, compiler, "-O2")
	outputs["oracle.generated.h"] = []byte("unused second")
	second := nativeFixtureObjects(t, outputs, compiler, "-O2")
	if first[0] != second[0] {
		t.Fatal("recompiled unchanged backend inputs")
	}
	outputs["used.h"] = []byte("#define VALUE 9\n")
	third := nativeFixtureObjects(t, outputs, compiler, "-O2")
	if first[0] == third[0] {
		t.Fatal("reused stale object after used-header change")
	}
	dir := t.TempDir()
	host := filepath.Join(dir, "host.c")
	program := filepath.Join(dir, "closure.exe")
	if err := os.WriteFile(host, []byte("int closure_value(void);int main(void){return closure_value()==9?0:1;}"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, compiler, third[0], host, "-o", program).CombinedOutput(); err != nil {
		t.Fatalf("link: %v %s", err, out)
	}
	if out, err := nativeCommand(t, program).CombinedOutput(); err != nil {
		t.Fatalf("stale compiled value: %v %s", err, out)
	}
}
