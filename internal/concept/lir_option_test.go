package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const optionFixturePath = "../../tests/runtime/Testing/Options.concept"

func optionFixture(t *testing.T) (Module, []byte, LIRModule, MachineModule, []byte) {
	t.Helper()
	source, err := os.ReadFile(optionFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(optionFixturePath, string(source))
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
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	return module, source, lir, machine, artifact
}
func TestEVT2h4CheckedOption(t *testing.T) {
	_, _, lir, machine, data := optionFixture(t)
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "410976620e998aac75eb2e79e1cc0949c01cb05f595516bb3f4fc6b7c1794f88" {
		t.Fatalf("frozen Option artifact changed: %s", got)
	}
	if len(lir.TaggedEnums) != 8 || machine.Pattern == nil {
		t.Fatal("missing closed options")
	}
	seen := map[string]bool{}
	for _, e := range lir.TaggedEnums {
		if seen[e.Identity] {
			t.Fatal("collapsed closed identity")
		}
		seen[e.Identity] = true
		var application GenericApplication
		if err := json.Unmarshal([]byte(e.Identity), &application); err != nil || application.Declaration.Name != evt1OptionName || len(application.Arguments) != 1 || application.Arguments[0].Kind != "type" {
			t.Fatalf("lost canonical closed application: %s (%v)", e.Identity, err)
		}
		if len(e.Variants) != 2 || e.Variants[0].Name != "Some" || e.Variants[0].Tag != 0 || len(e.Variants[0].Fields) != 1 || e.Variants[1].Name != "None" || e.Variants[1].Tag != 1 || len(e.Variants[1].Fields) != 0 {
			t.Fatalf("changed intrinsic Option contract: %+v", e)
		}
		t.Logf("%s %s size=%d align=%d", e.Name, e.Identity, e.Size, e.Alignment)
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
}

func TestEVT2h4Transport(t *testing.T) {
	start := time.Now()
	_, _, lir, machine, data := optionFixture(t)
	for run := 0; run < 100; run++ {
		_, _, other, next, raw := optionFixture(t)
		if !bytes.Equal(raw, data) || other.String() != lir.String() || next.String() != machine.String() {
			t.Fatalf("fresh generation %d differs", run)
		}
		decoded, err := DecodeMachineBridge(raw)
		if err != nil || decoded.String() != machine.String() {
			t.Fatalf("roundtrip %d: %v", run, err)
		}
	}
	t.Logf("100 fresh Option source/LIR/CFG/MachineIR/debug/codecs: %s; artifact=%d sha256=%x", time.Since(start), len(data), sha256.Sum256(data))
	// A constructor-only application uses the already-frozen tagged contract;
	// matches add the already-frozen pattern contract, without artificial flows.
	m, err := Parse("option_constructor.concept", "module OptionConstructor; profile Core; Option<int> Make() { return Option::Some(42); }")
	if err != nil {
		t.Fatal(err)
	}
	native, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeMachineBridge(native)
	if err != nil || !bytes.HasPrefix(raw, []byte("CMIRAMD8")) {
		t.Fatalf("constructor bridge: %v", err)
	}
}

func TestEVT2h4SourceBoundaries(t *testing.T) {
	groups := []struct {
		directory string
		native    bool
	}{{"invalid", false}, {"unsupported", true}}
	for _, group := range groups {
		entries, err := os.ReadDir("../../language/evt2h4/" + group.directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".concept") {
				continue
			}
			t.Run(group.directory+"/"+entry.Name(), func(t *testing.T) {
				path := "../../language/evt2h4/" + group.directory + "/" + entry.Name()
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				line := strings.SplitN(string(source), "\n", 2)[0]
				code := strings.TrimSpace(strings.TrimPrefix(line, "// expect: "))
				m, err := Parse(path, string(source))
				if group.native {
					if err != nil {
						t.Fatalf("must remain source-valid: %v", err)
					}
					_, err = GenerateMachineIR(m)
				}
				if err == nil || code == "" || !strings.Contains(err.Error(), code) {
					t.Fatalf("want %s, got %v", code, err)
				}
				t.Log(err)
			})
		}
	}
}

func TestEVT2h4LIRAdmission(t *testing.T) {
	_, _, lir, _, _ := optionFixture(t)
	blob, err := json.Marshal(lir)
	if err != nil {
		t.Fatal(err)
	}
	read := func(m *LIRModule) *LIRInstruction {
		for fi := range m.Functions {
			for bi := range m.Functions[fi].Blocks {
				for ii := range m.Functions[fi].Blocks[bi].Instructions {
					in := &m.Functions[fi].Blocks[bi].Instructions[ii]
					if in.Op == "checked_payload" {
						return in
					}
				}
			}
		}
		t.Fatal("no payload")
		return nil
	}
	for name, mutate := range map[string]func(*LIRModule){
		"closed identity":  func(m *LIRModule) { m.TaggedEnums[0].Identity = m.TaggedEnums[2].Identity },
		"geometry":         func(m *LIRModule) { m.TaggedEnums[0].PayloadOffset = 1 },
		"inactive None":    func(m *LIRModule) { read(m).Member = 1 },
		"wrong field type": func(m *LIRModule) { read(m).Type = m.TaggedEnums[0].Type },
		"missing selection": func(m *LIRModule) {
			for i := range m.Functions {
				m.Functions[i].Patterns = nil
			}
		},
		"copy permission": func(m *LIRModule) { m.TaggedEnums[0].CopyEligibility = "none" },
	} {
		t.Run(name, func(t *testing.T) {
			var m LIRModule
			if err := json.Unmarshal(blob, &m); err != nil {
				t.Fatal(err)
			}
			mutate(&m)
			if err := VerifyLIR(m); err == nil {
				t.Fatal("accepted malformed checked Option LIR")
			} else {
				t.Log(err)
			}
		})
	}
}
