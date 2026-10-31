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

func optionBadArtifacts(t *testing.T) []patternBadArtifact {
	t.Helper()
	_, _, _, machine, valid := optionFixture(t)
	blob, err := json.Marshal(machine.Pattern)
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
	add := func(name string, mutate func(*PatternModule)) {
		t.Helper()
		var m PatternModule
		if err := json.Unmarshal(blob, &m); err != nil {
			t.Fatal(err)
		}
		mutate(&m)
		data, err := encodePatternMachineBridge(m)
		if err != nil {
			t.Fatal(err)
		}
		addBytes(name, data)
	}
	instruction := func(m *PatternModule, kind string) *ObjectInstruction {
		for _, op := range m.Base.Operations {
			if op.Kind == kind {
				return &m.Base.Base.Objects.Functions[op.Function].Blocks[op.Block].Instructions[op.Instruction]
			}
		}
		t.Fatalf("missing %s", kind)
		return nil
	}
	binding := func(m *PatternModule) (*PatternFlow, *PatternArm) {
		for pi := range m.Flows {
			p := &m.Flows[pi]
			for ai := range p.Arms {
				if len(p.Arms[ai].Bindings) > 0 {
					return p, &p.Arms[ai]
				}
			}
		}
		t.Fatal("no binding")
		return nil, nil
	}
	add("closed application layout identity", func(m *PatternModule) {
		m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Identity = m.Base.Base.Objects.Types[m.Base.Domains[2].Type].Identity
	})
	add("same-sized application constructor", func(m *PatternModule) { instruction(m, "make_tagged").Type = m.Base.Domains[2].Type })
	add("None cannot authorize Some read", func(m *PatternModule) { instruction(m, "checked_payload").Member = 1 })
	add("Some missing payload", func(m *PatternModule) { instruction(m, "make_tagged").Args = nil })
	add("constructor variant", func(m *PatternModule) { instruction(m, "make_tagged").Member = 999 })
	add("constructor initialization", func(m *PatternModule) { instruction(m, "make_tagged").Op = "copy_value" })
	add("discriminant mismatch", func(m *PatternModule) { m.Base.Domains[0].Variants[0].Tag = 1 })
	add("variant identity", func(m *PatternModule) {
		m.Base.Domains[0].Variants[0].Identity = m.Base.Domains[1].Variants[0].Identity
	})
	add("duplicate variant", func(m *PatternModule) { m.Base.Domains[0].Variants[1] = m.Base.Domains[0].Variants[0] })
	add("payload offset", func(m *PatternModule) { m.Base.Domains[0].PayloadOffset = 1 })
	add("payload geometry", func(m *PatternModule) { m.Base.Domains[0].Variants[0].Fields[0].Offset = 1024 })
	add("payload type", func(m *PatternModule) { m.Base.Domains[0].Variants[0].Fields[0].Type = m.Base.Domains[2].Type })
	add("copy eligibility", func(m *PatternModule) { m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Copy = "none" })
	add("unqualified representation", func(m *PatternModule) { m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Representation = "scalar" })
	add("wrong closed return", func(m *PatternModule) { m.Base.Base.Objects.Functions[0].ResultType = m.Base.Domains[2].Type })
	add("wrong closed argument", func(m *PatternModule) {
		for fi := range m.Base.Base.Objects.Functions {
			f := &m.Base.Base.Objects.Functions[fi]
			if f.Name == "IntValue" {
				f.Slots[0].Type = m.Base.Domains[2].Type
				return
			}
		}
		t.Fatal("missing getter")
	})
	add("missing active authority", func(m *PatternModule) { m.Flows = nil })
	add("missing payload correspondence", func(m *PatternModule) {
		for i, op := range m.Base.Operations {
			if op.Kind == "checked_payload" {
				m.Base.Operations = append(m.Base.Operations[:i], m.Base.Operations[i+1:]...)
				return
			}
		}
	})
	add("payload outside binding scope", func(m *PatternModule) {
		p, a := binding(m)
		m.Base.Base.Authority[p.Function].SlotScopes[a.Bindings[0].Slot] = p.Scope
	})
	add("missing initialized binding", func(m *PatternModule) { _, a := binding(m); a.Bindings = nil })
	add("invalid dispatch successor", func(m *PatternModule) {
		p, _ := binding(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.True = 999
	})
	add("bypass active selection", func(m *PatternModule) {
		p, _ := binding(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.False = p.Arms[0].Body
	})
	add("wrong result join", func(m *PatternModule) { p, _ := binding(m); p.ResultSlot = p.SubjectSlot })
	add("source provenance", func(m *PatternModule) {
		p, _ := binding(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Instructions[1].Source.Line++
	})
	add("capacity", func(m *PatternModule) { m.Base.Domains = make([]TaggedDomain, 129) })
	for _, offset := range []int{8, 12} {
		data := bytes.Clone(valid)
		data[offset] ^= 255
		addBytes(fmt.Sprintf("header %d", offset), data)
	}
	addBytes("truncated", valid[:len(valid)-1])
	addBytes("trailing", append(bytes.Clone(valid), 0))
	return cases
}

func TestEVT2h4MalformedContracts(t *testing.T) {
	t.Logf("%d malformed closed Option contracts rejected", len(optionBadArtifacts(t)))
}

func TestEVT2h4ArtifactAdmission(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	cases := optionBadArtifacts(t)
	dir := t.TempDir()
	var records strings.Builder
	for i, tc := range cases {
		path := filepath.Join(dir, fmt.Sprintf("%02d.cmir9", i))
		if err := os.WriteFile(path, tc.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&records, "{%q,%q},\n", filepath.ToSlash(path), tc.name)
	}
	host := strings.Replace(patternMalformedHost, "/* CASES */", records.String(), 1)
	_, _, _, _, valid := optionFixture(t)
	path := filepath.Join(dir, "valid.cmir9")
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	roundtrip := fmt.Sprintf(`FILE*vf=fopen(%q,"rb");if(!vf)return 10;size_t vn=fread(input,1,sizeof input,vf);fclose(vf);for(int run=0;run<100;run++){concept_readonly_span_byte src={input,vn};concept_result_int_pattern_error r=F(validated_pattern_bridge_round_trip)(src,(concept_span_byte){output,sizeof output});if(r.tag||r.payload.ok.value!=(int)vn||memcmp(input,output,vn))return 11;}puts("100 valid exact Option artifact-only round trips");`, filepath.ToSlash(path))
	host = strings.Replace(host, "int main(void){", "int main(void){"+roundtrip, 1)
	// Logical transport admission is separate from physical Win64 capacity.
	// Eight explicit parameters plus an sret pointer exceed the frozen bound.
	boundarySource := "../../language/evt2h4/native_boundary/abi_capacity.concept"
	source, err := os.ReadFile(boundarySource)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(boundarySource, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(module)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	boundaryPath := filepath.Join(dir, "capacity.cmir8")
	if err := os.WriteFile(boundaryPath, boundary, 0600); err != nil {
		t.Fatal(err)
	}
	preflight := fmt.Sprintf(`{FILE*bf=fopen(%q,"rb");if(!bf)return 20;size_t bn=fread(input,1,sizeof input,bf);fclose(bf);memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_result_int_native_error e=F(emit_module)((concept_readonly_span_byte){input,bn},(concept_span_byte){output,sizeof output},&image);if(!e.tag||memcmp(&image,&before,sizeof image))return 21;for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 22;printf("Option physical ABI boundary: %%s; sentinels preserved\n",F(describe_native_error)(e.payload.error.error));}`, filepath.ToSlash(boundaryPath))
	host = strings.Replace(host, "int main(void){", "int main(void){"+preflight, 1)
	for _, verify := range []bool{false, true} {
		out := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "option_admission.c", host, "-pedantic-errors", "-O2")
		if !strings.Contains(out, "Option physical ABI boundary: AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED") {
			t.Fatal(out)
		}
		t.Log(out)
	}
}
