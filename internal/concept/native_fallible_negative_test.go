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

func fallibleBadArtifacts(t *testing.T) []patternBadArtifact {
	t.Helper()
	_, _, _, machine, valid := fallibleFixture(t)
	blob, err := json.Marshal(machine.Fallible)
	if err != nil {
		t.Fatal(err)
	}
	var cases []patternBadArtifact
	addBytes := func(name string, data []byte) {
		t.Helper()
		if _, err := DecodeMachineBridge(data); err == nil {
			t.Fatalf("Go accepted %s", name)
		}
		cases = append(cases, patternBadArtifact{name, data})
	}
	add := func(name string, mutate func(*FallibleModule)) {
		t.Helper()
		var m FallibleModule
		if err := json.Unmarshal(blob, &m); err != nil {
			t.Fatal(err)
		}
		mutate(&m)
		data, err := encodeFallibleMachineBridge(m)
		if err != nil {
			t.Fatal(err)
		}
		addBytes(name, data)
	}
	site := func(m *FallibleModule, name string) *FallibleSite {
		for i := range m.Sites {
			p := m.Base.Flows[m.Sites[i].Flow]
			if m.Base.Base.Base.Objects.Functions[p.Function].Name == name {
				return &m.Sites[i]
			}
		}
		t.Fatal("missing site", name)
		return nil
	}
	flow := func(m *FallibleModule, name string) *PatternFlow { return &m.Base.Flows[site(m, name).Flow] }
	handler := func(m *FallibleModule, name string) *FallibleHandler { return &m.Handlers[site(m, name).Handler] }
	body := func(m *FallibleModule, name string, arm int) *ObjectBlock {
		p := flow(m, name)
		return &m.Base.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[arm].Body]
	}
	add("guard missing", func(m *FallibleModule) { m.Sites = m.Sites[1:] })
	add("guard duplicate", func(m *FallibleModule) { m.Sites = append(m.Sites, m.Sites[0]) })
	add("guard wrong variant", func(m *FallibleModule) {
		p := flow(m, "RequireResult")
		m.Base.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Instructions[1].Member = 1
	})
	add("guard inactive payload", func(m *FallibleModule) { body(m, "RequireResult", 0).Instructions[1].Member = 1 })
	add("guard wrong kind", func(m *FallibleModule) { site(m, "RequireResult").Kind = "propagate" })
	add("forged handler identity", func(m *FallibleModule) { site(m, "Local").Handler = 99 })
	add("wrong nominal handler", func(m *FallibleModule) { s := site(m, "Local"); s.Handler++ })
	add("wrong nearest handler", func(m *FallibleModule) { s := site(m, "Nearest"); s.Handler++ })
	add("handler skips initialization", func(m *FallibleModule) {
		p := flow(m, "Local")
		body(m, "Local", 1).Term.True = m.Tries[handler(m, "Local").Try].Join
		_ = p
	})
	add("handler entry forged", func(m *FallibleModule) { handler(m, "Local").Entry = 0 })
	add("binding slot forged", func(m *FallibleModule) { h := handler(m, "Local"); h.BindingSlot = h.InputSlot })
	add("scope forged", func(m *FallibleModule) { handler(m, "Local").Scope = 1 })
	add("try identity forged", func(m *FallibleModule) { handler(m, "Local").Try = 999 })
	add("invalid try join", func(m *FallibleModule) { m.Tries[0].Join = 999 })
	add("missing error transfer", func(m *FallibleModule) { b := body(m, "Local", 1); b.Instructions = b.Instructions[:4] })
	add("uninitialized success", func(m *FallibleModule) {
		b := body(m, "RequireResult", 0)
		b.Instructions = b.Instructions[:len(b.Instructions)-2]
	})
	add("void binding forged", func(m *FallibleModule) { flow(m, "RequireVoid").Arms[0].Bindings = []PatternBinding{{Slot: 0}} })
	add("invalid propagation target", func(m *FallibleModule) {
		b := body(m, "ForwardOption", 1)
		b.Instructions[len(b.Instructions)-1].Member = 0
	})
	add("missing pattern authority", func(m *FallibleModule) { m.Base.Operations = nil })
	add("missing tagged authority", func(m *FallibleModule) { m.Base.Base.Operations = nil })
	add("wrong nominal sum identity", func(m *FallibleModule) {
		m.Base.Base.Base.Objects.Types[m.Base.Base.Domains[0].Type].Identity = "forged"
	})
	add("copy eligibility", func(m *FallibleModule) { m.Base.Base.Base.Objects.Types[m.Base.Base.Domains[0].Type].Copy = "none" })
	add("payload geometry", func(m *FallibleModule) { m.Base.Base.Domains[0].Variants[0].Fields[0].Offset = 999 })
	add("stale dependency", func(m *FallibleModule) {
		m.Base.Base.Base.Objects.Dependencies = append(m.Base.Base.Base.Objects.Dependencies, ObjectDependency{Identity: "forged", Digest: "bad"})
	})
	add("site capacity", func(m *FallibleModule) {
		for len(m.Sites) < 8193 {
			m.Sites = append(m.Sites, m.Sites[0])
		}
	})
	for _, n := range []int{0, 7, 8, 11, 43, len(valid) - 1} {
		addBytes(fmt.Sprintf("truncated %d", n), valid[:n])
	}
	raw := bytes.Clone(valid)
	raw[0] ^= 1
	addBytes("magic prefix", raw)
	raw = bytes.Clone(valid)
	raw[7] = 'B'
	addBytes("unknown dispatch version", raw)
	raw = bytes.Clone(valid)
	raw[8]++
	addBytes("version", raw)
	raw = bytes.Clone(valid)
	raw[12] ^= 1
	addBytes("schema hash", raw)
	addBytes("trailing bytes", append(bytes.Clone(valid), 0))
	return cases
}

func TestEVT2h7MalformedContracts(t *testing.T) {
	t.Logf("%d malformed contracts reject independently", len(fallibleBadArtifacts(t)))
}

func TestEVT2h7ArtifactAdmission(t *testing.T) {
	requireNativeQualification(t)
	cases := fallibleBadArtifacts(t)
	dir := t.TempDir()
	var records strings.Builder
	for i, tc := range cases {
		path := filepath.Join(dir, fmt.Sprintf("%02d.cmir10", i))
		if err := os.WriteFile(path, tc.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&records, "{%q,%q},\n", filepath.ToSlash(path), tc.name)
	}
	host := strings.Replace(patternMalformedHost, "/* CASES */", records.String(), 1)
	host = strings.NewReplacer("pattern_error", "fallible_error", "validate_pattern_machine_bridge", "validate_fallible_machine_bridge", "validated_pattern_bridge_round_trip", "validated_fallible_bridge_round_trip", "describe_pattern_error", "describe_fallible_error").Replace(host)
	for _, verify := range []bool{false, true} {
		t.Log(runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "fallible_malformed.c", host, "-pedantic-errors", "-O2"))
	}
}
