package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const genericSemanticModule = `module Standard.Generic;
profile Core;
template <typename T>
struct Box { T value; };
template <typename T, usize Capacity>
struct FixedStorage { T<array>[Capacity] values; };
template <typename T>
T Identity(T value) { return value; }
template <typename T>
class Holder
{
public:
    T value;
    T Get(ref const Holder self) { return self.value; }
};
template <typename Callback>
class Handler { public: Callback callback; };
template <typename T>
ref struct BufferView { Span<T> values; };
`

func buildSemanticArtifact(t *testing.T, path, source string, dependencies map[string][]byte) []byte {
	t.Helper()
	body, err := CompileSemanticModule(path, source, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func moduleOutput(t *testing.T, outputs Outputs, suffix string) string {
	t.Helper()
	for name, body := range outputs {
		if strings.HasSuffix(name, suffix) {
			return string(body)
		}
	}
	t.Fatalf("no output ending in %s; outputs=%v", suffix, outputs)
	return ""
}

func TestSemanticModuleArtifactIsInspectableAndByteIdenticalAcross100Runs(t *testing.T) {
	first := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil)
	if bytes.Contains(first, []byte("module Standard.Generic")) || bytes.Contains(first, []byte("generated.c")) {
		t.Fatal("semantic artifact embedded source or backend output")
	}
	artifact, module, err := LoadSemanticModuleArtifact(first)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.SchemaVersion != SemanticModuleSchema || artifact.ModuleIdentity != "Standard.Generic" || module.Name != "Standard.Generic" {
		t.Fatalf("wrong artifact identity: %#v", artifact)
	}
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil); !bytes.Equal(got, first) {
			t.Fatalf("module artifact changed on run %d", run)
		}
	}
}

