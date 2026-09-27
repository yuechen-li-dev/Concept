package concept

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7nTypedStoresArtifactConsumerAndC11(t *testing.T) {
	output := t.TempDir()
	if _, err := BuildPackage(filepath.Join("..", "..", "libraries"), output, "Standard"); err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(output, "Standard", "modules")}
	source := `module StoresConsumer;
profile Core;
import Standard.Collection.Stores;
struct Expr { int value; }
struct Stmt { int value; }
Result<Id<Expr>, StoreError> AppendExpr(ref DenseStore<Expr, 3> store, Expr value)
{
    return Append<Expr, 3>(ref store, move value);
}
int Main()
{
    Assert.Concept<NoAllocation>(AppendExpr, "artifact-only bounded append is nonallocating");
    DenseStore<Expr, 3> exprs = MakeDenseStore<Expr, 3>();
    Id<Expr> first = AppendExpr(ref exprs, Expr{7})!;
    Id<Expr> second = Append<Expr, 3>(ref exprs, Expr{9})!;
    GenerationalStore<Stmt, 2> stmts = MakeGenerationalStore<Stmt, 2>();
    GenerationalId<Stmt> old = Insert<Stmt, 2>(ref stmts, Stmt{11})!;
    Remove<Stmt, 2>(ref stmts, old)!;
    GenerationalId<Stmt> reused = Insert<Stmt, 2>(ref stmts, Stmt{13})!;
    if (Contains<Stmt, 2>(ref const stmts, old) or
        reused.index != old.index or reused.generation != old.generation + 1)
    {
        return 1;
    }
    return DenseGet<Expr, 3>(ref const exprs, first)!.value +
        DenseGet<Expr, 3>(ref const exprs, second)!.value +
        GenerationalGet<Stmt, 2>(ref const stmts, reused)!.value;
}`
	module, err := ParseWithSemanticModuleRoots("StoresConsumer.concept", source, roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		body := string(moduleOutput(t, outputs, ".generated.c"))
		if !strings.Contains(string(moduleOutput(t, outputs, ".mir.json")), "NoAllocation") {
			t.Fatal("store proof is absent from deterministic MIR")
		}
		for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "runtime_type_registry"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("store consumer emits %q", forbidden)
			}
		}
		runFoundationNativeHarness(t, outputs, "stores_consumer_harness.c", "#include \"storesconsumer.generated.h\"\nint main(void) { return concept_stores_consumer_main() == 29 ? 0 : 1; }\n")
		compiler, lookupErr := exec.LookPath("clang")
		if lookupErr != nil {
			compiler, lookupErr = exec.LookPath("gcc")
		}
		if lookupErr == nil {
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			command := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-Wno-unused-function", "-I", dir, "-fsyntax-only", filepath.Join(dir, "storesconsumer.generated.c"))
			if diagnostics, err := command.CombinedOutput(); err != nil {
				t.Fatalf("strict C11 store consumer: %v\n%s", err, diagnostics)
			}
		}
		for run := 2; run <= 100; run++ {
			repeated, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
			if err != nil || !equalOutputs(outputs, repeated) {
				t.Fatalf("store output changed on run %d: %v", run, err)
			}
		}
	}

	invalid := `module StoresInvalid; profile Core;
import Standard.Collection.Stores;
struct Expr { int value; }
struct Stmt { int value; }
void Mismatch(Id<Expr> expr) { Id<Stmt> wrong = expr; }`
	if _, err := ParseWithSemanticModuleRoots("StoresInvalid.concept", invalid, roots); err == nil || !strings.Contains(err.Error(), "CV") {
		t.Fatalf("mixed Id types must be rejected, got %v", err)
	}
	immovable := `module StoresPinned; profile Core;
import Standard.Collection.Stores;
immovable struct Pinned { int value; }
void Probe()
{
    DenseStore<Pinned, 2> store = MakeDenseStore<Pinned, 2>();
    Append<Pinned, 2>(ref store, Pinned{1})!;
}`
	if _, err := ParseWithSemanticModuleRoots("StoresPinned.concept", immovable, roots); err == nil {
		t.Fatal("immovable insertion must reject rather than relocate a pinned value")
	}
}
