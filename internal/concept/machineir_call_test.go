package concept

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func callMachineFixture(t *testing.T) MachineModule {
	t.Helper()
	path := "testdata/evt2e_calls.concept"
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
	return machine
}

func callInstruction(t *testing.T, m *MachineModule, name string) *MachineInstruction {
	t.Helper()
	for fi := range m.Functions {
		f := &m.Functions[fi]
		if f.Name != name {
			continue
		}
		for bi := range f.Blocks {
			for ii := range f.Blocks[bi].Instructions {
				in := &f.Blocks[bi].Instructions[ii]
				if in.Op == "CALL" && in.Calls[0].Result != "void" {
					return in
				}
			}
		}
	}
	t.Fatalf("missing scalar call in %s", name)
	return nil
}

func uncheckedCallArtifact(t *testing.T, m MachineModule, header []byte) []byte {
	t.Helper()
	w := &machineBridgeWriter{}
	if err := w.writeWireMachineModule(m); err != nil {
		t.Fatal(err)
	}
	return append(bytes.Clone(header[:machineBridgeHeaderSize]), w.Bytes()...)
}

func callMalformedArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	base := callMachineFixture(t)
	valid, err := EncodeMachineBridge(base)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{}
	mutations := map[string]func(*MachineModule, *MachineInstruction){
		"missing callee":          func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Target = "" },
		"invalid callee identity": func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Target = "Add" },
		"malformed matched identity": func(m *MachineModule, in *MachineInstruction) {
			for i := range m.Functions {
				if m.Functions[i].Name == "Add" {
					m.Functions[i].Identity = "Add|Bogus()"
				}
			}
			in.Calls[0].Target = "Add|Bogus()"
		},
		"unavailable callee":   func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Target = "Missing|Missing(int, int)" },
		"wrong argument count": func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Arguments = in.Calls[0].Arguments[:1] },
		"reordered metadata": func(m *MachineModule, in *MachineInstruction) {
			x := callInstruction(t, m, "MixedCall")
			x.Calls[0].Arguments[0], x.Calls[0].Arguments[1] = x.Calls[0].Arguments[1], x.Calls[0].Arguments[0]
		},
		"mismatched argument class": func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Arguments[0].Type = "u32" },
		"void argument class":       func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Arguments[0].Type = "void" },
		"void with result":          func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Result = "void" },
		"missing scalar result":     func(m *MachineModule, in *MachineInstruction) { in.Dst = MachineOperand{}; in.Width = 0 },
		"mismatched result class":   func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Result = "u32" },
		"invalid virtual value":     func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Arguments[0].Value = 999999 },
		"negative virtual value":    func(m *MachineModule, in *MachineInstruction) { in.Calls[0].Arguments[0].Value = -1 },
		"duplicate call contract":   func(m *MachineModule, in *MachineInstruction) { in.Calls = append(in.Calls, in.Calls[0]) },
		"missing provenance":        func(m *MachineModule, in *MachineInstruction) { in.Source.Line = 0 },
		"wrong LIR provenance":      func(m *MachineModule, in *MachineInstruction) { in.LIRBlock = 999 },
	}
	for name, mutate := range mutations {
		m := callMachineFixture(t)
		in := callInstruction(t, &m, "AddThenCall")
		mutate(&m, in)
		if err := VerifyMachineIR(m); err == nil {
			t.Fatalf("verifier accepted %s", name)
		}
		cases[name] = uncheckedCallArtifact(t, m, valid)
	}
	_, fields := bridgeTracedPayload(t, base)
	for name, field := range map[string]string{"unknown convention": "Convention", "unsupported indirect tag": "Kind", "invalid result enum": "Result"} {
		bad := bytes.Clone(valid)
		offset := machineBridgeHeaderSize + bridgeFieldOffset(t, fields, "WireMachineCall", field)
		binary.LittleEndian.PutUint32(bad[offset:], 0x7fffffff)
		cases[name] = bad
	}
	bad := bytes.Clone(valid)
	offset := machineBridgeHeaderSize + bridgeFieldOffset(t, fields, "WireMachineCallArgument", "Type")
	binary.LittleEndian.PutUint32(bad[offset:], 0x7fffffff)
	cases["invalid argument enum"] = bad
	cases["truncated call"] = bytes.Clone(valid[:offset+2])
	bad = bytes.Clone(valid)
	copy(bad[:8], "CMIRAMD2")
	binary.LittleEndian.PutUint32(bad[8:12], 2)
	cases["CMIRAMD2 version"] = bad
	bad = bytes.Clone(valid)
	bad[12] ^= 1
	cases["stale schema"] = bad
	return cases
}

