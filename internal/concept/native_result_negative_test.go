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

func resultBadArtifacts(t *testing.T) []patternBadArtifact {
	t.Helper()
	_, _, _, machine, valid := resultFixture(t)
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
		raw, err := encodePatternMachineBridge(m)
		if err != nil {
			t.Fatal(err)
		}
		addBytes(name, raw)
	}
	flow := func(m *PatternModule, name string) *PatternFlow {
		for i := range m.Flows {
			p := &m.Flows[i]
			if m.Base.Base.Objects.Functions[p.Function].Name == name {
				return p
			}
		}
		t.Fatal("missing flow " + name)
		return nil
	}
	instruction := func(m *PatternModule, kind string) *ObjectInstruction {
		for _, op := range m.Base.Operations {
			if op.Kind == kind {
				return &m.Base.Base.Objects.Functions[op.Function].Blocks[op.Block].Instructions[op.Instruction]
			}
		}
		t.Fatal("missing " + kind)
		return nil
	}
	add("closed Result identity", func(m *PatternModule) {
		m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Identity = m.Base.Base.Objects.Types[m.Base.Domains[1].Type].Identity
	})
	add("swapped success and error types", func(m *PatternModule) {
		d := &m.Base.Domains[0]
		d.Variants[0].Fields[0].Type = d.Variants[1].Fields[0].Type
	})
	add("constructor arity", func(m *PatternModule) { instruction(m, "make_tagged").Args = nil })
	add("constructor discriminant", func(m *PatternModule) { instruction(m, "make_tagged").Member = 77 })
	add("inactive Error read in Ok arm", func(m *PatternModule) {
		p := flow(m, "Double")
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[p.Arms[0].Body].Instructions[1].Member = 1
	})
	add("missing variant authority", func(m *PatternModule) { m.Flows = nil })
	add("missing operation correspondence", func(m *PatternModule) { m.Operations = nil })
	add("propagation Error bypasses return", func(m *PatternModule) {
		p := flow(m, "Double")
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[p.Arms[1].Body].Term.Op = "jump"
		f.Blocks[p.Arms[1].Body].Term.True = p.Join
		f.Blocks[p.Arms[1].Body].Term.Value = -1
	})
	add("propagation success jumps into Error", func(m *PatternModule) {
		p := flow(m, "Double")
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Body].Term.True = p.Arms[1].Body
	})
	add("uninitialized propagation result", func(m *PatternModule) {
		p := flow(m, "Double")
		f := &m.Base.Base.Objects.Functions[p.Function]
		for i := range f.Blocks[p.Arms[0].Body].Instructions {
			in := &f.Blocks[p.Arms[0].Body].Instructions[i]
			if in.Op == "initialize_slot" && in.Slot == p.ResultSlot {
				in.Op = "load_slot"
				in.Result = 511
				in.Args = nil
				return
			}
		}
		t.Fatal("no initialized result")
	})
	add("nondominating propagation binding", func(m *PatternModule) {
		p := flow(m, "Double")
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[p.Arms[0].Body].Instructions[1].Args[0].Value = f.Blocks[p.Arms[1].Body].Instructions[0].Result
	})
	add("wrong propagation return application", func(m *PatternModule) {
		p := flow(m, "Double")
		m.Base.Base.Objects.Functions[p.Function].ResultType = m.Base.Domains[1].Type
	})
	add("void Ok fabricated payload", func(m *PatternModule) {
		p := flow(m, "VoidCode")
		typ := m.Base.Base.Objects.Functions[p.Function].Slots[p.SubjectSlot].Type
		for i := range m.Base.Domains {
			d := &m.Base.Domains[i]
			if d.Type == typ {
				d.Variants[0].Fields = append(d.Variants[0].Fields, d.Variants[1].Fields[0])
				return
			}
		}
		t.Fatal("no void domain")
	})
	add("void Ok payload binding", func(m *PatternModule) {
		p := flow(m, "VoidCode")
		p.Arms[0].Bindings = append(p.Arms[0].Bindings, PatternBinding{Slot: p.SubjectSlot})
	})
	add("escaped propagation binding", func(m *PatternModule) {
		p := flow(m, "Double")
		m.Base.Base.Authority[p.Function].SlotScopes[p.Arms[0].Bindings[0].Slot] = p.Scope
	})
	add("swapped nominal error enum", func(m *PatternModule) { m.Enums[0].Identity = m.Enums[1].Identity })
	add("invalid error enum discriminant", func(m *PatternModule) { m.Enums[0].Variants[0].Tag = 99 })
	add("copy eligibility", func(m *PatternModule) { m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Copy = "none" })
	add("payload geometry", func(m *PatternModule) { m.Base.Domains[0].PayloadOffset = 1 })
	add("variant identity", func(m *PatternModule) {
		m.Base.Domains[0].Variants[1].Identity = m.Base.Domains[1].Variants[1].Identity
	})
	add("domain capacity", func(m *PatternModule) { m.Base.Domains = make([]TaggedDomain, 129) })
	add("CFG successor", func(m *PatternModule) {
		p := flow(m, "Double")
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.True = 999
	})
	add("guard type", func(m *PatternModule) {
		p := flow(m, "Double")
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[p.Arms[0].Dispatch].Term.Value = f.Blocks[p.Arms[0].Dispatch].Instructions[0].Result
	})
	for _, offset := range []int{8, 12} {
		raw := bytes.Clone(valid)
		raw[offset] ^= 255
		addBytes(fmt.Sprintf("header %d", offset), raw)
	}
	addBytes("truncated", valid[:len(valid)-1])
	addBytes("trailing", append(bytes.Clone(valid), 0))
	return cases
}

