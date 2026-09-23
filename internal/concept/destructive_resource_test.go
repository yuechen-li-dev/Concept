package concept

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestR7f2DestructiveReferenceContract(t *testing.T) {
	provider := `module Library.Resource; profile Core;
struct Store { int value; };
ref const int Borrow(ref const Store store) { return ref const store.value; }
void Reset(ref Store store) { store.value = 0; }
requires compiler.InvalidatesBorrows(Reset, store);
`
	artifact := buildSemanticArtifact(t, "Library/Resource.concept", provider, nil)
	deps := map[string][]byte{"Library.Resource": artifact}
	valid := `module App; profile Core; import Library.Resource;
int Main() { Store first = Store{7}; Store second = Store{9}; int result = 0; { ref const int item = Borrow(ref const first); Reset(ref second); result = item; } Reset(ref first); return result + first.value; }
`
	module, err := ParseWithSemanticModules("App.concept", valid, deps)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r7f2_resource_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 7 ? 0 : 1; }\n")
	invalid := strings.Replace(valid, "Reset(ref second);", "Reset(ref first);", 1)
	_, err = ParseWithSemanticModules("Invalid.concept", invalid, deps)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("overlap = %v", err)
	}
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Proof == nil {
		t.Fatalf("destructive conflict omitted proof: %v", err)
	}
	proof, err := SerializeProof(*diagnostic.Proof)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"InvalidatesBorrows", "item", "store", "ModuleSummaryEffect"} {
		if !bytes.Contains(proof, []byte(expected)) {
			t.Fatalf("proof omitted %s: %s", expected, proof)
		}
	}
	firstMIR := moduleOutput(t, outputs, ".mir.json")
	firstC := moduleOutput(t, outputs, ".generated.c")
	for i := 0; i < 100; i++ {
		again := buildSemanticArtifact(t, "Library/Resource.concept", provider, nil)
		if !bytes.Equal(artifact, again) {
			t.Fatalf("artifact changed on run %d", i)
		}
		recompiled, err := ParseWithSemanticModules("App.concept", valid, map[string][]byte{"Library.Resource": again})
		if err != nil {
			t.Fatal(err)
		}
		generated, err := Generate(recompiled, []byte(valid))
		if err != nil {
			t.Fatal(err)
		}
		if moduleOutput(t, generated, ".mir.json") != firstMIR || moduleOutput(t, generated, ".generated.c") != firstC {
			t.Fatalf("MIR or C changed on run %d", i)
		}
		_, conflict := ParseWithSemanticModules("Invalid.concept", invalid, map[string][]byte{"Library.Resource": again})
		var repeated Diagnostic
		if !errors.As(conflict, &repeated) || repeated.Proof == nil {
			t.Fatalf("proof missing on run %d: %v", i, conflict)
		}
		againProof, err := SerializeProof(*repeated.Proof)
		if err != nil || !bytes.Equal(proof, againProof) {
			t.Fatalf("proof changed on run %d: %v", i, err)
		}
	}
}

func TestR7f2OrdinaryMutationIsNotLifetimeInvalidation(t *testing.T) {
	source := `profile Core; struct Store { int value; };
ref const int Borrow(ref const Store store) { return ref const store.value; }
void Update(ref Store store) { store.value = 8; }
int Main() { Store store = Store{7}; ref const int held = Borrow(ref const store); Update(ref store); return held; }
`
	module, err := Parse("ordinary_mutation.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "ordinary_mutation_harness.c", "#include \"ordinary_mutation.generated.h\"\nint main(void) { return concept_ordinary_mutation_main() == 8 ? 0 : 1; }\n")
}

func TestR7f2InvalidDestructiveContractTarget(t *testing.T) {
	for _, source := range []string{
		`profile Core; void Reset(int value) { } requires compiler.InvalidatesBorrows(Reset, value);`,
		`profile Core; struct Store { int value; }; void Reset(ref Store store) { } requires compiler.InvalidatesBorrows(Reset, missing);`,
	} {
		_, err := Parse("bad_resource_effect.concept", source)
		if err == nil || !strings.Contains(err.Error(), "OPERATION_EFFECT_RESOURCE_INVALID") {
			t.Fatalf("invalid contract target = %v", err)
		}
	}
}

func TestR7f2GenericDestructiveReferenceContract(t *testing.T) {
	source := `profile Core; template <typename T> struct Store { T value; };
template <typename T> ref const T Borrow(ref const Store<T> store) { return ref const store.value; }
template <typename T> void Reset(ref Store<T> store) { }
requires compiler.InvalidatesBorrows(Reset, store);
int Main() { Store<int> store = Store<int>{7}; ref const int item = Borrow<int>(ref const store); Reset<int>(ref store); return item; }
`
	_, err := Parse("generic_destructive.concept", source)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("generic overlap = %v", err)
	}
}