func TestImportedGenericTypesFunctionsMethodsAndNonTypeParametersUseLocalTypes(t *testing.T) {
	artifact := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil)
	source := `module App;
profile Core;
import Standard.Generic;
struct Widget { int id; }
auto MakeCallback() { return callback() { return 3; }; }
using WidgetBox = Box<Widget>;
using Callback = typeof(MakeCallback());
static_assert(SizeOf<Box<Widget>>() == 4, "box layout");
static_assert(SizeOf<WidgetBox>() == 4, "imported alias layout");
static_assert(AlignOf<FixedStorage<Widget, 4>>() == 4, "storage alignment");
int Invoke(ref const Handler<Callback> handler) { return handler.callback(); }
int Inspect(ref const BufferView<int> view) { return Len(view.values); }
int Main()
{
    Widget widget = Widget{7};
    WidgetBox box = Box<Widget>{Identity<Widget>(widget)};
    Holder<int> holder = Holder<int>{box.value.id};
    int<array>[2] values = [1, 2];
    Span<int> span = Span(values);
    BufferView<int> view = BufferView<int>{span};
    return holder.Get() + Inspect(ref const view);
}

`
	module, err := ParseWithSemanticModules("App.concept", source, map[string][]byte{"Standard.Generic": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := moduleOutput(t, outputs, ".generated.h")
	body := moduleOutput(t, outputs, ".generated.c")
	for _, want := range []string{"concept_box_widget_", "concept_handler_make_callback_callback0_", "concept_buffer_view_int_"} {
		if !strings.Contains(header, want) {
			t.Fatalf("imported generic output omitted %q:\n%s", want, header)
		}
	}
	for _, want := range []string{"concept_template_identity__widget", "concept_app_get"} {
		if !strings.Contains(strings.ToLower(body), want) {
			t.Fatalf("imported generic implementation omitted %q:\n%s", want, body)
		}
	}
}

func TestImportedGenericMethodSubstitutesForeachStatements(t *testing.T) {
	producer := `module Standard.Bounded;
profile Core;
template <typename Configuration, usize Capacity>
class Counter
{
public:
    int<array>[Capacity] values;
    int Sum(ref const Counter self)
    {
        int total = 0;
        foreach (int value in self.values) { total = total + value; }
        return total;
    }
};
`
	artifact := buildSemanticArtifact(t, "Standard/Bounded.concept", producer, nil)
	consumer := `module App;
profile Core;
import Standard.Bounded;
record struct AppConfiguration { int identity; };
int Main()
{
    Counter<AppConfiguration, 3> counter = Counter<AppConfiguration, 3>{[1, 2, 3]};
    Counter<AppConfiguration, 2> smaller = Counter<AppConfiguration, 2>{[4, 5]};
    return counter.Sum() + smaller.Sum();
}

`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Standard.Bounded": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	body := moduleOutput(t, outputs, ".generated.c")
	if !strings.Contains(body, "concept_app_sum") {
		t.Fatal("imported generic foreach method was not emitted")
	}
	if strings.Count(body, "concept_app_sum__") < 4 {
		t.Fatalf("distinct non-type generic method instances were not retained:\n%s", body)
	}
}

func TestImportedGenericNestedSymbolicNonTypeArgumentCloses(t *testing.T) {
	producer := `module Standard.Nested;
profile Core;
template <typename Configuration, usize Capacity>
struct Inner { int<array>[Capacity] values; };
template <typename Configuration, usize Capacity>
struct Outer { Inner<Configuration, Capacity> inner; };
`
	artifact := buildSemanticArtifact(t, "Standard/Nested.concept", producer, nil)
	consumer := `module App;
profile Core;
import Standard.Nested;
record struct AppConfiguration { int identity; };
usize Main() { return SizeOf<Outer<AppConfiguration, 3>>(); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Standard.Nested": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(consumer)); err != nil {
		t.Fatal(err)
	}
}

func TestR6gImportedMultiParameterFunctionTemplateInstantiatesWithoutSourceReparse(t *testing.T) {
	producer := `module Standard.Geometry;
profile Core;
template <typename T, typename U, usize Tag>
T First(T first, U second) { return first; }
`
	artifact := buildSemanticArtifact(t, "Standard/Geometry.concept", producer, nil)
	consumer := `module App;
profile Core;
import Standard.Geometry;
int Main() { uint other = 2; return First<int, uint, 7>(3, other); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Standard.Geometry": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ToLower(moduleOutput(t, outputs, ".generated.c"))
	if !strings.Contains(body, "concept_template_first__int__uint__7") {
		t.Fatalf("imported multi-parameter instance missing:\n%s", body)
	}
}

func TestImportedGenericConstraintIsCheckedAgainstConsumerType(t *testing.T) {
	library := `module Standard.Constrained;
profile Core;
concept Readable<T> { requires int Read(T value); }
template <typename T>
requires Readable<T>
class Owner { public: T value; };
`
	artifact := buildSemanticArtifact(t, "Standard/Constrained.concept", library, nil)
	valid := `module App; profile Core; import Standard.Constrained;
struct Resource { int id; }
int Read(Resource value) { return value.id; }
usize Main() { return SizeOf<Owner<Resource>>(); }
`
	if _, err := ParseWithSemanticModules("valid.concept", valid, map[string][]byte{"Standard.Constrained": artifact}); err != nil {
		t.Fatal(err)
	}
	invalid := `module App; profile Core; import Standard.Constrained;
struct ImmovableThing { int id; }
usize Main() { return SizeOf<Owner<ImmovableThing>>(); }
`
	_, err := ParseWithSemanticModules("invalid.concept", invalid, map[string][]byte{"Standard.Constrained": artifact})
	if err == nil || !strings.Contains(err.Error(), "Readable") || !strings.Contains(err.Error(), "ImmovableThing") {
		t.Fatalf("expected concrete imported constraint failure, got %v", err)
	}
}

func TestImportedGenericPreservesDropAndReferenceEscapeSemantics(t *testing.T) {
	library := `module Standard.Ownership; profile Core;
template <typename T> struct Owner { owned T value; };
template <typename T> ref struct View { ref T value; };
`
	artifact := buildSemanticArtifact(t, "Standard/Ownership.concept", library, nil)
	dropSource := `module App; profile Core; import Standard.Ownership;
struct Resource { int handle; }
void Drop(owned Resource value);
int Main() { owned Owner<Resource> owner = Owner<Resource>{Resource{7}}; return 0; }
`
	module, err := ParseWithSemanticModules("drop.concept", dropSource, map[string][]byte{"Standard.Ownership": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(dropSource))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(moduleOutput(t, outputs, ".generated.c"), "concept_drop_drop((owner).value);"); count != 1 {
		t.Fatalf("imported generic emitted %d resource drops", count)
	}
	escapeSource := `module App; profile Core; import Standard.Ownership;
View<int> Bad() { int local = 1; View<int> view = View<int>{ref local}; return view; }
`
	_, err = ParseWithSemanticModules("escape.concept", escapeSource, map[string][]byte{"Standard.Ownership": artifact})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4521" {
		t.Fatalf("expected ordinary imported ref escape rejection, got %v", err)
	}
}

func TestImportedEffectSummariesProveDisproveAndPreserveUnknown(t *testing.T) {
	library := `module Standard.Host; profile Core;
extern "C" byte* HostAllocate(usize size);
requires compiler.Allocates(HostAllocate);
extern "C" int HostOpaque();
int Acquire(usize size) { byte* storage = HostAllocate(size); return 1; }
int Pure() { return 1; }
int OpaqueWrapper() { return HostOpaque(); }
`
	artifactBody := buildSemanticArtifact(t, "Standard/Host.concept", library, nil)
	artifact, _, err := LoadSemanticModuleArtifact(artifactBody)
	if err != nil {
		t.Fatal(err)
	}
	effects := map[string]string{}
	for _, summary := range artifact.OperationEffects {
		effects[summary.Operation] = summary.Effect
	}
	if effects["Acquire"] != "Allocates" || effects["Pure"] != "NoAllocation" || effects["OpaqueWrapper"] != "Unknown" {
		t.Fatalf("unexpected module effect lattice: %#v", effects)
	}
	cases := []struct {
		name, call, code string
	}{
		{"proven", "Pure()", ""},
		{"disproven", "Acquire(SizeOf<int>())", "CONCEPT_ASSERT_DISPROVEN"},
		{"unknown", "OpaqueWrapper()", "CONCEPT_ASSERT_UNKNOWN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "module App; profile Core; import Standard.Host;\nint Prepare() { " + tc.call + "; return 0; }\nint Verify() { Assert.Concept<NoAllocation>(Prepare, \"module proof\"); return 0; }\n"
			_, err := ParseWithSemanticModules(tc.name+".concept", source, map[string][]byte{"Standard.Host": artifactBody})
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code || diagnostic.Proof == nil {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
			if !strings.Contains(RenderProofVerbose(*diagnostic.Proof), "ModuleSummaryEffect") {
				t.Fatalf("proof omitted module summary origin: %s", RenderProofVerbose(*diagnostic.Proof))
			}
		})
	}
}

func TestImportedInterfaceOperationAllocationAllowanceIsPreserved(t *testing.T) {
	contract := `module Standard.AllocatorContract; profile Core;
interface Allocator<T>
{
    requires int Allocate(ref T self, usize size);
    requires compiler.Allocates(Allocate);
}
`
	artifact := buildSemanticArtifact(t, "Standard/AllocatorContract.concept", contract, nil)
	consumer := `module App; profile Core; import Standard.AllocatorContract;
struct Arena { int used; }
int Allocate(ref Arena self, usize size) { self.used = self.used + 1; return self.used; }
requires compiler.Allocates(Allocate);
requires Allocator<Arena>;
int Main() { Arena arena = Arena{0}; return Allocate(ref arena, SizeOf<int>()); }
`
	module, err := ParseWithSemanticModules("interface_effect.concept", consumer, map[string][]byte{"Standard.AllocatorContract": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(moduleOutput(t, outputs, ".mir.json"), `"Name": "Allocates"`) && !strings.Contains(moduleOutput(t, outputs, ".mir.json"), `"name": "Allocates"`) {
		t.Fatal("MIR omitted imported interface allocation allowance")
	}

	strictContract := `module Standard.StrictContract; profile Core;
interface StrictAllocator<T> { requires int Allocate(ref T self, usize size); }
`
	strictArtifact := buildSemanticArtifact(t, "Standard/StrictContract.concept", strictContract, nil)
	strictConsumer := strings.ReplaceAll(consumer, "Standard.AllocatorContract", "Standard.StrictContract")
	strictConsumer = strings.ReplaceAll(strictConsumer, "Allocator<Arena>", "StrictAllocator<Arena>")
	strictModule, err := ParseWithSemanticModules("interface_effect_mismatch.concept", strictConsumer, map[string][]byte{"Standard.StrictContract": strictArtifact})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "INTERFACE_EFFECT_MISMATCH" {
		t.Fatalf("expected interface effect mismatch, got %v (effects=%#v assertions=%#v source=%s)", err, strictModule.OperationEffects, strictModule.Assertions, strictConsumer)
	}
}

func TestSemanticModuleGraphRejectsMissingCycleSchemaAndCorruption(t *testing.T) {
	if _, err := ParseWithSemanticModules("app.concept", "module App; profile Core; import Missing;", nil); err == nil || !strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
		t.Fatalf("expected missing import, got %v", err)
	}
	leaf := buildSemanticArtifact(t, "A.concept", "module A; profile Core;", nil)
	var a SemanticModuleArtifact
	if err := json.Unmarshal(leaf, &a); err != nil {
		t.Fatal(err)
	}
	b := a
	a.ModuleIdentity, a.SourceIdentity = "A", "A"
	b.ModuleIdentity, b.SourceIdentity = "B", "B"
	aModule, _ := encodeSemanticModule(Module{Name: "A", Path: "A", Profile: "Core"})
	bModule, _ := encodeSemanticModule(Module{Name: "B", Path: "B", Profile: "Core"})
	a.SemanticPayload, b.SemanticPayload = aModule, bModule
	a.Dependencies = []SemanticModuleDependency{{ModuleIdentity: "B"}}
	b.Dependencies = []SemanticModuleDependency{{ModuleIdentity: "A"}}
	// Cycles cannot possess mutually closed hashes. The graph detector runs
	// before dependency-hash validation, so placeholders are sufficient.
	a.Dependencies[0].ContentSHA256 = "cycle-b"
	b.Dependencies[0].ContentSHA256 = "cycle-a"
	a.ContentSHA256, _ = semanticArtifactHash(a)
	b.ContentSHA256, _ = semanticArtifactHash(b)
	aBody, _ := json.Marshal(a)
	bBody, _ := json.Marshal(b)
	_, err := ParseWithSemanticModules("app.concept", "module App; profile Core; import A;", map[string][]byte{"A": aBody, "B": bBody})
	if err == nil || !strings.Contains(err.Error(), "MODULE_IMPORT_CYCLE") {
		t.Fatalf("expected cycle rejection, got %v", err)
	}
	corrupt := append([]byte{}, leaf...)
	corrupt[len(corrupt)-2] ^= 1
	if _, _, err := LoadSemanticModuleArtifact(corrupt); err == nil || !strings.Contains(err.Error(), "MODULE_ARTIFACT_CORRUPT") {
		t.Fatalf("expected corruption rejection, got %v", err)
	}
	var unsupported SemanticModuleArtifact
	_ = json.Unmarshal(leaf, &unsupported)
	unsupported.SchemaVersion = "concept-module.v2"
	unsupported.ContentSHA256, _ = semanticArtifactHash(unsupported)
	unsupportedBody, _ := json.Marshal(unsupported)
	if _, _, err := LoadSemanticModuleArtifact(unsupportedBody); err == nil || !strings.Contains(err.Error(), "MODULE_SCHEMA_UNSUPPORTED") {
		t.Fatalf("expected schema rejection, got %v", err)
	}
}

func TestCrossModuleProgramCompilesAndRunsAsStrictC11(t *testing.T) {
	compiler, err := exec.LookPath("zig")
	if err != nil {
		t.Skip("zig is not installed")
	}
	generic := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil)
	hostSource := `module Platform.Host; profile Core;
extern "C" byte* ConceptHostAllocate(usize size);
requires compiler.Allocates(ConceptHostAllocate);
int Acquire(usize size) { byte* storage = ConceptHostAllocate(size); return 1; }
`
	host := buildSemanticArtifact(t, "Platform/Host.concept", hostSource, nil)
	consumer := `module App; profile Core; import Standard.Generic; import Platform.Host;
int Main()
{
    Box<int> box = Box<int>{Identity<int>(7)};
    usize bytes = SizeOf<int>();
    int acquired = Acquire(bytes);
    return box.value + acquired;
}
`
	module, err := ParseWithSemanticModules("app.concept", consumer, map[string][]byte{"Standard.Generic": generic, "Platform.Host": host})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var generatedC string
	for name, contents := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, contents, 0o644); err != nil {
				t.Fatal(err)
			}
			if strings.HasSuffix(name, ".generated.c") {
				generatedC = path
			}
		}
	}
	hostC := `#include "app.generated.h"
static uint8_t storage[64];
uint8_t* ConceptHostAllocate(size_t size) { return size <= sizeof(storage) ? storage : 0; }
int main(void) { return concept_app_main() == 8 ? 0 : 1; }
`
	hostPath := filepath.Join(dir, "host.c")
	if err := os.WriteFile(hostPath, []byte(hostC), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "module.exe")
	command := exec.Command(compiler, "cc", "-std=c11", "-Wall", "-Wextra", "-I", dir, generatedC, hostPath, "-o", exe)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 module compile failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(exe).CombinedOutput(); err != nil {
		t.Fatalf("strict C11 module execution failed: %v\n%s", err, output)
	}
}

func TestFilesystemResolverUsesExactRootsAndRejectsStaleOrDuplicateIdentity(t *testing.T) {
	artifact := buildSemanticArtifact(t, "Standard/Generic.concept", genericSemanticModule, nil)
	writeRoot := func(root string) {
		t.Helper()
		dir := filepath.Join(root, "Standard")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Generic.concept-module.json"), artifact, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Generic.concept"), []byte(genericSemanticModule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	writeRoot(root)
	consumer := `module App; profile Core; import Standard.Generic; usize Main() { return SizeOf<Box<int>>(); }`
	if _, err := ParseWithSemanticModuleRoots("app.concept", consumer, []string{root}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Standard", "Generic.concept"), []byte(genericSemanticModule+"\n// changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWithSemanticModuleRoots("app.concept", consumer, []string{root}); err == nil || !strings.Contains(err.Error(), "MODULE_ARTIFACT_STALE") {
		t.Fatalf("expected stale artifact rejection, got %v", err)
	}
	second := t.TempDir()
	writeRoot(second)
	if _, err := ResolveSemanticModuleArtifacts([]string{second, second}, []string{"Standard.Generic"}); err != nil {
		t.Fatalf("duplicate root spelling should be normalized: %v", err)
	}
	writeRoot(root)
	if _, err := ResolveSemanticModuleArtifacts([]string{root, second}, []string{"Standard.Generic"}); err == nil || !strings.Contains(err.Error(), "MODULE_IDENTITY_DUPLICATE") {
		t.Fatalf("expected duplicate module identity rejection, got %v", err)
	}
}