func TestEVT2h5MalformedContracts(t *testing.T) {
	t.Logf("%d malformed Result/propagation contracts rejected", len(resultBadArtifacts(t)))
}

func TestEVT2h5ArtifactAdmission(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	cases := resultBadArtifacts(t)
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
	_, _, _, _, valid := resultFixture(t)
	path := filepath.Join(dir, "valid.cmir9")
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	roundtrip := fmt.Sprintf(`FILE*vf=fopen(%q,"rb");if(!vf)return 10;size_t vn=fread(input,1,sizeof input,vf);fclose(vf);for(int run=0;run<%d;run++){concept_readonly_span_byte src={input,vn};concept_result_int_pattern_error r=F(validated_pattern_bridge_round_trip)(src,(concept_span_byte){output,sizeof output});if(r.tag||r.payload.ok.value!=(int)vn||memcmp(input,output,vn))return 11;}puts("exact Result artifact-only round trips passed");`, filepath.ToSlash(path), resultSamples())
	host = strings.Replace(host, "int main(void){", "int main(void){"+roundtrip, 1)
	var preflight strings.Builder
	for i, name := range []string{"valid/constructor_only.concept", "valid/scalar_constructor_only.concept", "native_boundary/abi_capacity.concept"} {
		data := resultStandalone(t, name)
		path := filepath.Join(dir, fmt.Sprintf("standalone-%d.cmir", i))
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&preflight, `{FILE*f=fopen(%q,"rb");if(!f)return 20;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte src={input,n};`, filepath.ToSlash(path))
		if i == 2 {
			preflight.WriteString(`memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_result_int_native_error e=F(emit_module)(src,(concept_span_byte){output,sizeof output},&image);if(!e.tag||strcmp(F(describe_native_error)(e.payload.error.error),"AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED")||memcmp(&image,&before,sizeof image))return 21;for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 22;puts("Result ABI capacity rejected with code/image sentinels");}`)
		} else {
			preflight.WriteString(`concept_result_int_native_error e=F(emit_module)(src,(concept_span_byte){output,sizeof output},&image);if(e.tag||image.symbolCount!=1)return 23;unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 24;memcpy(memory,output,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 25;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);typedef struct{uint32_t tag;int value;}R;typedef R(*Make)(_Bool,int);Make make;void*p=memory+image.symbols.data[0].offset;memcpy(&make,&p,sizeof make);R a=make(1,42),b=make(0,42);if(a.tag||a.value!=42||b.tag!=1||!VirtualFree(memory,0,MEM_RELEASE))return 26;puts("standalone constructor-only native Result executed");}`)
		}
	}
	host = strings.Replace(host, "#include <stdio.h>", "#include <stdio.h>\n#include <stdint.h>\n#include <windows.h>", 1)
	host = strings.Replace(host, "int main(void){", "int main(void){"+preflight.String(), 1)
	for _, verify := range []bool{false, true} {
		t.Log(runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "result_admission.c", host, "-pedantic-errors", "-O2"))
	}
}
