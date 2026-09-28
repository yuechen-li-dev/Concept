package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestR8eMustUseAndDiscard(t *testing.T) {
	const header = `module R8e.Core; profile Core;
[[must_use]] enum Ticket { Valid, }
Ticket Issue() { return Ticket::Valid; }
[[must_use]] int Acquire() { return 3; }
Result<int, int> Attempt() { return Result::Ok(5); }
`
	for _, call := range []string{"Issue()", "Acquire()", "Attempt()"} {
		_, err := Parse("ignored.concept", header+"int Run() { "+call+"; return 0; }")
		if err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") || !strings.Contains(err.Error(), "discard") {
			t.Fatalf("ignored %s: %v", call, err)
		}
	}
	source := header + `int Run() { discard Issue(); discard Acquire(); discard Attempt(); return 0; }`
	module, err := Parse("discard.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(outputs["discard.mir.json"], []byte(`"kind": "discard_value"`)); got != 3 {
		t.Fatalf("MIR has %d explicit discards, want 3", got)
	}
	for _, call := range []string{"concept_r8e__core_issue()", "concept_r8e__core_acquire()", "concept_r8e__core_attempt()"} {
		if got := bytes.Count(outputs["discard.generated.c"], []byte(call)); got != 1 {
			t.Fatalf("generated C has %d invocations of %s, want one", got, call)
		}
	}
	assertR8cStrictC11(t, outputs, "discard.generated.c")
}

func TestR8eMustUseArtifactOnly(t *testing.T) {
	const provider = `module R8e.Provider; profile Core;
[[must_use]] enum Ticket { Valid, }
Ticket Issue() { return Ticket::Valid; }
[[must_use]] int Acquire() { return 4; }
`
	artifact, err := CompileSemanticModule("provider.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = `module R8e.Consumer; profile Core; import R8e.Provider; `
	for _, call := range []string{"Issue()", "Acquire()"} {
		_, err := ParseWithSemanticModules("consumer.concept", prefix+"int Run() { "+call+"; return 0; }", map[string][]byte{"R8e.Provider": artifact})
		if err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
			t.Fatalf("artifact-only ignored %s: %v", call, err)
		}
		_, err = ParseWithSemanticModules("consumer.concept", prefix+"int Run() { discard "+call+"; return 0; }", map[string][]byte{"R8e.Provider": artifact})
		if err != nil {
			t.Fatalf("artifact-only discard %s: %v", call, err)
		}
	}
}

func TestR8eGenericMustUseAndExactOnce(t *testing.T) {
	const generic = `module R8e.Generic; profile Core;
[[must_use]] enum Ticket { Valid, }
template <typename T> T Identity(T value) { return value; }
int Run() { Identity<Ticket>(Ticket::Valid); return 0; }
`
	if _, err := Parse("generic.concept", generic); err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
		t.Fatalf("closed generic result was not MustUse: %v", err)
	}
	const alias = `module R8e.Alias; profile Core;
[[must_use]] enum Ticket { Valid, }
using TicketAlias = Ticket;
TicketAlias Issue() { return Ticket::Valid; }
int Run() { Issue(); return 0; }
`
	if _, err := Parse("alias.concept", alias); err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
		t.Fatalf("alias result was not MustUse: %v", err)
	}
	const source = `module R8e.Once; profile Core;
extern "C" int Tick();
[[must_use]] int Acquire() { return Tick(); }
int Run() { discard Acquire(); return 0; }
`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("once.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "once.generated.c")
		runFoundationNativeHarness(t, outputs, "r8e_once_harness.c", `#include "once.generated.h"
static int ticks = 0;
int Tick(void) { return ++ticks; }
int main(void) {
  if (concept_r8e__once_run() != 0) return 1;
  return ticks == 1 ? 0 : 2;
}`)
	}
}

func TestR8eForeignMustUseArtifact(t *testing.T) {
	const provider = `module R8e.Native; profile Core;
[[must_use]] extern "C" int native_status();
`
	artifact, err := CompileSemanticModule("native.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = `module R8e.Client; profile Core; import R8e.Native; `
	_, err = ParseWithSemanticModules("client.concept", prefix+`int Run() { native_status(); return 0; }`, map[string][]byte{"R8e.Native": artifact})
	if err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
		t.Fatalf("foreign artifact return was not MustUse: %v", err)
	}
	_, err = ParseWithSemanticModules("client.concept", prefix+`int Run() { discard native_status(); return 0; }`, map[string][]byte{"R8e.Native": artifact})
	if err != nil {
		t.Fatalf("explicit discard of foreign return failed: %v", err)
	}
}

