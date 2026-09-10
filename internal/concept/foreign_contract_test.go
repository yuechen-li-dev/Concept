package concept

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func foreignMemoryGeometryArtifact(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	return buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(source), nil)
}

func TestForeignInteropFixtureCorpus(t *testing.T) {
	root := "../../language/evt1/tooling/interop"
	geometry := foreignMemoryGeometryArtifact(t)
	host := buildSemanticArtifact(t, "Platform/Host/Memory.concept", foreignHostModuleSource(t), map[string][]byte{"Standard.MemoryGeometry": geometry})
	artifacts := map[string][]byte{"Standard.MemoryGeometry": geometry, "Platform.Host.Memory": host}
	valid, err := filepath.Glob(filepath.Join(root, "valid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 10 {
		t.Fatalf("foreign valid fixture count = %d, want 10", len(valid))
	}
	for _, path := range valid {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if _, parseErr := ParseWithSemanticModules(filepath.ToSlash(path), string(source), artifacts); parseErr != nil {
				t.Fatal(parseErr)
			}
		})
	}
	invalid := map[string]string{
		"foreign_no_contract_unknown.concept":          "CONCEPT_ASSERT_UNKNOWN",
		"foreign_address_from_bits_bind.concept":       "RAW_BIND_PROVENANCE_UNKNOWN",
		"foreign_alignment_strengthen_invalid.concept": "FOREIGN_REGION_GEOMETRY_AUTHORITY_MISMATCH",
		"foreign_region_escape.concept":                "SEMANTIC_REGION_LIFETIME_ESCAPE",
		"foreign_null_region_invalid.concept":          "FOREIGN_CONTRACT_UNKNOWN",
		"foreign_contract_abi_mismatch.concept":        "FOREIGN_CONTRACT_ABI_MISMATCH",
		"foreign_unknown_host_access.concept":          "FOREIGN_HOST_ACCESS_UNKNOWN",
	}
	for name, code := range invalid {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "invalid", name)
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			_, parseErr := ParseWithSemanticModules(filepath.ToSlash(path), string(source), artifacts)
			var diagnostic Diagnostic
			if !errors.As(parseErr, &diagnostic) || diagnostic.Code != code {
				t.Fatalf("expected %s, got %v", code, parseErr)
			}
		})
	}
	tests, err := filepath.Glob(filepath.Join(root, "tests", "*.concept_test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 3 {
		t.Fatalf("foreign concept-test count = %d, want 3", len(tests))
	}
	for _, path := range tests {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			module, parseErr := ParseWithSemanticModules(filepath.ToSlash(path), string(source), artifacts)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if _, generateErr := Generate(module, source); generateErr != nil {
				t.Fatal(generateErr)
			}
		})
	}
}