func TestEVT2e2CallTransportAndMalformed(t *testing.T) {
	m := callMachineFixture(t)
	artifact, err := EncodeMachineBridge(m)
	if err != nil {
		t.Fatal(err)
	}
	if MachineBridgeSchema != "CMIRAMD3" || MachineBridgeVersion != 3 {
		t.Fatal("call bridge is not version 3")
	}
	if MachineBridgeSchemaHash != "3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312" {
		t.Fatal("call schema fingerprint changed without requalification")
	}
	for i := 0; i < 100; i++ {
		regenerated := callMachineFixture(t)
		if regenerated.String() != m.String() {
			t.Fatalf("MachineIR printer changed run %d", i)
		}
		next, err := EncodeMachineBridge(regenerated)
		if err != nil || !bytes.Equal(next, artifact) {
			t.Fatalf("artifact changed run %d: %v", i, err)
		}
		decoded, err := DecodeMachineBridge(next)
		if err != nil || decoded.String() != m.String() {
			t.Fatalf("roundtrip changed run %d: %v", i, err)
		}
	}
	for name, data := range callMalformedArtifacts(t) {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeMachineBridge(data); err == nil {
				t.Fatal("malformed artifact accepted")
			}
		})
	}
	t.Logf("%d malformed call artifacts rejected; fixture %d bytes", len(callMalformedArtifacts(t)), len(artifact))
	started := time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := EncodeMachineBridge(m); err != nil {
			t.Fatal(err)
		}
	}
	encodeTime := time.Since(started)
	started = time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := DecodeMachineBridge(artifact); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("fixture 1000 encodes=%s; 1000 decodes=%s", encodeTime, time.Since(started))
}

func TestEVT2e2GeneratedCallDeclarations(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeCodec.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	count, callCount := 0, 0
	for _, fn := range module.Functions {
		if fn.Generated == nil {
			continue
		}
		count++
		if fn.Generated.ReflectedType.Name == "WireMachineCall" || fn.Generated.ReflectedType.Name == "WireMachineCallArgument" {
			callCount++
		}
	}
	if count != 41 || callCount != 4 {
		t.Fatalf("generated declarations=%d call records=%d", count, callCount)
	}
	view, err := InspectGeneratedDeclarations(module, "")
	if err != nil || !bytes.Contains(view, []byte("WireMachineCall")) {
		t.Fatalf("missing call declarations: %v", err)
	}
	t.Logf("generated declarations=%d, baseline=31, delta=10", count)
}