func TestR8eDiscardDropsOwnedResult(t *testing.T) {
	const source = `module R8e.Owned; profile Core;
extern "C" void ObserveDrop(int value);
[[must_use]] struct Resource { int handle; }
void Drop(owned Resource resource) { ObserveDrop(resource.handle); }
owned Resource MakeResource() { return Resource{42}; }
int Run() { discard MakeResource(); return 0; }
`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("owned.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "owned.generated.c")
		runFoundationNativeHarness(t, outputs, "r8e_owned_harness.c", `#include "owned.generated.h"
static int drops = 0;
void ObserveDrop(int value) { if (value == 42) drops++; }
int main(void) {
  if (concept_r8e__owned_run() != 0) return 1;
  return drops == 1 ? 0 : 2;
}`)
	}
}

func TestR8eMustUseDeterminism100(t *testing.T) {
	const source = `module R8e.Stable; profile Core;
[[must_use]] enum Ticket { Valid, }
Ticket Issue() { return Ticket::Valid; }
int Run() { discard Issue(); return 0; }
`
	const invalid = `module R8e.Stable; profile Core;
[[must_use]] enum Ticket { Valid, }
Ticket Issue() { return Ticket::Valid; }
int Run() { Issue(); return 0; }
`
	var firstArtifact, firstC, firstMIR []byte
	var firstDiagnostic string
	for i := 0; i < 100; i++ {
		artifact, err := CompileSemanticModule("stable.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse("stable.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse("stable.concept", invalid)
		if err == nil {
			t.Fatal("ignored MustUse value unexpectedly accepted")
		}
		if i == 0 {
			firstArtifact, firstC, firstMIR, firstDiagnostic = artifact, outputs["stable.generated.c"], outputs["stable.mir.json"], err.Error()
			continue
		}
		if !bytes.Equal(artifact, firstArtifact) || !bytes.Equal(outputs["stable.generated.c"], firstC) || !bytes.Equal(outputs["stable.mir.json"], firstMIR) || err.Error() != firstDiagnostic {
			t.Fatalf("MustUse artifact, C, MIR, or diagnostic drifted on run %d", i)
		}
	}
}

func TestR8eFirstContactDiagnostics(t *testing.T) {
	cases := []struct{ source, code, hint string }{
		{`int Run(int count) { if (count) { return 1; } return 0; }`, "CV4186", "require bool"},
		{`int Run(int count) { for (int i = 0; i < count; ++i) { } return 0; }`, "FOREACH_ITERATOR_INVALID", "descend"},
		{`int Run(float value) { return (int)value; }`, "C_STYLE_CAST_UNSUPPORTED", "TruncTo"},
		{`int Run(int value) { return reinterpret_cast<int>(value); }`, "REINTERPRET_CAST_UNSUPPORTED", "Storage/bind"},
		{`int Run(int value) { return const_cast<int>(value); }`, "CONST_CAST_UNSUPPORTED", "mutable"},
		{`int Run(int value) { return std::move(value); }`, "STD_MOVE_UNSUPPORTED", "move value"},
	}
	for _, tc := range cases {
		_, err := Parse("first_contact.concept", "profile Core; "+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) || !strings.Contains(err.Error(), tc.hint) {
			t.Errorf("%s: %v", tc.source, err)
		}
	}
}

func TestR8eMethodMustUse(t *testing.T) {
	const prefix = `module R8e.Method; profile Core;
class Device {
public:
    int value;
    [[must_use]] int Read() { return value; }
}
`
	_, err := Parse("method.concept", prefix+`int Run() { Device device = Device{3}; device.Read(); return 0; }`)
	if err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
		t.Fatalf("method result was not MustUse: %v", err)
	}
	_, err = Parse("method.concept", prefix+`int Run() { Device device = Device{3}; discard device.Read(); return 0; }`)
	if err != nil {
		t.Fatalf("explicitly discarded method result: %v", err)
	}
}
