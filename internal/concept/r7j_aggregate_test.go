package concept

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7jNativeABIEvidenceDeterministic100(t *testing.T) {
	if testing.Short() {
		t.Skip("100 native compiler probes skipped in short mode")
	}
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	root := t.TempDir()
	for name, body := range map[string]string{
		"manifest.concept": "// isolated ABI determinism fixture\n",
		"bridge.h":         "#include <cstdint>\ntypedef struct Pair { int32_t x; int32_t y; } Pair;\n",
		"bridge.cpp":       "#include \"bridge.h\"\n",
		"Native.concept":   "module Native; profile Core; [[repr(C)]] record struct Pair { int x; int y; }\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	project := NativeProject{Name: "DeterministicABI", Toolchain: "Clang", Root: root,
		Targets:    []NativeTarget{{Name: "bridge", Language: "Cpp", Standard: "Cpp17", Kind: "StaticLibrary", Output: "libbridge.a", Sources: []string{"bridge.cpp"}, Includes: []string{"."}}},
		Companions: []string{"Native.concept"},
		ABI:        []NativeABIClaim{{Header: "bridge.h", TypeName: "Pair", Companion: "Native.concept", Size: 8, Alignment: 4, Fields: []string{"x", "y"}, Offsets: []int{0, 4}}},
	}
	var first []byte
	for run := 0; run < 100; run++ {
		if err := CheckNativeABI(project); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(filepath.Join(project.Root, ".native-build", "abi.json"))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = body
		} else if !bytes.Equal(first, body) {
			t.Fatalf("native ABI evidence changed on run %d", run+1)
		}
	}
}

