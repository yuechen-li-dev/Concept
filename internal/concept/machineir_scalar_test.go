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

func scalarMachineFixture(t *testing.T) MachineModule {
	t.Helper()
	m, _ := evt2fScalarFixture(t)
	out, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func scalarFunction(m *ScalarMachineModule, name string) *ScalarMachineFunction {
	for i := range m.Functions {
		if m.Functions[i].Name == name {
			return &m.Functions[i]
		}
	}
	panic(name)
}
func scalarInstruction(f *ScalarMachineFunction, op string) *ScalarMachineInstruction {
	for bi := range f.Blocks {
		for ii := range f.Blocks[bi].Instructions {
			if f.Blocks[bi].Instructions[ii].Op == op {
				return &f.Blocks[bi].Instructions[ii]
			}
		}
	}
	panic(op)
}
func scalarClone(t *testing.T, m MachineModule) MachineModule {
	t.Helper()
	b, e := EncodeMachineBridge(m)
	if e != nil {
		t.Fatal(e)
	}
	out, e := DecodeScalarMachineBridge(b)
	if e != nil {
		t.Fatal(e)
	}
	return out
}

func TestEVT2f2MachineIRTransport(t *testing.T) {
	m := scalarMachineFixture(t)
	text := m.String()
	blob, err := EncodeMachineBridge(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"CMIRAMD4 v4", "float32 / XMM width=4", "float64 / XMM width=8", "bool / GPR width=1", "fconst float32", "literal=\"0x3dcccccd\"", "literal=\"0x3ff0000000000001\"", "fadd", "fsub", "fmul", "fdiv", "fneg", "feq", "fne", "flt", "fle", "fgt", "fge", "virtual)"} {
		if !strings.Contains(text, s) {
			t.Fatalf("missing %s", s)
		}
	}
	if strings.Contains(text, "xmm0") || strings.Contains(text, "rax") {
		t.Fatal("physical location leaked into v4")
	}
	for _, name := range []string{"Caller", "Caller8"} {
		f := scalarFunction(m.Scalar, name)
		in := scalarInstruction(f, "call")
		expected := []string{"i32", "float64", "i32", "float32"}
		if name == "Caller8" {
			expected = []string{"float64", "i32", "float32", "i32", "float64", "float32", "i32", "float64"}
		}
		for i, a := range in.Calls[0].Arguments {
			if a.Type != expected[i] || a.Value != in.Args[i].Value {
				t.Fatalf("mixed order lost: %v", in.Calls[0])
			}
		}
	}
	for _, name := range []string{"InterpretPressure", "InterpretDistance"} {
		f := scalarFunction(m.Scalar, name)
		if len(f.Blocks[0].Instructions) != 0 || f.Blocks[0].Term.Value != f.Params[0].Value {
			t.Fatal("interpret gained a runtime op")
		}
	}
	start := time.Now()
	for i := 0; i < 100; i++ {
		next := scalarMachineFixture(t)
		data, e := EncodeMachineBridge(next)
		if e != nil || !bytes.Equal(blob, data) || next.String() != text {
			t.Fatalf("encode differs run %d: %v", i, e)
		}
		decoded, e := DecodeScalarMachineBridge(data)
		if e != nil || decoded.String() != text {
			t.Fatalf("decode differs run %d: %v", i, e)
		}
		again, e := EncodeMachineBridge(decoded)
		if e != nil || !bytes.Equal(data, again) || digest(blob) != digest(again) {
			t.Fatalf("roundtrip differs: %v", e)
		}
	}
	t.Logf("float module=%d bytes, 100 generate/encode/decode/roundtrips=%s", len(blob), time.Since(start))
	integer := callMachineFixture(t)
	ib, e := EncodeMachineBridge(integer)
	if e != nil {
		t.Fatal(e)
	}
	if string(ib[:8]) != "CMIRAMD3" {
		t.Fatal("integer version changed")
	}
	t.Logf("representative integer CMIRAMD3=%d bytes", len(ib))
	for _, encode := range []bool{true, false} {
		start := time.Now()
		for i := 0; i < 1000; i++ {
			if encode {
				_, err = EncodeMachineBridge(m)
			} else {
				_, err = DecodeScalarMachineBridge(blob)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("1000 iterations encode=%t: %s", encode, time.Since(start))
	}
}

func TestEVT2f2DigestSensitivity(t *testing.T) {
	base := scalarMachineFixture(t)
	b, err := EncodeMachineBridge(base)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*ScalarMachineModule){
		"float32-bits": func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0x3dccccce"
		},
		"float64-bits": func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant64"), "fconst").Literal = "0x3ff0000000000002"
		},
		"operation": func(m *ScalarMachineModule) { scalarInstruction(&m.Functions[0], "fadd").Op = "fsub" },
		"representation": func(m *ScalarMachineModule) {
			f := scalarFunction(m, "Constant32")
			f.Result = "float64"
			in := scalarInstruction(f, "fconst")
			in.Representation = "float64"
			in.Literal = "0x3fb99999a0000000"
			f.VRegs[in.Result].Representation = "float64"
			f.VRegs[in.Result].Width = 8
		},
	} {
		m := scalarClone(t, base)
		change(m.Scalar)
		data, e := EncodeMachineBridge(m)
		if e != nil || digest(data) == digest(b) {
			t.Fatalf("%s identity unchanged: %v", name, e)
		}
	}
	var previous string
	for _, signature := range []string{"int a,double b,int c,float d", "int a,float b,int c,float d", "float d,double b,int c,int a"} {
		var arguments []string
		for _, parameter := range strings.Split(signature, ",") {
			arguments = append(arguments, strings.Fields(parameter)[1])
		}
		source := "profile Core; double Target(" + signature + "){return 1.0;} double Caller(" + signature + "){return Target(" + strings.Join(arguments, ",") + ");}"
		m, e := Parse("digest.concept", source)
		if e != nil {
			t.Fatal(e)
		}
		mir, e := GenerateMachineIR(m)
		if e != nil {
			t.Fatal(e)
		}
		data, e := EncodeMachineBridge(mir)
		if e != nil {
			t.Fatal(e)
		}
		if previous == digest(data) {
			t.Fatal("call type/order identity unchanged")
		}
		previous = digest(data)
	}
}

