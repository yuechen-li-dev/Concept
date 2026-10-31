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

const resultFixturePath = "../../tests/runtime/Testing/Results.concept"

func resultFixture(t *testing.T) (Module, []byte, LIRModule, MachineModule, []byte) {
	t.Helper()
	source, err := os.ReadFile(resultFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(resultFixturePath, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	return module, source, lir, machine, data
}
func TestEVT2h5CheckedResult(t *testing.T) {
	_, _, lir, m, data := resultFixture(t)
	if m.Pattern == nil {
		t.Fatal("missing checked CFG")
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, d := range lir.TaggedEnums {
		var app GenericApplication
		if err := json.Unmarshal([]byte(d.Identity), &app); err != nil || app.Declaration.Name != evt1ResultName || len(app.Arguments) != 2 || seen[d.Identity] {
			t.Fatalf("closed identity %s: %v", d.Identity, err)
		}
		seen[d.Identity] = true
		if len(d.Variants) != 2 || d.Variants[0].Name != "Ok" || d.Variants[0].Tag != 0 || d.Variants[1].Name != "Error" || d.Variants[1].Tag != 1 {
			t.Fatal("lost canonical variants")
		}
		if app.Arguments[0].Type.Name == "void" && len(d.Variants[0].Fields) != 0 {
			t.Fatal("fabricated void success payload")
		}
		t.Logf("%s size=%d align=%d", d.Name, d.Size, d.Alignment)
	}
	t.Logf("%d Result applications, %d enums; artifact=%d", len(lir.TaggedEnums), len(lir.Enums), len(data))
}

func TestEVT2h5Determinism(t *testing.T) {
	start := time.Now()
	_, _, lir, machine, data := resultFixture(t)
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "df95976a73a53afc06d280ebc13b31d00aeb47086aa6dd9274947bb938fdad2d" {
		t.Fatalf("exact Result artifact changed: %s", got)
	}
	for trial := 0; trial < resultSamples(); trial++ {
		_, _, next, other, raw := resultFixture(t)
		if !bytes.Equal(raw, data) || next.String() != lir.String() || other.String() != machine.String() {
			t.Fatalf("fresh trial %d differs", trial)
		}
		decoded, err := DecodeMachineBridge(raw)
		if err != nil || decoded.String() != machine.String() {
			t.Fatalf("roundtrip %d: %v", trial, err)
		}
	}
	t.Logf("%d fresh source/LIR/CFG/MachineIR/debug/codec generations in %s; artifact=%d sha256=%x", resultSamples(), time.Since(start), len(data), sha256.Sum256(data))
}

func TestEVT2h5SourceBoundaries(t *testing.T) {
	for _, group := range []string{"invalid", "unsupported"} {
		paths, err := filepath.Glob("../../language/evt2h5/" + group + "/*.concept")
		if err != nil || len(paths) == 0 {
			t.Fatalf("specimens %s: %v", group, err)
		}
		for _, path := range paths {
			t.Run(group+"/"+filepath.Base(path), func(t *testing.T) {
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				code := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(source), "\n", 2)[0], "// expect: "))
				module, err := Parse(path, string(source))
				if group == "unsupported" {
					if err != nil {
						t.Fatalf("must remain source-valid: %v", err)
					}
					_, err = GenerateMachineIR(module)
				}
				if err == nil || code == "" || !strings.Contains(err.Error(), code) {
					t.Fatalf("want %s, got %v", code, err)
				}
				t.Log(err)
			})
		}
	}
}

func resultStandalone(t *testing.T, name string) []byte {
	t.Helper()
	path := "../../language/evt2h5/" + name
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(module)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	return data
}
func TestEVT2h5ConstructorContracts(t *testing.T) {
	for _, name := range []string{"valid/constructor_only.concept", "valid/scalar_constructor_only.concept"} {
		data := resultStandalone(t, name)
		version := "CMIRAMD9"
		if strings.Contains(name, "scalar") {
			version = "CMIRAMD8"
		}
		if !bytes.HasPrefix(data, []byte(version)) {
			t.Fatalf("%s must use %s", name, version)
		}
	}
}

func TestEVT2h5LIRAdmission(t *testing.T) {
	_, _, lir, _, _ := resultFixture(t)
	blob, err := json.Marshal(lir)
	if err != nil {
		t.Fatal(err)
	}
	owner := func(m *LIRModule) *LIRFunction {
		for i := range m.Functions {
			if m.Functions[i].Name == "Double" {
				return &m.Functions[i]
			}
		}
		t.Fatal("missing propagation")
		return nil
	}
	for name, mutate := range map[string]func(*LIRModule){
		"closed identity": func(m *LIRModule) { m.TaggedEnums[0].Identity = m.TaggedEnums[1].Identity },
		"inactive payload": func(m *LIRModule) {
			f := owner(m)
			p := f.Patterns[0]
			f.Blocks[p.Arms[0].Body].Instructions[1].Member = 1
		},
		"error skips return": func(m *LIRModule) {
			f := owner(m)
			p := f.Patterns[0]
			f.Blocks[p.Arms[1].Body].Term.Op = "jump"
			f.Blocks[p.Arms[1].Body].Term.True = p.Join
		},
		"uninitialized join": func(m *LIRModule) {
			f := owner(m)
			p := f.Patterns[0]
			body := &f.Blocks[p.Arms[0].Body]
			for i, in := range body.Instructions {
				if in.Slot == p.ResultSlot && in.Op == "initialize_slot" {
					body.Instructions = append(body.Instructions[:i], body.Instructions[i+1:]...)
					return
				}
			}
			t.Fatal("missing initialization")
		},
		"wrong return type": func(m *LIRModule) { owner(m).Result = m.TaggedEnums[1].Type },
		"missing authority": func(m *LIRModule) { owner(m).Patterns = nil },
		"copy permission":   func(m *LIRModule) { m.TaggedEnums[0].CopyEligibility = "none" },
	} {
		t.Run(name, func(t *testing.T) {
			var m LIRModule
			if err := json.Unmarshal(blob, &m); err != nil {
				t.Fatal(err)
			}
			mutate(&m)
			if err := VerifyLIR(m); err == nil {
				t.Fatal("accepted malformed Result LIR")
			} else {
				t.Log(err)
			}
		})
	}
}