func foreignHostModuleSource(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("../../language/evt1/tooling/modules/Platform/Host/Memory.concept")
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

func TestForeignContractEstablishesTrustedStorageWithoutTrustingExternByDefault(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	artifact := buildSemanticArtifact(t, "Platform/Host/Memory.concept", foreignHostModuleSource(t), map[string][]byte{"Standard.MemoryGeometry": geometry})
	if !bytes.Contains(artifact, []byte(`"foreign_contracts"`)) || !bytes.Contains(artifact, []byte(`"DeclaredForeign"`)) {
		t.Fatalf("artifact omitted inspectable foreign authority: %s", artifact)
	}
	consumer := `module App;
profile Core;
import Platform.Host.Memory;
int Main()
{
    usize<byte> size = 4;
    usize<byte> alignment = 4;
	usize<byte> tooLarge = 128;
	Assert.Error(AcquireHost(tooLarge, alignment), "null allocation must remain an error");
    owned HostAllocation acquired = AcquireHost(size, alignment)!;
    owned HostAllocation allocation = move acquired;
    MemoryRegion<SystemMemory> region = RegionOf(ref const allocation);
    Assert.Concept<HostAccessible>(region, "foreign host storage must be accessible");
    Assert.Concept<Aligned<4>>(region, "foreign host storage must be aligned");
    Storage<int> storage = bind<int>(region.start, region.length);
    ref int value = Initialize(storage, 19);
    int observed = value;
    Destroy(storage);
    return observed;
}`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Platform.Host.Memory": artifact, "Standard.MemoryGeometry": geometry})
	if err != nil {
		t.Fatalf("%v\nartifact=%s", err, artifact)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	generatedC := moduleOutput(t, outputs, ".generated.c")
	if !strings.Contains(mir, `"allocation_effect_origin": "DeclaredForeign"`) || !strings.Contains(mir, `"origin": "DeclaredForeign"`) || !strings.Contains(mir, `"authority": "HostAllocationContract"`) || strings.Contains(generatedC, "HostAllocationContract") {
		t.Fatalf("foreign authority must be visible in MIR and erased from C\nMIR:%s\nC:%s", mir, generatedC)
	}
	for _, forbidden := range []string{"contract_registry", "provenance_registry", "region_id", "malloc(", "calloc(", "realloc("} {
		if strings.Contains(strings.ToLower(generatedC), forbidden) {
			t.Fatalf("generated C contains forbidden semantic runtime %q", forbidden)
		}
	}
	harness := `#include "app.generated.h"
#include <stddef.h>
#include <stdint.h>

_Alignas(64) static uint8_t host_storage[64];
static int allocations;
static int frees;

uint8_t* ConceptHostAllocate(size_t size, size_t alignment) {
    allocations += 1;
    if (size > sizeof(host_storage) || alignment > _Alignof(host_storage)) return 0;
    return host_storage;
}
void ConceptHostFree(uint8_t* address) {
    if (address == host_storage) frees += 1;
}
bool ConceptHostAddressIsNull(uint8_t* address) { return address == 0; }

int main(void) {
    int result = concept_app_main();
    return result == 19 && allocations == 2 && frees == 1 ? 0 : 1;
}
`
	runFoundationNativeHarness(t, outputs, "foreign_host_harness.c", harness)
}

func TestForeignContractRejectsUncontractedAndMismatchedAuthority(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	cases := []struct{ name, source, code string }{
		{"unknown_contract", `module Bad; profile Core; import Standard.MemoryGeometry; struct Owner { byte* address; usize<byte> length; usize<byte> alignment; } MemoryRegion<SystemMemory> Bad(ref Owner owner) { return EstablishExternalRegion<SystemMemory>(owner, owner.address, owner.length, owner.alignment, "Missing"); }`, "FOREIGN_CONTRACT_UNKNOWN"},
		{"abi_mismatch", `module Bad; profile Core; import Standard.MemoryGeometry; extern "C" byte* Host(int size, usize alignment); foreign concept BadContract on Host { requires compiler.ExternalStorage<SystemMemory>(result, size, alignment); requires compiler.HostAccessible(result); }`, "FOREIGN_CONTRACT_ABI_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseWithSemanticModules(tc.name+".concept", tc.source, map[string][]byte{"Standard.MemoryGeometry": geometry})
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestAddressFromBitsRemainsUntrustedAfterForeignContracts(t *testing.T) {
	source := `profile Core; struct SystemMemory {} void Bad(usize bits, usize<byte> extent) { Address<SystemMemory> address = AddressFromBits<SystemMemory>(bits); Storage<int> storage = bind<int>(address, extent); }`
	_, err := Parse("foreign_bits.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "RAW_BIND_PROVENANCE_UNKNOWN" {
		t.Fatalf("expected unchanged untrusted bits rule, got %v", err)
	}
}

func TestForeignStorageAuthorityCannotEscapeOrOutliveOwner(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	artifact := buildSemanticArtifact(t, "Platform/Host/Memory.concept", foreignHostModuleSource(t), map[string][]byte{"Standard.MemoryGeometry": geometry})
	cases := []struct {
		name, body, code string
	}{
		{
			name: "escape",
			body: `MemoryRegion<SystemMemory> Bad() {
    usize<byte> size = 4; usize<byte> alignment = 4;
    owned HostAllocation allocation = AcquireHost(size, alignment)!;
    return RegionOf(ref const allocation);
}`,
			code: "SEMANTIC_REGION_LIFETIME_ESCAPE",
		},
		{
			name: "use_after_drop",
			body: `void Bad() {
    usize<byte> size = 4; usize<byte> alignment = 4;
    owned HostAllocation allocation = AcquireHost(size, alignment)!;
    MemoryRegion<SystemMemory> region = RegionOf(ref const allocation);
    Drop(move allocation);
    Storage<int> storage = bind<int>(region.start, region.length);
}`,
			code: "FOREIGN_STORAGE_AUTHORITY_ENDED",
		},
		{
			name: "consumer_forges_region",
			body: `MemoryRegion<SystemMemory> Bad(ref const HostAllocation allocation) {
    return EstablishExternalRegion<SystemMemory>(allocation, allocation.address, allocation.length, allocation.alignment, "HostAllocationContract");
}`,
			code: "FOREIGN_CONTRACT_CONTEXT_REQUIRED",
		},
		{
			name: "forged_wrapper_has_no_authority",
			body: `extern "C" byte* OpaqueAddress(usize<byte> size);
void Bad() {
    usize<byte> size = 4; usize<byte> alignment = 4;
    HostAllocation forged = HostAllocation{OpaqueAddress(size), size, alignment};
    MemoryRegion<SystemMemory> region = RegionOf(ref const forged);
    Storage<int> storage = bind<int>(region.start, region.length);
}`,
			code: "RAW_BIND_PROVENANCE_UNKNOWN",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "module App; profile Core; import Platform.Host.Memory;\n" + tc.body
			_, err := ParseWithSemanticModules(tc.name+".concept", source, map[string][]byte{"Platform.Host.Memory": artifact, "Standard.MemoryGeometry": geometry})
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestForeignStorageOwnerCannotBeCopied(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	artifact := buildSemanticArtifact(t, "Platform/Host/Memory.concept", foreignHostModuleSource(t), map[string][]byte{"Standard.MemoryGeometry": geometry})
	source := `module App; profile Core; import Platform.Host.Memory;
void Bad(usize<byte> size, usize<byte> alignment)
{
    owned HostAllocation first = AcquireHost(size, alignment)!;
    owned HostAllocation duplicate = first;
}`
	_, err := ParseWithSemanticModules("foreign_owner_copy.concept", source, map[string][]byte{"Platform.Host.Memory": artifact, "Standard.MemoryGeometry": geometry})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "COPY_OF_NONCOPYABLE" {
		t.Fatalf("expected owned wrapper copy rejection, got %v", err)
	}
}

func TestForeignTemplateFactsAreExplicitAndBounded(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	cases := []struct{ name, source, code string }{
		{"non_foreign_target", `module Bad; profile Core; import Standard.MemoryGeometry; byte* Host(usize<byte> size, usize<byte> alignment); foreign concept C on Host { requires compiler.ExternalStorage<SystemMemory>(result, size, alignment); requires compiler.HostAccessible(result); }`, "FOREIGN_CONTRACT_TARGET_INVALID"},
		{"non_pointer_result", `module Bad; profile Core; import Standard.MemoryGeometry; extern "C" int Host(usize<byte> size, usize<byte> alignment); foreign concept C on Host { requires compiler.ExternalStorage<SystemMemory>(result, size, alignment); requires compiler.HostAccessible(result); }`, "FOREIGN_CONTRACT_ABI_MISMATCH"},
		{"missing_host_access", `module Bad; profile Core; import Standard.MemoryGeometry; extern "C" byte* Host(usize<byte> size, usize<byte> alignment); foreign concept C on Host { requires compiler.ExternalStorage<SystemMemory>(result, size, alignment); } struct Owner { byte* address; usize<byte> length; usize<byte> alignment; } MemoryRegion<SystemMemory> Bad(ref Owner owner) { return EstablishExternalRegion<SystemMemory>(owner, owner.address, owner.length, owner.alignment, "C"); }`, "FOREIGN_HOST_ACCESS_UNKNOWN"},
		{"alignment_strengthen", `module Bad; profile Core; import Standard.MemoryGeometry; extern "C" byte* Host(usize<byte> size, usize<byte> alignment); foreign concept C on Host { requires compiler.ExternalStorage<SystemMemory>(result, size, alignment); requires compiler.HostAccessible(result); } struct Owner { byte* address; usize<byte> length; usize<byte> alignment; } MemoryRegion<SystemMemory> Bad(ref Owner owner) { usize<byte> stronger = 64; return EstablishExternalRegion<SystemMemory>(owner, owner.address, owner.length, stronger, "C"); }`, "FOREIGN_REGION_GEOMETRY_AUTHORITY_MISMATCH"},
		{"superseded_template_keyword", `module Bad; profile Core; import Standard.MemoryGeometry; extern "C" byte* Host(usize<byte> size, usize<byte> alignment); foreign template C for Host { satisfies compiler.ExternalStorage<SystemMemory>(result, size, alignment); }`, "CV4020"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseWithSemanticModules(tc.name+".concept", tc.source, map[string][]byte{"Standard.MemoryGeometry": geometry})
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestForeignAllocationProofReportsDeclaredForeign(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	source := foreignHostModuleSource(t) + `
int RejectFalseClaim()
{
    Assert.Concept<NoAllocation>(ConceptHostAllocate, "foreign allocation is declared");
    return 0;
}`
	_, err := ParseWithSemanticModules("Platform/Host/Memory.concept", source, map[string][]byte{"Standard.MemoryGeometry": geometry})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_DISPROVEN" || diagnostic.Proof == nil {
		t.Fatalf("expected disproven proof, got %v", err)
	}
	if rendered := RenderProofVerbose(*diagnostic.Proof); !strings.Contains(rendered, "DeclaredForeign") || !strings.Contains(rendered, "ConceptHostAllocate Allocates") {
		t.Fatalf("proof omitted foreign declaration authority:\n%s", rendered)
	}
}

func TestForeignInteropArtifactsAndOutputsAreDeterministic(t *testing.T) {
	geometry := foreignMemoryGeometryArtifact(t)
	source := foreignHostModuleSource(t)
	baseline, err := CompileSemanticModule("Platform/Host/Memory.concept", source, map[string][]byte{"Standard.MemoryGeometry": geometry})
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		actual, compileErr := CompileSemanticModule("Platform/Host/Memory.concept", source, map[string][]byte{"Standard.MemoryGeometry": geometry})
		if compileErr != nil {
			t.Fatal(compileErr)
		}
		if !bytes.Equal(actual, baseline) {
			t.Fatalf("foreign semantic artifact changed on run %d", run+1)
		}
	}
	consumer := `module App; profile Core; import Platform.Host.Memory;
void Check(usize<byte> size)
{
	usize<byte> alignment = 4;
	owned HostAllocation allocation = AcquireHost(size, alignment)!;
    MemoryRegion<SystemMemory> region = RegionOf(ref const allocation);
    Assert.Concept<HostAccessible>(region, "foreign fact must retain its origin");
}`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Platform.Host.Memory": baseline, "Standard.MemoryGeometry": geometry})
	if err != nil {
		t.Fatal(err)
	}
	want, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		actual, generateErr := Generate(module, []byte(consumer))
		if generateErr != nil {
			t.Fatal(generateErr)
		}
		for name, expected := range want {
			if !bytes.Equal(actual[name], expected) {
				t.Fatalf("%s changed on run %d", name, run+1)
			}
		}
	}
}