func TestEVT2f2SchemaAndExactRecord(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeSchema4.concept"
	src, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	m, e := ParseWithBuiltSemanticModuleRoots(path, string(src), []string{"../../libraries"})
	if e != nil {
		t.Fatal(e)
	}
	g, e := GenerateMachineBridgeCodecs(m)
	if e != nil {
		t.Fatal(e)
	}
	for path, want := range map[string][]byte{"../../internal/concept/machineir_bridge4_codec_generated.go": g.Go, "../../libraries/Standard/Backend/BridgeCodec4.concept": g.Concept, "../../docs/design/EVT2-MACHINEIR-BRIDGE4.schema.json": g.Metadata} {
		got, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
			t.Fatalf("stale generated %s: %v", path, e)
		}
	}
	for i := 0; i < 100; i++ {
		next, e := GenerateMachineBridgeCodecs(m)
		if e != nil || next.Hash != g.Hash || !bytes.Equal(next.Go, g.Go) || !bytes.Equal(next.Concept, g.Concept) {
			t.Fatalf("schema generation changed: %v", e)
		}
	}
	w := &machineBridgeWriter{}
	if e := w.writeWireScalarMachineVReg(ScalarMachineVReg{ID: 3, Class: "XMM", Representation: "float32", Width: 4}); e != nil {
		t.Fatal(e)
	}
	if got := hex.EncodeToString(w.Bytes()); got != "03000000010000000a0000000400000000000000" {
		t.Fatalf("float record transport bytes=%s", got)
	}
	t.Logf("%s v%d hash=%s; 20-byte float vreg transport golden", g.Magic, g.Version, g.Hash)
}

func scalarMalformedArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	base := scalarMachineFixture(t)
	cases := map[string][]byte{}
	mutations := []struct {
		name   string
		change func(*ScalarMachineModule)
	}{
		{"float32-gpr", func(m *ScalarMachineModule) { m.Functions[0].VRegs[0].Class = "GPR" }},
		{"float64-gpr", func(m *ScalarMachineModule) { scalarFunction(m, "Arithmetic64").VRegs[0].Class = "GPR" }},
		{"integer-xmm", func(m *ScalarMachineModule) { scalarFunction(m, "Mixed").VRegs[0].Class = "XMM" }},
		{"bool-xmm", func(m *ScalarMachineModule) { f := scalarFunction(m, "Equal32"); f.VRegs[2].Class = "XMM" }},
		{"float-width", func(m *ScalarMachineModule) { m.Functions[0].VRegs[0].Width = 8 }},
		{"float-address", func(m *ScalarMachineModule) { m.Functions[0].VRegs[0].Address = true }},
		{"result-mismatch", func(m *ScalarMachineModule) { scalarInstruction(&m.Functions[0], "fadd").Representation = "float64" }},
		{"mixed-float-width", func(m *ScalarMachineModule) {
			v := &m.Functions[0].VRegs[0]
			v.Representation = "float64"
			v.Width = 8
			m.Functions[0].Params[0].Type = "float64"
		}},
		{"integer-float-operation", func(m *ScalarMachineModule) { scalarInstruction(&m.Functions[0], "fadd").Op = "add" }},
		{"wrong-negation-arity", func(m *ScalarMachineModule) {
			in := scalarInstruction(scalarFunction(m, "Negative32"), "fneg")
			in.Args = append(in.Args, in.Args[0])
		}},
		{"integer-compare-float", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "Equal32"), "feq").Op = "eq" }},
		{"compare-xmm-bool", func(m *ScalarMachineModule) {
			f := scalarFunction(m, "Equal32")
			in := scalarInstruction(f, "feq")
			in.Representation = "float32"
			f.VRegs[in.Result].Representation = "float32"
			f.VRegs[in.Result].Class = "XMM"
			f.VRegs[in.Result].Width = 4
		}},
		{"constant32-short", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0x0"
		}},
		{"constant32-long", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0x0000000000"
		}},
		{"constant64-short", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant64"), "fconst").Literal = "0x00000000"
		}},
		{"constant64-long", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant64"), "fconst").Literal = "0x000000000000000000"
		}},
		{"constant-uppercase", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0x3DCCCCCD"
		}},
		{"constant-decimal", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0.1"
		}},
		{"constant-prefix", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0X3dcccccd"
		}},
		{"constant-nonhex", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Literal = "0x3gcccccd"
		}},
		{"float-bits-in-integer-form", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Constant32"), "fconst").Op = "const"
		}},
		{"float-constant-integer-result", func(m *ScalarMachineModule) {
			f := scalarFunction(m, "Constant32")
			in := scalarInstruction(f, "fconst")
			in.Representation = "i32"
			f.VRegs[in.Result].Representation = "i32"
			f.VRegs[in.Result].Class = "GPR"
			f.Result = "i32"
		}},
		{"slot-width", func(m *ScalarMachineModule) { m.Functions[0].Slots[0].Size = 8 }},
		{"slot-alignment", func(m *ScalarMachineModule) { m.Functions[0].Slots[0].Align = 8 }},
		{"slot-representation", func(m *ScalarMachineModule) { m.Functions[0].Slots[0].Representation = "i32" }},
		{"store-representation", func(m *ScalarMachineModule) { scalarInstruction(&m.Functions[0], "store_slot").Representation = "i32" }},
		{"index-stride", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Load32"), "index_address").Stride = 8
		}},
		{"array-type", func(m *ScalarMachineModule) { scalarFunction(m, "Load32").Params[0].Type = "[4]float64" }},
		{"missing-index-guard", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "Load32"), "check_index").Extent = 3 }},
		{"call-arg-type", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Arguments[1].Type = "float32"
		}},
		{"call-result-type", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Result = "i64"
		}},
		{"call-reorder", func(m *ScalarMachineModule) {
			a := scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Arguments
			a[1], a[3] = a[3], a[1]
		}},
		{"call-void-result", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Result = "void"
		}},
		{"call-no-result", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "Caller"), "call").Result = -1 }},
		{"call-empty", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "Caller"), "call").Calls = nil }},
		{"call-target", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Target = "Missing|Missing()"
		}},
		{"call-convention", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "Caller"), "call").Calls[0].Convention = "sysv"
		}},
		{"provenance", func(m *ScalarMachineModule) { m.Functions[0].Blocks[0].Instructions[0].LIRInstruction = 99 }},
		{"undefined-value", func(m *ScalarMachineModule) { m.Functions[0].Blocks[0].Instructions[0].Args[0].Value = 127 }},
		{"return-type", func(m *ScalarMachineModule) { m.Functions[0].Result = "float64" }},
		{"duplicate-identity", func(m *ScalarMachineModule) { m.Functions[1].Identity = m.Functions[0].Identity }},
	}
	for _, tc := range mutations {
		m := scalarClone(t, base)
		tc.change(m.Scalar)
		if e := VerifyMachineIR(m); e == nil {
			t.Fatalf("Go accepted mutation %s", tc.name)
		}
		data, e := encodeScalarMachineBridge(*m.Scalar)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = DecodeScalarMachineBridge(data); e == nil {
			t.Fatalf("Go reader accepted %s", tc.name)
		}
		cases[tc.name] = data
	}
	valid, e := EncodeMachineBridge(base)
	if e != nil {
		t.Fatal(e)
	}
	for _, version := range []int{1, 2, 3} {
		b := append([]byte(nil), valid...)
		b[7] = byte('0' + version)
		binary.LittleEndian.PutUint32(b[8:12], uint32(version))
		cases[fmt.Sprintf("legacy-v%d", version)] = b
	}
	b := append([]byte(nil), valid...)
	b[12] ^= 1
	cases["stale-v4-hash"] = b
	b = append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(b[8:12], 3)
	cases["stale-number"] = b
	cases["trailing-byte"] = append(append([]byte(nil), valid...), 0)
	cases["truncated-payload"] = valid[:len(valid)-1]
	cases["truncated-header"] = valid[:40]
	var fields []machineBridgeWireField
	w := &machineBridgeWriter{trace: &fields}
	if e := w.writeWireScalarMachineModule(*base.Scalar); e != nil {
		t.Fatal(e)
	}
	for _, target := range []struct{ record, field, name string }{{"WireScalarMachineVReg", "Representation", "invalid-representation"}, {"WireScalarMachineVReg", "Class", "invalid-class"}, {"WireScalarMachineCallArgument", "Type", "unsupported-float16-call-tag"}} {
		for _, field := range fields {
			if field.Record == target.record && field.Field == target.field {
				b := append([]byte(nil), valid...)
				binary.LittleEndian.PutUint32(b[44+field.Offset:44+field.Offset+4], 999)
				cases[target.name] = b
				break
			}
		}
	}
	for name, data := range cases {
		if _, e := DecodeScalarMachineBridge(data); e == nil {
			t.Fatalf("strict Go reader accepted %s", name)
		}
	}
	return cases
}

