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

type patternBadArtifact struct {
	name string
	data []byte
}

func patternBadArtifacts(t *testing.T) []patternBadArtifact {
	t.Helper()
	_, _, _, machine, valid := patternFixture(t, false)
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
	flow := func(m *PatternModule) *PatternFlow {
		for i := range m.Flows {
			for _, a := range m.Flows[i].Arms {
				if len(a.Bindings) > 0 {
					return &m.Flows[i]
				}
			}
		}
		t.Fatal("no binding flow")
		return nil
	}
	binding := func(m *PatternModule) (*PatternFlow, *PatternArm) {
		p := flow(m)
		for i := range p.Arms {
			if len(p.Arms[i].Bindings) > 0 {
				return p, &p.Arms[i]
			}
		}
		panic("binding")
	}
	dispatch := func(m *PatternModule) *ObjectInstruction {
		p := flow(m)
		return &m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Instructions[1]
	}
	read := func(m *PatternModule) *ObjectInstruction {
		p, a := binding(m)
		return &m.Base.Base.Objects.Functions[p.Function].Blocks[a.Body].Instructions[1]
	}
	add("missing flows", func(m *PatternModule) { m.Flows = nil })
	add("missing typed tests", func(m *PatternModule) { m.Operations = nil })
	add("duplicate flow", func(m *PatternModule) { m.Flows = append(m.Flows, m.Flows[len(m.Flows)-1]) })
	add("flow capacity", func(m *PatternModule) { m.Flows = make([]PatternFlow, 8193) })
	add("operation site", func(m *PatternModule) { m.Operations[0].Instruction = 9999 })
	add("duplicate operation", func(m *PatternModule) { m.Operations = append(m.Operations, m.Operations[0]) })
	add("wrong nominal", func(m *PatternModule) { dispatch(m).ParentType = 0 })
	add("unknown variant", func(m *PatternModule) { dispatch(m).Member = 999 })
	add("source correspondence", func(m *PatternModule) { dispatch(m).Source.Line++ })
	add("duplicate pattern", func(m *PatternModule) {
		p := flow(m)
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[p.Arms[1].Dispatch].Instructions[1].Member = f.Blocks[p.Arms[0].Dispatch].Instructions[1].Member
	})
	add("active variant mismatch", func(m *PatternModule) { read(m).Member = 0 })
	add("payload field mismatch", func(m *PatternModule) { read(m).Extent = 999 })
	add("payload type mismatch", func(m *PatternModule) { read(m).Type = read(m).ParentType })
	add("missing binding", func(m *PatternModule) { _, a := binding(m); a.Bindings = nil })
	add("binding slot", func(m *PatternModule) { _, a := binding(m); a.Bindings[0].Slot = 999 })
	add("binding scope", func(m *PatternModule) {
		p, a := binding(m)
		m.Base.Base.Authority[p.Function].SlotScopes[a.Bindings[0].Slot] = p.Scope
	})
	add("escaping binding", func(m *PatternModule) {
		p, a := binding(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Join].Instructions[0].Slot = a.Bindings[0].Slot
	})
	add("nondominating payload snapshot", func(m *PatternModule) {
		p, a := binding(m)
		f := &m.Base.Base.Objects.Functions[p.Function]
		f.Blocks[a.Body].Instructions[1].Args[0].Value = f.Blocks[p.Arms[len(p.Arms)-1].Dispatch].Instructions[0].Result
	})
	add("scrutinee writable", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].Slots[p.SubjectSlot].Readonly = false
	})
	add("scrutinee alias", func(m *PatternModule) { p, a := binding(m); p.SubjectSlot = a.Bindings[0].Slot })
	add("invalid successor", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.True = 999
	})
	add("bypass selection", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.False = p.Arms[1].Body
	})
	add("join not result", func(m *PatternModule) { p := flow(m); p.ResultSlot = p.SubjectSlot })
	add("join block", func(m *PatternModule) { p := flow(m); p.Join = p.Arms[0].Body })
	add("uninitialized result", func(m *PatternModule) {
		p := flow(m)
		f := &m.Base.Base.Objects.Functions[p.Function]
		for bi := range f.Blocks {
			for ii := range f.Blocks[bi].Instructions {
				in := &f.Blocks[bi].Instructions[ii]
				if in.Slot == p.ResultSlot && in.Op == "initialize_slot" {
					in.Slot = p.SubjectSlot
					return
				}
			}
		}
	})
	add("undefined binding", func(m *PatternModule) { read(m).Args[0].Value = 511 })
	add("invalid guard operand", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Term.Value = m.Base.Base.Objects.Functions[p.Function].Blocks[p.Arms[0].Dispatch].Instructions[0].Result
	})
	add("payload geometry", func(m *PatternModule) { m.Base.Domains[0].PayloadOffset = 1 })
	add("variant tag", func(m *PatternModule) { m.Base.Domains[0].Variants[0].Tag = 99 })
	add("copy authority", func(m *PatternModule) { m.Base.Base.Objects.Types[m.Base.Domains[0].Type].Copy = "none" })
	add("foreign reference", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].Slots[p.SubjectSlot].Type = 999
	})
	add("nominal ABI return", func(m *PatternModule) {
		p := flow(m)
		m.Base.Base.Objects.Functions[p.Function].ResultType = m.Base.Domains[0].Type
	})
	for _, tc := range []struct {
		name   string
		offset int
	}{{"version", 8}, {"schema hash", 12}} {
		data := bytes.Clone(valid)
		data[tc.offset] ^= 255
		addBytes(tc.name, data)
	}
	addBytes("truncated header", valid[:20])
	addBytes("truncated payload", valid[:len(valid)-1])
	addBytes("trailing data", append(bytes.Clone(valid), 0))
	path := "../../language/evt2h3/valid/arm_regions.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	native, err := GenerateMachineIR(module)
	if err != nil {
		t.Fatal(err)
	}
	p := native.Pattern.Flows[0]
	f := &native.Pattern.Base.Base.Objects.Functions[p.Function]
	f.Blocks[p.Arms[1].Body].Term.True = f.Blocks[p.Arms[0].Body].Term.True
	raw, err := encodePatternMachineBridge(*native.Pattern)
	if err != nil {
		t.Fatal(err)
	}
	addBytes("cross-arm nested entry", raw)
	return cases
}
func TestEVT2h3MalformedContracts(t *testing.T) {
	cases := patternBadArtifacts(t)
	t.Logf("%d malformed contracts rejected by independent Go consistency checks", len(cases))
}
func TestEVT2h3MalformedArtifactAdmission(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	cases := patternBadArtifacts(t)
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
	for _, verify := range []bool{false, true} {
		t.Log(runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "match_malformed.c", host, "-pedantic-errors", "-O2"))
	}
}