func TestR7f2GenericForwardedEffect(t *testing.T) {
	base := `profile Core; template <typename T> struct Store { T value; };
template <typename T> ref const T Borrow(ref const Store<T> store) { return ref const store.value; }
template <typename T> void Reset(ref Store<T> store) { }
requires compiler.InvalidatesBorrows(Reset, store);
template <typename T> void Forward(ref Store<T> store) { Reset<T>(ref store); }
int Main() { Store<int> store = Store<int>{7}; ref const int held = Borrow<int>(ref const store); Forward<int>(ref store); return held; }
`
	_, err := Parse("generic_forward_missing.concept", base)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_EFFECT_NOT_PROPAGATED") {
		t.Fatalf("generic forwarding = %v", err)
	}
	declared := strings.Replace(base, "int Main()", "requires compiler.InvalidatesBorrows(Forward, store);\nint Main()", 1)
	_, err = Parse("generic_forward_overlap.concept", declared)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("generic overlap = %v", err)
	}
}

func TestR7f2ForwardedDestructiveEffectMustBeDeclared(t *testing.T) {
	base := `profile Core; struct Store { int value; };
ref const int Borrow(ref const Store store) { return ref const store.value; }
void Reset(ref Store store) { store.value = 0; }
requires compiler.InvalidatesBorrows(Reset, store);
void Forward(ref Store store) { Reset(ref store); }
int Main() { Store store = Store{7}; ref const int held = Borrow(ref const store); Forward(ref store); return held; }
`
	_, err := Parse("forward_missing.concept", base)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_EFFECT_NOT_PROPAGATED") {
		t.Fatalf("undeclared forwarding = %v", err)
	}
	declared := strings.Replace(base, "int Main()", "requires compiler.InvalidatesBorrows(Forward, store);\nint Main()", 1)
	_, err = Parse("forward_overlap.concept", declared)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("forwarded overlap = %v", err)
	}
}

func TestR7f2ScopedLeaseCannotSilentlyCrossSuspension(t *testing.T) {
	base := `profile Core; struct Store { int value; };
ref struct Lease { ref const Store owner; };
Lease Borrow(ref const Store store) { return Lease{ref const store}; }
void Reset(ref Store store) { store.value = 0; }
requires compiler.InvalidatesBorrows(Reset, store);
`
	asyncSource := base + `async int Child() { return 1; }
async int Work() { Store store = Store{7}; Lease lease = Borrow(ref const store); int value = await Child(); return lease.owner.value + value; }
`
	_, err := Parse("lease_await.concept", asyncSource)
	if err == nil || !strings.Contains(err.Error(), "SCOPED_AUTHORITY_CROSSES_AWAIT") {
		t.Fatalf("lease across await = %v", err)
	}
	yieldSource := base + `automata Worker { machine Run { state Waiting { Store store = Store{7}; Lease lease = Borrow(ref const store); yield; } } }
int Main() { instance Worker worker(); Step(worker, Run); return 0; }
`
	_, err = Parse("lease_yield.concept", yieldSource)
	if err == nil || !strings.Contains(err.Error(), "SCOPED_AUTHORITY_CROSSES_YIELD") {
		t.Fatalf("lease across yield = %v", err)
	}
	persistent := base + `automata Worker with state { Lease lease; } { machine Run { state Waiting { yield; } } }
`
	_, err = Parse("lease_field.concept", persistent)
	if err == nil || !strings.Contains(err.Error(), "SCOPED_AUTHORITY_PERSISTENT_FIELD") {
		t.Fatalf("persistent lease field = %v", err)
	}
}

func TestR7f2AwaitedDestructiveOperationConflictsBeforeSuspension(t *testing.T) {
	source := `profile Core; struct Store { int value; };
ref struct Lease { ref const Store owner; };
Lease Borrow(ref const Store store) { return Lease{ref const store}; }
void Reset(ref Store store) { store.value = 0; }
requires compiler.InvalidatesBorrows(Reset, store);
async int ResetLater(ref Store store) { Reset(ref store); return 1; }
requires compiler.InvalidatesBorrows(ResetLater, store);
async int Work() { Store store = Store{7}; Lease lease = Borrow(ref const store); int result = await ResetLater(ref store); return lease.owner.value + result; }
`
	_, err := Parse("awaited_reclaim.concept", source)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("awaited destructive call = %v", err)
	}
}
