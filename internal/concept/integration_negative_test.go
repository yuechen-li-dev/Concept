package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func integrationBadArtifacts(t *testing.T) []patternBadArtifact {
	t.Helper()
	s := integrationSpecimens(t)[1]
	raw, _ := json.Marshal(s.machine.Fallible)
	var cases []patternBadArtifact
	addBytes := func(name string, data []byte) {
		t.Helper()
		if _, err := DecodeMachineBridge(data); err == nil {
			t.Fatalf("Go admitted %s", name)
		}
		cases = append(cases, patternBadArtifact{name, data})
	}
	add := func(name string, mutate func(*FallibleModule)) {
		t.Helper()
		var m FallibleModule
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		mutate(&m)
		data, err := encodeFallibleMachineBridge(m)
		if err != nil {
			t.Fatal(err)
		}
		addBytes(name, data)
	}
	owner := func(m *FallibleModule, name string) *ObjectFunction {
		for i := range m.Base.Base.Base.Objects.Functions {
			f := &m.Base.Base.Base.Objects.Functions[i]
			if f.Name == name {
				return f
			}
		}
		t.Fatal(name)
		return nil
	}
	flow := func(m *FallibleModule, name string) *PatternFlow {
		for i := range m.Base.Flows {
			p := &m.Base.Flows[i]
			if m.Base.Base.Base.Objects.Functions[p.Function].Name == name {
				return p
			}
		}
		t.Fatal(name)
		return nil
	}
	add("inactive success payload", func(m *FallibleModule) {
		p := flow(m, "GateAnd")
		owner(m, "GateAnd").Blocks[p.Arms[0].Body].Instructions[1].Member = 1
	})
	add("unknown logical successor", func(m *FallibleModule) { owner(m, "GateAnd").Blocks[0].Term.True = 999 })
	add("nondominating logical input", func(m *FallibleModule) { owner(m, "GateAnd").Blocks[0].Term.Value = 9999 })
	add("uninitialized logical result", func(m *FallibleModule) {
		f := owner(m, "GateAnd")
		for bi := range f.Blocks {
			for ii, in := range f.Blocks[bi].Instructions {
				if (in.Op == "store_slot" || in.Op == "initialize_slot") && strings.HasPrefix(f.Slots[in.Slot].Name, "$logical") {
					f.Blocks[bi].Instructions = append(f.Blocks[bi].Instructions[:ii], f.Blocks[bi].Instructions[ii+1:]...)
					return
				}
			}
		}
		t.Fatal("logical slot")
	})
	add("forged handler destination", func(m *FallibleModule) { m.Handlers[0].Entry = 0 })
	add("forged handler nominal type", func(m *FallibleModule) { m.Handlers[0].Type = m.Handlers[len(m.Handlers)-1].Type })
	add("invalid handler join", func(m *FallibleModule) { m.Tries[0].Join = 999 })
	add("missing failure correspondence", func(m *FallibleModule) { m.Sites = nil })
	add("missing pattern correspondence", func(m *FallibleModule) { m.Base.Operations = nil })
	add("missing tagged correspondence", func(m *FallibleModule) { m.Base.Base.Operations = nil })
	add("wrong closed sum identity", func(m *FallibleModule) {
		m.Base.Base.Base.Objects.Types[m.Base.Base.Domains[0].Type].Identity = "forged"
	})
	add("ineligible sum copy", func(m *FallibleModule) { m.Base.Base.Base.Objects.Types[m.Base.Base.Domains[0].Type].Copy = "none" })
	add("payload geometry", func(m *FallibleModule) { m.Base.Base.Domains[0].Variants[0].Fields[0].Offset = 999 })
	add("enum field geometry", func(m *FallibleModule) {
		for i := range m.Base.Base.Base.Objects.Types {
			p := &m.Base.Base.Base.Objects.Types[i]
			if p.Name == "Session" {
				p.Fields[0].Size = 8
				return
			}
		}
		t.Fatal("Session")
	})
	add("frame geometry", func(m *FallibleModule) { m.Base.Base.Base.Objects.Types[owner(m, "Run").Slots[0].Type].Align = 3 })
	add("flow capacity", func(m *FallibleModule) {
		for len(m.Sites) < 8193 {
			m.Sites = append(m.Sites, m.Sites[0])
		}
	})
	for _, n := range []int{0, 7, 43, len(s.data) - 1} {
		addBytes(fmt.Sprintf("truncated %d", n), s.data[:n])
	}
	for _, index := range []int{0, 7, 8, 12} {
		data := bytes.Clone(s.data)
		data[index] ^= 1
		addBytes(fmt.Sprintf("header byte %d", index), data)
	}
	addBytes("trailing data", append(bytes.Clone(s.data), 0))
	return cases
}

func TestEVT2h8MalformedIntegration(t *testing.T) {
	t.Logf("%d malformed integration artifacts rejected by Go", len(integrationBadArtifacts(t)))
}

func TestEVT2h8SourceDiagnostics(t *testing.T) {
	paths, err := filepath.Glob("../../language/evt2h8/invalid/*.concept")
	if err != nil || len(paths) != 3 {
		t.Fatal("missing integration diagnostics", err)
	}
	want := map[string]string{"closed_option_swap.concept": "CV4116", "logical_integer.concept": "CV4028", "nominal_handler.concept": "CV4545"}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Parse(path, string(source)); err == nil {
				t.Fatal("accepted invalid integration source")
			} else {
				if !strings.Contains(err.Error(), want[filepath.Base(path)]) {
					t.Fatal(err)
				}
				t.Log(err)
			}
		})
	}
}

func TestEVT2h8ArtifactAdmission(t *testing.T) {
	requireNativeQualification(t)
	dir := t.TempDir()
	var records strings.Builder
	for i, tc := range integrationBadArtifacts(t) {
		path := filepath.Join(dir, fmt.Sprintf("%02d.cmir10", i))
		if err := os.WriteFile(path, tc.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&records, "{%q,%q},\n", filepath.ToSlash(path), tc.name)
	}
	host := strings.Replace(patternMalformedHost, "/* CASES */", records.String(), 1)
	host = strings.NewReplacer("pattern_error", "fallible_error", "validate_pattern_machine_bridge", "validate_fallible_machine_bridge", "validated_pattern_bridge_round_trip", "validated_fallible_bridge_round_trip", "describe_pattern_error", "describe_fallible_error").Replace(host)
	for _, verify := range []bool{false, true} {
		t.Log(runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "integration_malformed.c", host, "-pedantic-errors", "-O2"))
	}
}