const patternMalformedHost = `#include "amd64.generated.h"
#include <stdio.h>
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
static unsigned char input[2000000],output[262144];static concept_native_image image,before;
static const struct{const char*path,*name;}cases[]={/* CASES */};
int main(void){for(size_t i=0;i<sizeof cases/sizeof cases[0];i++){
 FILE*f=fopen(cases[i].path,"rb");if(!f)return 1;size_t n=fread(input,1,sizeof input,f);fclose(f);
 concept_readonly_span_byte src={input,n};concept_result_void_pattern_error a=F(validate_pattern_machine_bridge)(src);
 if(!a.tag){fprintf(stderr,"Concept accepted %s\n",cases[i].name);return 2;}
 memset(output,0xa5,sizeof output);concept_result_int_pattern_error rt=F(validated_pattern_bridge_round_trip)(src,(concept_span_byte){output,sizeof output});
 if(!rt.tag){fprintf(stderr,"roundtrip accepted %s\n",cases[i].name);return 3;}
 for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 4;
 memset(&image,0x5a,sizeof image);before=image;concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){output,sizeof output},&image);
 if(!emitted.tag||memcmp(&image,&before,sizeof image)){fprintf(stderr,"emission accepted/mutated %s\n",cases[i].name);return 5;}
 for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 6;
 printf("%s: %s; roundtrip/code/image sentinels preserved\n",cases[i].name,F(describe_pattern_error)(a.payload.error.error));
 }return 0;}`
