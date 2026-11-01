package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func assertEVT2h7Promotion(t *testing.T, m Module) {
	t.Helper()
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("CMIRAMDA")) {
		t.Fatal("missing checked failure contract")
	}
	if _, err = DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	t.Log("historical unsupported specimen promoted by EVT2h7; checked CMIRAMD10 roundtrip")
}

func TestEVT2h7CheckedFallible(t *testing.T) {
	_, _, lir, m, data := fallibleFixture(t)
	if m.Fallible == nil {
		t.Fatal("missing checked failure contract")
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	local := 0
	for _, f := range lir.Functions {
		if f.Fallible != nil {
			for _, s := range f.Fallible.Sites {
				kinds[s.Kind]++
				if s.Handler >= 0 {
					local++
				}
			}
		}
	}
	if kinds["require_success"] < 5 || kinds["propagate"] < 10 || local < 10 {
		t.Fatalf("insufficient fixture: %v local=%d", kinds, local)
	}
	t.Logf("%d functions, %d tries, %d handlers, sites=%v; artifact=%d sha256=%x", len(lir.Functions), len(m.Fallible.Tries), len(m.Fallible.Handlers), kinds, len(data), sha256.Sum256(data))
}

func TestEVT2h7FreshDeterminism(t *testing.T) {
	start := time.Now()
	_, _, lir, m, data := fallibleFixture(t)
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "6c2bc22978e37d03a4561910f2fa1646ee4fc5980d2b677c75ef65ceaae94bf7" {
		t.Fatalf("exact fallible artifact changed: %s", got)
	}
	for trial := 0; trial < resultSamples(); trial++ {
		_, _, next, other, raw := fallibleFixture(t)
		if !bytes.Equal(raw, data) || next.String() != lir.String() || other.String() != m.String() {
			t.Fatalf("fresh trial %d differs", trial)
		}
		decoded, err := DecodeMachineBridge(raw)
		if err != nil || decoded.String() != m.String() {
			t.Fatalf("roundtrip: %v", err)
		}
	}
	t.Logf("%d fresh checked CFG/artifact/codec generations in %s; sha256=%x", resultSamples(), time.Since(start), sha256.Sum256(data))
}

func TestEVT2h7SourceDiagnostics(t *testing.T) {
	paths, err := filepath.Glob("../../language/evt2h7/invalid/*.concept")
	if err != nil || len(paths) == 0 {
		t.Fatal("missing negative fixtures", err)
	}
	paths = append(paths, "../../language/evt1/failure/invalid/try_except_duplicate_error.concept", "../../language/evt1/failure/invalid/try_except_unhandled_error.concept", "../../language/evt1/failure/invalid/question_result_wrong_error.concept", "../../language/evt1/failure/invalid/question_option_in_result.concept")
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(path, string(source))
			if err == nil {
				t.Fatal("accepted invalid failure source")
			}
			if strings.HasPrefix(string(source), "// expect: ") {
				want := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(source), "\n", 2)[0], "// expect: "))
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("want %s got %v", want, err)
				}
			}
			t.Log(err)
		})
	}
}

