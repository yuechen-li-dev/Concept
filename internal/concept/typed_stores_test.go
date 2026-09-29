package concept

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypedStoresArtifactConsumerAndC11(t *testing.T) {
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

requires compiler.InvalidatesBorrows(AppendExpr, store);
int Main()
{
    Assert.Concept<NoAllocation>(AppendExpr, "artifact-only bounded append is nonallocating");
    DenseStore<Expr, 3> exprs = MakeDenseStore<Expr, 3>();
    Id<Expr> first = AppendExpr(ref exprs, Expr{7})!;
    Id<Expr> second = Append<Expr, 3>(ref exprs, Expr{9})!;
    GenerationalStore<Stmt, 2> stmts = MakeGenerationalStore<Stmt, 2>();
    if (GenerationalCount<Stmt, 2>(ref const stmts) != 0) { return 2; }
    GenerationalId<Stmt> old = Insert<Stmt, 2>(ref stmts, Stmt{11})!;
    if (GenerationalCount<Stmt, 2>(ref const stmts) != 1) { return 3; }
    Remove<Stmt, 2>(ref stmts, old)!;
    if (GenerationalCount<Stmt, 2>(ref const stmts) != 0) { return 4; }
    GenerationalId<Stmt> reused = Insert<Stmt, 2>(ref stmts, Stmt{13})!;
    if (GenerationalCount<Stmt, 2>(ref const stmts) != 1) { return 5; }
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

func TestDenseImmovableArtifactEmplace(t *testing.T) {
	output := t.TempDir()
	if _, err := BuildPackage(filepath.Join("..", "..", "libraries"), output, "Standard"); err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(output, "Standard", "modules")}
	source := `module PinnedStoreConsumer;
profile Core;
import Standard.Collection.Stores;
extern "C" void ObserveDrop(int value);
extern "C" void Checkpoint(int stage);
enum BuildError { Failed }
struct Part { int value; }
void Drop(owned Part part) { ObserveDrop(part.value); }
immovable struct Pinned { owned Part part; int key; }
Result<int, BuildError> Key(bool fail)
{
    if (fail) { return Result::Error(BuildError::Failed); }
    return Result::Ok(9);
}
Id<Pinned> EmplacePinned(ref DenseStore<Pinned, 4> store, int key)
{
    return Emplace(ref store, Pinned{Part{7}, key});
}
requires compiler.InvalidatesBorrows(EmplacePinned, store);
Result<int, BuildError> Build(bool fail)
{
    DenseStore<Pinned, 4> store = DenseStore<Pinned, 4>{Uninitialized()};
    Id<Pinned> id = Emplace(ref store, Pinned{Part{7}, Key(fail)?});
    ReadOnlySpan<Pinned> values = ReadOnlyValues<Pinned, 4>(ref const store);
    if (id.index != 0 or Len(values) != 1) { return Result::Ok(1); }
    return Result::Ok(values[0].key);
}
Result<int, BuildError> BuildGenerational(bool fail)
{
    GenerationalStore<Pinned, 2> store = GenerationalStore<Pinned, 2>{Uninitialized(), [1 ...], [0 ...], 0, 0, 0};
    GenerationalId<Pinned> id = Emplace(ref store, Pinned{Part{7}, Key(fail)?});
    ref const Pinned value = GenerationalGet<Pinned, 2>(ref const store, id)!;
    return Result::Ok(value.key);
}
Result<GenerationalId<Pinned>, BuildError> TryPinned(
    ref GenerationalStore<Pinned, 2> store, bool fail)
{
    GenerationalId<Pinned> id = Emplace(ref store, Pinned{Part{7}, Key(fail)?});
    return Result::Ok(id);
}
Result<int, BuildError> BuildReusedSlot(bool fail)
{
    GenerationalStore<Pinned, 2> store = GenerationalStore<Pinned, 2>{Uninitialized(), [1 ...], [0 ...], 0, 0, 0};
    GenerationalId<Pinned> first = Emplace(ref store, Pinned{Part{7}, 1});
    Remove<Pinned, 2>(ref store, first)!;
    match (TryPinned(ref store, fail))
    {
        Result::Ok(reused) => {
            if (fail or reused.index != first.index or reused.generation != first.generation + 1 or
                Contains<Pinned, 2>(ref const store, first)) { return Result::Ok(1); }
        }
        Result::Error(error) => {
            if (error != BuildError::Failed or not fail or store.freeCount != 1 or store.next != 1 or
                store.generation[first.index] != first.generation + 1) { return Result::Ok(1); }
            GenerationalId<Pinned> after = Emplace(ref store, Pinned{Part{7}, 9});
            if (after.index != first.index or after.generation != first.generation + 1) { return Result::Ok(1); }
        }
    }
    return Result::Ok(0);
}
int Main()
{
    Assert.Concept<NoAllocation>(EmplacePinned, "stationary bounded insertion allocates nothing");
    discard Build(true);
    Checkpoint(1);
    discard Build(false);
    Checkpoint(2);
    discard BuildGenerational(true);
    Checkpoint(3);
    discard BuildGenerational(false);
    Checkpoint(4);
    discard BuildReusedSlot(true);
    Checkpoint(7);
    discard BuildReusedSlot(false);
    Checkpoint(9);
    return 0;
}`
	module, err := ParseWithSemanticModuleRoots("PinnedStoreConsumer.concept", source, roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		body := string(moduleOutput(t, outputs, ".generated.c"))
		for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "memcpy("} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("stationary store emitted %q", forbidden)
			}
		}
		header := string(moduleOutput(t, outputs, ".generated.h"))
		if !strings.Contains(header, "int count;") || !strings.Contains(header, "bool live[") {
			t.Fatal("generated C lacks dense prefix count or sparse live bits")
		}
		if !strings.Contains(string(moduleOutput(t, outputs, ".mir.json")), "NoAllocation") {
			t.Fatal("stationary NoAllocation proof is absent from MIR")
		}
		harness := `#include "pinnedstoreconsumer.generated.h"
static int drops = 0;
static int bad = 0;
void ObserveDrop(int value) { drops += 1; if (value != 7) bad = 1; }
void Checkpoint(int stage) { if (drops != stage) bad = 1; }
int main(void) { return concept_pinned_store_consumer_main() == 0 && drops == 9 && !bad ? 0 : 1; }
`
		runFoundationNativeHarness(t, outputs, "pinned_store_harness.c", harness)
		compiler, lookupErr := exec.LookPath("clang")
		if lookupErr != nil {
			compiler, lookupErr = exec.LookPath("gcc")
		}
		if lookupErr == nil {
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			command := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-Wno-unused-function", "-I", dir, "-fsyntax-only", filepath.Join(dir, "pinnedstoreconsumer.generated.c"))
			if diagnostics, err := command.CombinedOutput(); err != nil {
				t.Fatalf("strict C11 stationary stores: %v\n%s", err, diagnostics)
			}
		}
		for run := 2; run <= 100; run++ {
			repeated, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
			if err != nil || !equalOutputs(outputs, repeated) {
				t.Fatalf("stationary store output changed on run %d: %v", run, err)
			}
		}
	}
	invalid := `module PinnedStoreMove; profile Core; import Standard.Collection.Stores;
immovable struct Pinned { int value; }
void Probe()
{
    DenseStore<Pinned, 2> store = DenseStore<Pinned, 2>{Uninitialized()};
    Emplace(ref store, Pinned{7});
    DenseStore<Pinned, 2> moved = move store;
}`
	if _, err := ParseWithSemanticModuleRoots("PinnedStoreMove.concept", invalid, roots); err == nil || !strings.Contains(err.Error(), "CV4505") {
		t.Fatalf("immovable dense store move = %v, want CV4505", err)
	}
	invalidGenerationalMove := `module PinnedGenerationalMove; profile Core; import Standard.Collection.Stores;
immovable struct Pinned { int value; }
void Probe()
{
    GenerationalStore<Pinned, 2> store = GenerationalStore<Pinned, 2>{Uninitialized(), [1 ...], [0 ...], 0, 0, 0};
    Emplace(ref store, Pinned{7});
    GenerationalStore<Pinned, 2> moved = move store;
}`
	if _, err := ParseWithSemanticModuleRoots("PinnedGenerationalMove.concept", invalidGenerationalMove, roots); err == nil || !strings.Contains(err.Error(), "CV4505") {
		t.Fatalf("immovable sparse store move = %v, want CV4505", err)
	}
	invalidInsert := `module PinnedStoreInsert; profile Core; import Standard.Collection.Stores;
immovable struct Pinned { int value; }
void Probe()
{
    GenerationalStore<Pinned, 2> store = GenerationalStore<Pinned, 2>{Uninitialized(), [1 ...], [0 ...], 0, 0, 0};
    Pinned value = Pinned{7};
    Insert<Pinned, 2>(ref store, move value)!;
}`
	if _, err := ParseWithSemanticModuleRoots("PinnedStoreInsert.concept", invalidInsert, roots); err == nil || !strings.Contains(err.Error(), "CV4136") {
		t.Fatalf("by-value immovable Insert = %v, want CV4136", err)
	}
	borrowConflict := `module PinnedStoreBorrow; profile Core; import Standard.Collection.Stores;
struct Item { int value; }
void Probe()
{
    DenseStore<Item, 2> store = MakeDenseStore<Item, 2>();
    ReadOnlySpan<Item> values = ReadOnlyValues<Item, 2>(ref const store);
    Emplace(ref store, Item{7});
}`
	if _, err := ParseWithSemanticModuleRoots("PinnedStoreBorrow.concept", borrowConflict, roots); err == nil || !strings.Contains(err.Error(), "RAW_STORAGE_BORROW_CONFLICT") {
		t.Fatalf("append with live Span = %v, want RAW_STORAGE_BORROW_CONFLICT", err)
	}
	escapingView := `module EscapingStoreView; profile Core; import Standard.Collection.Stores;
struct Item { int value; }
ReadOnlySpan<Item> Escape()
{
    DenseStore<Item, 2> store = MakeDenseStore<Item, 2>();
    Emplace(ref store, Item{7});
    return ReadOnlyValues<Item, 2>(ref const store);
}`
	if _, err := ParseWithSemanticModuleRoots("EscapingStoreView.concept", escapingView, roots); err == nil || !strings.Contains(err.Error(), "CV4521") {
		t.Fatalf("escaping dense view = %v, want CV4521", err)
	}
	constMutableView := `module ConstStoreView; profile Core; import Standard.Collection.Stores;
struct Item { int value; }
void Probe(ref const DenseStore<Item, 2> store)
{
    Span<Item> mutableView = Values<Item, 2>(ref const store);
}`
	if _, err := ParseWithSemanticModuleRoots("ConstStoreView.concept", constMutableView, roots); err == nil {
		t.Fatal("mutable Span from const store must reject")
	}
	sparseBorrowConflict := `profile Core;
struct Item { int value; }
void Probe()
{
    Item<sparse>[2] slots = Uninitialized();
    SparseInitialize(ref slots, 0, Item{7});
    ref Item held = SparseGet(ref slots, 0);
    SparseRemove(ref slots, 0);
    held.value = 9;
}`
	if _, err := Parse("SparseBorrowConflict.concept", sparseBorrowConflict); err == nil || !strings.Contains(err.Error(), "RAW_STORAGE_BORROW_CONFLICT") {
		t.Fatalf("remove with live sparse ref = %v, want borrow conflict", err)
	}
	allocatingInitializer := `module StoreAllocatingInitializer; profile Core; import Standard.Collection.Stores;
extern "C" int AllocatingField();
requires compiler.Allocates(AllocatingField);
struct Item { int value; }
Id<Item> Add(ref DenseStore<Item, 2> store)
{
    return Emplace(ref store, Item{AllocatingField()});
}
void Probe()
{
    Assert.Concept<NoAllocation>(Add, "field allocation must propagate through Emplace");
}`
	if _, err := ParseWithSemanticModuleRoots("StoreAllocatingInitializer.concept", allocatingInitializer, roots); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
		t.Fatalf("allocating stationary initializer proof = %v, want disproven", err)
	}
}