func TestEVT2e2CallRecordExactBytesAndIdentity(t *testing.T) {
	record := MachineCall{Kind: "direct", Target: "Id|Id(int)", Convention: "win64", Arguments: []MachineCallArgument{{Value: 7, Type: "i32"}}, Result: "i32"}
	w := &machineBridgeWriter{}
	if err := w.writeWireMachineCall(record); err != nil {
		t.Fatal(err)
	}
	// Direct/Win64 tags are 0; i32 is tag 6 in the Concept-authored enum.
	want := "000000000a00000049647c496428696e74290000000001000000070000000600000006000000"
	if hex.EncodeToString(w.Bytes()) != want {
		t.Fatalf("call bytes %x", w.Bytes())
	}
	baseline := digest(w.Bytes())
	for _, mutate := range []func(*MachineCall){
		func(c *MachineCall) { c.Target = "Other|Other(int)" },
		func(c *MachineCall) { c.Arguments = []MachineCallArgument{{Value: 8, Type: "i32"}} },
		func(c *MachineCall) { c.Arguments = []MachineCallArgument{{Value: 7, Type: "u32"}} },
		func(c *MachineCall) { c.Result = "void" },
	} {
		changed := record
		mutate(&changed)
		out := &machineBridgeWriter{}
		if err := out.writeWireMachineCall(changed); err != nil {
			t.Fatal(err)
		}
		if digest(out.Bytes()) == baseline {
			t.Fatal("semantic call mutation lost from identity")
		}
	}
	ordered := record
	ordered.Arguments = []MachineCallArgument{{Value: 7, Type: "i32"}, {Value: 8, Type: "i32"}}
	a := &machineBridgeWriter{}
	if err := a.writeWireMachineCall(ordered); err != nil {
		t.Fatal(err)
	}
	ordered.Arguments[0], ordered.Arguments[1] = ordered.Arguments[1], ordered.Arguments[0]
	b := &machineBridgeWriter{}
	if err := b.writeWireMachineCall(ordered); err != nil {
		t.Fatal(err)
	}
	if digest(a.Bytes()) == digest(b.Bytes()) {
		t.Fatal("argument order omitted from identity")
	}
	changedConvention := bytes.Clone(w.Bytes())
	binary.LittleEndian.PutUint32(changedConvention[18:22], 1)
	if digest(changedConvention) == baseline {
		t.Fatal("convention omitted from identity")
	}
	start := time.Now()
	for i := 0; i < 10000; i++ {
		out := &machineBridgeWriter{}
		if err := out.writeWireMachineCall(record); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("minimal call record=%d bytes; encode 10000 records=%s", w.Len(), time.Since(start))
}

func TestEVT2e2ConceptArtifactOnlyCallRoundTrip(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeValidate.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	// The wire-view assertions use Vocabulary's typed Verdict rather than an
	// invented runtime proof protocol. Retain its measured geometry evidence.
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"WireMachineCall", "WireMachineCallArgument"} {
		subject, err := evt1ResolveConceptAssertionSubject(env, newEVT1Scope(nil), &NameExpr{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		verdict, err := evt1InvokePredicateOnMeasured(env, env, "PlainDataHolds", []Value{evt1TypenameValue(subject.typeValue)}, Span{}, nil)
		if err != nil || verdict.Evidence == nil || verdict.Evidence.Fields["size"].UintValue == 0 || verdict.Evidence.Fields["alignment"].UintValue == 0 {
			t.Fatalf("%s lacks typed PlainData geometry evidence: %+v %v", name, verdict, err)
		}
		t.Logf("%s Verdict evidence: size=%d alignment=%d", name, verdict.Evidence.Fields["size"].UintValue, verdict.Evidence.Fields["alignment"].UintValue)
	}
	valid, err := EncodeMachineBridge(callMachineFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	cases := callMalformedArtifacts(t)
	cases["valid"] = valid
	var names []string
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	var c strings.Builder
	c.WriteString("#include \"bridgevalidate.generated.h\"\n#include <string.h>\n#include <stdio.h>\n#include <time.h>\nint main(void){\n")
	for index, name := range names {
		data := cases[name]
		fmt.Fprint(&c, "{static const unsigned char input[]={")
		for _, b := range data {
			fmt.Fprintf(&c, "%d,", b)
		}
		fmt.Fprintf(&c, "};unsigned char output[%d]; memset(output,0x5a,sizeof output);concept_readonly_span_byte src={input,sizeof input};concept_span_byte dst={output,sizeof output};\n", len(data))
		runs := 2
		if name == "valid" {
			runs = 100
		}
		fmt.Fprintf(&c, "clock_t start=clock(); for(int run=0;run<%d;++run){concept_result_int_backend_error r=concept_standard__backend__bridge_validate_validated_bridge_round_trip(src,dst);\n", runs)
		if name == "valid" {
			fmt.Fprintf(&c, "if(r.tag || r.payload.ok.value!=(int)sizeof input || memcmp(input,output,sizeof input))return %d;\n", index+1)
		} else {
			fmt.Fprintf(&c, "if(r.tag!=1)return %d;for(size_t i=0;i<sizeof output;++i){if(output[i]!=0x5a)return %d;}\n", index+1, index+1)
		}
		c.WriteString("}\n")
		if name == "valid" {
			c.WriteString("printf(\"Concept 100 validated artifact roundtrips: %.3f ms\\n\",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);\n")
		}
		c.WriteString("}\n")
	}
	c.WriteString("return 0;}\n")
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		mode := "Normal"
		if verify {
			policy = VerifyCompilationPolicy()
			mode = "Verify"
		}
		outputs, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(mode, func(t *testing.T) { runFoundationNativeHarness(t, outputs, "call_contract.c", c.String()) })
	}
	t.Logf("Concept artifact-only exact roundtrip and %d malformed cases", len(cases)-1)
}