func TestEVT2h7LIRAdmission(t *testing.T) {
	_, _, lir, _, _ := fallibleFixture(t)
	raw, _ := json.Marshal(lir)
	owner := func(m *LIRModule, name string) *LIRFunction {
		for i := range m.Functions {
			if m.Functions[i].Name == name {
				return &m.Functions[i]
			}
		}
		t.Fatal(name)
		return nil
	}
	for name, mutate := range map[string]func(*LIRModule){
		"missing control": func(m *LIRModule) { owner(m, "RequireResult").Fallible = nil },
		"duplicate site": func(m *LIRModule) {
			f := owner(m, "RequireResult")
			f.Fallible.Sites = append(f.Fallible.Sites, f.Fallible.Sites[0])
		},
		"wrong success variant": func(m *LIRModule) {
			f := owner(m, "RequireResult")
			p := f.Patterns[0]
			f.Blocks[p.Arms[0].Dispatch].Instructions[1].Member = 1
		},
		"inactive payload": func(m *LIRModule) {
			f := owner(m, "RequireResult")
			p := f.Patterns[0]
			f.Blocks[p.Arms[0].Body].Instructions[1].Member = 1
		},
		"wrong handler": func(m *LIRModule) { f := owner(m, "Local"); f.Fallible.Sites[0].Handler = 1 },
		"handler bypass": func(m *LIRModule) {
			f := owner(m, "Local")
			p := f.Patterns[0]
			f.Blocks[p.Arms[1].Body].Term.True = f.Fallible.Tries[0].Join
		},
		"forged binding": func(m *LIRModule) {
			f := owner(m, "Local")
			f.Fallible.Handlers[0].BindingSlot = f.Fallible.Handlers[0].InputSlot
		},
		"handler entry":            func(m *LIRModule) { f := owner(m, "Local"); f.Fallible.Handlers[0].Entry = 0 },
		"scope escape":             func(m *LIRModule) { f := owner(m, "Local"); f.Fallible.Handlers[0].Scope = 1 },
		"invalid try":              func(m *LIRModule) { f := owner(m, "Local"); f.Fallible.Handlers[0].Try = 99 },
		"wrong error nominal type": func(m *LIRModule) { f := owner(m, "Local"); f.Fallible.Handlers[0].Type = f.Fallible.Handlers[1].Type },
		"missing transfer": func(m *LIRModule) {
			f := owner(m, "Local")
			p := f.Patterns[0]
			f.Blocks[p.Arms[1].Body].Instructions = f.Blocks[p.Arms[1].Body].Instructions[:4]
		},
		"nonnearest handler": func(m *LIRModule) { f := owner(m, "Nearest"); f.Fallible.Sites[0].Handler = 1 },
		"invalid return destination": func(m *LIRModule) {
			f := owner(m, "ForwardOption")
			p := f.Patterns[0]
			b := &f.Blocks[p.Arms[1].Body]
			b.Instructions[len(b.Instructions)-1].Member = 0
		},
		"uninitialized success": func(m *LIRModule) {
			f := owner(m, "RequireResult")
			p := f.Patterns[0]
			b := &f.Blocks[p.Arms[0].Body]
			b.Instructions = b.Instructions[:len(b.Instructions)-2]
		},
		"void payload":      func(m *LIRModule) { f := owner(m, "RequireVoid"); f.Patterns[0].Arms[0].Bindings = []int{0} },
		"borrow truncation": func(m *LIRModule) { owner(m, "Local").Borrow.SlotScopes = nil },
		"capacity": func(m *LIRModule) {
			f := owner(m, "Local")
			for len(f.Fallible.Sites) < 65 {
				f.Fallible.Sites = append(f.Fallible.Sites, f.Fallible.Sites[0])
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			var next LIRModule
			if err := json.Unmarshal(raw, &next); err != nil {
				t.Fatal(err)
			}
			mutate(&next)
			if err := VerifyLIR(next); err == nil {
				t.Fatal("admitted malformed failure LIR")
			} else {
				t.Log(err)
			}
		})
	}
}

func TestEVT2h7GeneratedContract(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeSchema10.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := GenerateMachineBridgeCodecs(module)
	if err != nil {
		t.Fatal(err)
	}
	if first.Hash != FallibleMachineBridgeSchemaHash {
		t.Fatal("schema identity drift")
	}
	for path, want := range map[string][]byte{"../../internal/concept/machineir_bridge10_codec_generated.go": first.Go, "../../libraries/Standard/Backend/BridgeCodec10.concept": first.Concept, "../../docs/design/EVT2-MACHINEIR-BRIDGE10.schema.json": first.Metadata} {
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("stale generated %s: %v", path, err)
		}
	}
	for trial := 0; trial < resultSamples(); trial++ {
		next, err := GenerateMachineBridgeCodecs(module)
		if err != nil || !bytes.Equal(next.Go, first.Go) || !bytes.Equal(next.Concept, first.Concept) || !bytes.Equal(next.Metadata, first.Metadata) {
			t.Fatalf("fresh codecs %d: %v", trial, err)
		}
	}
	t.Logf("%d generated codec/schema agreements; hash=%s", resultSamples(), first.Hash)
}

func TestEVT2h7NativeBoundaries(t *testing.T) {
	paths, err := filepath.Glob("../../language/evt2h7/unsupported/*.concept")
	if err != nil || len(paths) != 4 {
		t.Fatal("missing native boundary specimens", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(source), "\n", 2)[0], "// expect: "))
			module, err := Parse(path, string(source))
			if err != nil {
				t.Fatalf("must remain source-valid: %v", err)
			}
			if filepath.Base(path) == "short_circuit.concept" {
				assertEVT2h7Promotion(t, module)
				t.Log("EVT2h8 promotes the historical native and/or boundary")
				return
			}
			if _, err := GenerateMachineIR(module); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want %s, got %v", want, err)
			} else {
				t.Log(err)
			}
		})
	}
}