func TestR7jAggregateDeclarationSurvivesArtifactOnlyImport(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "dogfood", "tinyxml2", "concept", "Native.concept")
	companion, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileSemanticModule("Native.concept", string(companion), nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module Consumer; profile Core; import Native;
requires CAbiValue<ConceptXmlStats>;
int UseStats() {
    ConceptXmlStats value = ConceptXmlRoundTripStats(ConceptXmlStats{1, 2});
    return value.children;
}`
	module, err := ParseWithSemanticModules("Consumer.concept", consumer, map[string][]byte{"Native": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["consumer.generated.h"]), "concept_concept_xml_stats ConceptXmlRoundTripStats(concept_concept_xml_stats stats)") {
		t.Fatalf("artifact-only import lost aggregate C ABI declaration:\n%s", outputs["consumer.generated.h"])
	}
}

func TestR7jCAbiAdmissionRejectsUnsafeRepresentations(t *testing.T) {
	cases := []struct{ name, source, code, reason string }{
		{"missing repr", `profile Core; record struct Pair { int x; int y; } extern "C" Pair Exchange(Pair value);`, "EXTERN_C_ABI_TYPE_INVALID", "lacks [[repr(C)]]"},
		{"ordinary struct", `profile Core; [[repr(C)]] struct Pair { int x; int y; }`, "C_ABI_REPR_INVALID", "nonempty record struct"},
		{"bool field", `profile Core; [[repr(C)]] record struct Pair { bool x; int y; }`, "C_ABI_REPR_INVALID", "field x"},
		{"pointer field", `profile Core; [[repr(C)]] record struct Pair { byte* data; int count; }`, "C_ABI_REPR_INVALID", "field data"},
		{"bad repr", `profile Core; [[repr(packed)]] record struct Pair { int x; int y; }`, "C_ABI_REPR_INVALID", "only [[repr(C)]]"},
		{"enum repr", `profile Core; [[repr(C)]] enum Kind { One, Two, }`, "C_ABI_REPR_INVALID", "non-generic record struct"},
		{"enum boundary", `profile Core; enum Kind { One, Two, } extern "C" Kind GetKind();`, "EXTERN_C_ABI_TYPE_INVALID", "enum lacks an explicit fixed underlying"},
		{"bad offset", `profile Core; [[repr(C)]] record struct Pair { int x; int y; } static_assert(OffsetOf<Pair>(Pair.missing) == 0, "bad");`, "C_ABI_OFFSET_INVALID", "unknown field Pair.missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("invalid_abi.concept", tc.source)
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("expected %s with %q, got %v", tc.code, tc.reason, err)
			}
		})
	}
}

func TestR7jAggregateRoundTripThroughStrictC11(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	const source = `module AbiRoundTrip; profile Core;
[[repr(C)]] record struct Pair { int x; int y; }
[[repr(C)]] record struct Outer { Pair pair; int flags; }
[[repr(C)]] record struct Packet { byte<array>[16] bytes; }
[[repr(C)]] record struct Padded { byte tag; uint64 value; }
static_assert(SizeOf<Pair>() == 8, "pair size");
static_assert(AlignOf<Pair>() == 4, "pair alignment");
static_assert(OffsetOf<Pair>(Pair.y) == 4, "pair offset");
static_assert(SizeOf<Outer>() == 12, "nested size");
static_assert(OffsetOf<Outer>(Outer.flags) == 8, "nested offset");
static_assert(SizeOf<Packet>() == 16, "array size");
static_assert(SizeOf<Padded>() == 16, "padded size");
static_assert(AlignOf<Padded>() == 8, "padded alignment");
static_assert(OffsetOf<Padded>(Padded.value) == 8, "padded offset");
extern "C" Pair MakePair(int x, int y);
extern "C" Pair RoundTripPair(Pair value);
extern "C" Outer RoundTripOuter(Outer value);
extern "C" Packet MakePacket(int seed);
extern "C" Packet RoundTripPacket(Packet value);
extern "C" Padded RoundTripPadded(Padded value);
int Main(int ignored) {
    Pair p = RoundTripPair(MakePair(2, 3));
    Outer o = RoundTripOuter(Outer{p, 7});
    Packet packet = RoundTripPacket(MakePacket(4));
    Padded padded = RoundTripPadded(Padded{1, 9});
    if (o.pair.x == 3 and o.pair.y == 5 and o.flags == 10 and packet.bytes[0] == 9 and padded.value == 15) {
        return 42;
    }
    return 0;
}`
	module, err := Parse("abi_roundtrip.concept", source)
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
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	const native = `#include <stdint.h>
typedef struct Pair { int32_t x; int32_t y; } Pair;
typedef struct Outer { Pair pair; int32_t flags; } Outer;
typedef struct Packet { uint8_t bytes[16]; } Packet;
typedef struct Padded { uint8_t tag; uint64_t value; } Padded;
Pair MakePair(int32_t x, int32_t y) { return (Pair){x, y}; }
Pair RoundTripPair(Pair p) { return (Pair){p.x + 1, p.y + 2}; }
Outer RoundTripOuter(Outer o) { o.flags += 3; return o; }
Packet MakePacket(int32_t seed) { Packet p = {{0}}; p.bytes[0] = (uint8_t)seed; return p; }
Packet RoundTripPacket(Packet p) { p.bytes[0] += 5; return p; }
Padded RoundTripPadded(Padded p) { p.value += 6; return p; }
`
	if err := os.WriteFile(filepath.Join(dir, "native.c"), []byte(native), 0o644); err != nil {
		t.Fatal(err)
	}
	const host = `#include "abi_roundtrip.generated.h"
int main(void) { return concept_abi_round_trip_main(0) == 42 ? 0 : 1; }
`
	if err := os.WriteFile(filepath.Join(dir, "host.c"), []byte(host), 0o644); err != nil {
		t.Fatal(err)
	}
	compilers := []string{compiler}
	if gcc, err := exec.LookPath("gcc"); err == nil {
		compilers = append(compilers, gcc)
	}
	for i, cc := range compilers {
		exe := filepath.Join(dir, "abi"+string(rune('0'+i))+".exe")
		cmd := exec.Command(cc, "-std=c11", "-pedantic-errors", "-Wall", "-Wextra", filepath.Join(dir, "abi_roundtrip.generated.c"), filepath.Join(dir, "native.c"), filepath.Join(dir, "host.c"), "-o", exe)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11 aggregate link failed: %v\n%s", cc, err, output)
		}
		if output, err := exec.Command(exe).CombinedOutput(); err != nil {
			t.Fatalf("%s native aggregate round trip failed: %v\n%s", cc, err, output)
		}
	}
}

func TestR7jNativeProbeComparesNestedArrayAndPadding(t *testing.T) {
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	root := t.TempDir()
	files := map[string]string{
		"manifest.concept": "// test input identity\n",
		"bridge.cpp":       "#include \"bridge.h\"\n",
		"bridge.h": `#include <cstdint>
typedef struct Pair { int32_t x; int32_t y; } Pair;
typedef struct Outer { Pair pair; int32_t flags; } Outer;
typedef struct Packet { uint8_t bytes[16]; int32_t status; } Packet;
typedef struct Padded { uint8_t tag; uint64_t value; } Padded;
`,
		"Native.concept": `module Native; profile Core;
[[repr(C)]] record struct Pair { int x; int y; }
[[repr(C)]] record struct Outer { Pair pair; int flags; }
[[repr(C)]] record struct Packet { byte<array>[16] bytes; int status; }
[[repr(C)]] record struct Padded { byte tag; uint64 value; }
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	project := NativeProject{Name: "AggregateProbe", Toolchain: "Clang", Root: root,
		Targets:    []NativeTarget{{Name: "bridge", Language: "Cpp", Standard: "Cpp17", Kind: "StaticLibrary", Output: "libbridge.a", Sources: []string{"bridge.cpp"}, Includes: []string{"."}}},
		Companions: []string{"Native.concept"},
		ABI: []NativeABIClaim{
			{Header: "bridge.h", TypeName: "Pair", Companion: "Native.concept", Size: 8, Alignment: 4, Fields: []string{"x", "y"}, Offsets: []int{0, 4}},
			{Header: "bridge.h", TypeName: "Outer", Companion: "Native.concept", Size: 12, Alignment: 4, Fields: []string{"pair", "flags"}, Offsets: []int{0, 8}},
			{Header: "bridge.h", TypeName: "Packet", Companion: "Native.concept", Size: 20, Alignment: 4, Fields: []string{"bytes", "status"}, Offsets: []int{0, 16}},
			{Header: "bridge.h", TypeName: "Padded", Companion: "Native.concept", Size: 16, Alignment: 8, Fields: []string{"tag", "value"}, Offsets: []int{0, 8}},
		},
	}
	plan, err := NativeBuildPlan(project)
	if err != nil || len(plan.ABIProbes) != 4 {
		t.Fatalf("native plan omitted ABI probe steps: %v, %+v", err, plan.ABIProbes)
	}
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(root, ".native-build", "abi.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), `"origin": "NativeToolchainProbe"`) || !strings.Contains(string(first), `"target":`) {
		t.Fatalf("probe evidence lacks origin or target: %s", first)
	}
	if _, err := exec.LookPath("g++"); err == nil {
		gccProject := project
		gccProject.Toolchain = "GCC"
		if err := CheckNativeABI(gccProject); err != nil {
			t.Fatalf("GCC ABI probe failed: %v", err)
		}
		gccEvidence, err := os.ReadFile(filepath.Join(root, ".native-build", "abi.json"))
		if err != nil || !strings.Contains(string(gccEvidence), `"compiler": "g++"`) {
			t.Fatalf("GCC evidence missing: %v", err)
		}
	}
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(root, ".native-build", "abi.json"))
	if err != nil || string(first) != string(second) {
		t.Fatalf("ABI evidence changed for identical inputs: %v", err)
	}
	missing := project
	missing.ABI = project.ABI[:3]
	if err := CheckNativeABI(missing); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_CLAIM_MISSING") {
		t.Fatalf("unclaimed repr(C) type was not rejected: %v", err)
	}
	project.ABI[3].Offsets[1] = 4
	if err := CheckNativeABI(project); err == nil || !strings.Contains(err.Error(), "Padded.value") {
		t.Fatalf("incorrect offset was not rejected: %v", err)
	}
	project.ABI[3].Offsets[1] = 8
	if err := os.WriteFile(filepath.Join(root, "bridge.h"), []byte(files["bridge.h"]+"\n// changed native boundary input\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	changed, err := os.ReadFile(filepath.Join(root, ".native-build", "abi.json"))
	if err != nil || string(first) == string(changed) {
		t.Fatalf("native header drift did not invalidate ABI identity: %v", err)
	}
	packed := "#pragma pack(push, 1)\n" + files["bridge.h"] + "\n#pragma pack(pop)\n"
	if err := os.WriteFile(filepath.Join(root, "bridge.h"), []byte(packed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeABI(project); err == nil || !strings.Contains(err.Error(), "NATIVE_ABI_MISMATCH") {
		t.Fatalf("unrepresented native packing was not rejected: %v", err)
	}
}