func TestEVT2f2ConceptArtifactOnlyRoundTrip(t *testing.T) {
	cases := scalarMalformedArtifacts(t)
	valid, e := EncodeMachineBridge(scalarMachineFixture(t))
	if e != nil {
		t.Fatal(e)
	}
	cases["valid"] = valid
	comparison, e := Parse("special-comparison.concept", "profile Core; bool Equal(float x){float value=0.1;return value == x;} bool Unequal(double x){return 1.0 != x;} void Sink(float x){} void Caller(float x){Sink(x);}")
	if e != nil {
		t.Fatal(e)
	}
	comparisonMIR, e := GenerateMachineIR(comparison)
	if e != nil {
		t.Fatal(e)
	}
	for _, rep := range []string{"float32", "float64"} {
		name, op, bits := "Equal", "feq", []string{"0x00000000", "0x80000000", "0x7f800000", "0xff800000", "0x7fc01234"}
		if rep == "float64" {
			name, op, bits = "Unequal", "fne", []string{"0x0000000000000000", "0x8000000000000000", "0x7ff0000000000000", "0xfff0000000000000", "0x7ff8000000001234"}
		}
		for _, literal := range bits {
			m := scalarClone(t, comparisonMIR)
			f := scalarFunction(m.Scalar, name)
			scalarInstruction(f, "fconst").Literal = literal
			scalarInstruction(f, op)
			data, e := EncodeMachineBridge(m)
			if e != nil {
				t.Fatal(e)
			}
			cases["valid-comparison-"+rep+literal] = data
		}
	}
	// Payload-bearing special constants remain unchanged, including comparisons.
	base := scalarMachineFixture(t)
	for _, rep := range []string{"float32", "float64"} {
		bits := []string{"0x00000000", "0x80000000", "0x7f800000", "0xff800000", "0x7fc01234"}
		name := "Constant32"
		if rep == "float64" {
			bits = []string{"0x0000000000000000", "0x8000000000000000", "0x7ff0000000000000", "0xfff0000000000000", "0x7ff8000000001234"}
			name = "Constant64"
		}
		for _, literal := range bits {
			m := scalarClone(t, base)
			scalarInstruction(scalarFunction(m.Scalar, name), "fconst").Literal = literal
			data, e := EncodeMachineBridge(m)
			if e != nil {
				t.Fatal(e)
			}
			cases["valid-special-"+rep+literal] = data
		}
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <string.h>\n#include <stdio.h>\n#include <time.h>\nint main(void){\n")
	for index, name := range names {
		data := cases[name]
		fmt.Fprint(&c, "{static const unsigned char input[]={")
		for _, v := range data {
			fmt.Fprintf(&c, "%d,", v)
		}
		fmt.Fprintf(&c, "};unsigned char output[%d];memset(output,0x5a,sizeof output);concept_readonly_span_byte src={input,sizeof input};concept_span_byte dst={output,sizeof output};\n", len(data))
		runs := 1
		if name == "valid" {
			runs = 100
		}
		fmt.Fprintf(&c, "clock_t start=clock();for(int run=0;run<%d;run++){concept_result_int_backend_error r=concept_standard__backend__amd64_validated_scalar_bridge_round_trip(src,dst);\n", runs)
		if strings.HasPrefix(name, "valid") {
			fmt.Fprintf(&c, "if(r.tag || r.payload.ok.value!=(int)sizeof input || memcmp(input,output,sizeof input)){fprintf(stderr,\"roundtrip %s failed\\n\");return %d;}\n", name, index+1)
		} else {
			fmt.Fprintf(&c, "if(!r.tag){fprintf(stderr,\"accepted %s\\n\");return %d;}for(size_t i=0;i<sizeof output;i++)if(output[i]!=0x5a)return %d;\n", name, index+1, index+1)
		}
		c.WriteString("}\n")
		if name == "valid" || (!strings.HasPrefix(name, "valid") && len(data) >= 8 && string(data[:8]) == "CMIRAMD4") {
			want := "AMD64_NATIVE_INVALID_INPUT"
			if name == "valid" {
				want = "AMD64_UNSUPPORTED_XMM_ALLOCATION"
			}
			fmt.Fprintf(&c, "memset(output,0x5a,sizeof output);concept_native_image image,before;memset(&image,0x3c,sizeof image);memcpy(&before,&image,sizeof image);concept_result_int_native_error native=concept_standard__backend__amd64_emit_module(src,dst,&image);if(!native.tag || strcmp(concept_standard__backend__amd64_describe_native_error(native.payload.error.error),\"%s\") || memcmp(&image,&before,sizeof image))return %d;for(size_t i=0;i<sizeof output;i++)if(output[i]!=0x5a)return %d;\n", want, index+1, index+1)
		}
		if name == "valid" {
			c.WriteString("printf(\"Concept 100 validated v4 roundtrips %.3f ms\\n\",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);\n")
		}
		c.WriteString("}\n")
	}
	c.WriteString("return 0;}\n")
	for _, verify := range []bool{false, true} {
		t.Run(fmt.Sprint(verify), func(t *testing.T) {
			runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "scalar_transport_host.c", c.String(), "-pedantic-errors", "-O2")
		})
	}
	t.Logf("%d malformed artifacts rejected in both Concept and Go; source-free exact roundtrip", len(cases)-21)
}
