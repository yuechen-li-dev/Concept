package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2g3bCheckedReferences(t *testing.T) {
	path := "../../language/evt2g3b/valid/references.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
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
	if _, err = DecodeMachineBridge(artifact); err != nil {
		t.Fatal(err)
	}
	baseline, err := json.Marshal(lir)
	if err != nil {
		t.Fatal(err)
	}
	debug := machine.String()
	lirText := lir.String()
	for run := 0; run < 100; run++ {
		next, err := GenerateLIR(m)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(next)
		if !bytes.Equal(baseline, data) || next.String() != lirText {
			t.Fatalf("reference LIR/order/debug changed at %d", run)
		}
		lowered, err := LowerLirToAmd64Machine(next)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeMachineBridge(lowered)
		if err != nil || !bytes.Equal(encoded, artifact) || lowered.String() != debug {
			t.Fatalf("reference artifact/debug changed at %d: %v", run, err)
		}
	}
	w := &machineBridgeWriter{}
	if err := w.writeWireObjectModule(machine.Borrow.Objects); err != nil {
		t.Fatal(err)
	}
	refs, scopes := 0, 0
	for _, f := range machine.Borrow.Authority {
		refs += len(f.References)
		scopes += len(f.Scopes)
	}
	t.Logf("reference rows=%d scopes=%d metadata=%d embedded-object-payload=%d bytes; 100 fresh LIR/artifact/debug cycles passed", refs, scopes, len(artifact)-BorrowMachineBridgeHeaderSize-w.Len(), w.Len())
	t.Logf("CMIRAMD6 artifact=%d functions=%d", len(artifact), len(machine.Borrow.Objects.Functions))
}

func TestEVT2g3bReferenceProofBoundaries(t *testing.T) {
	proof := `module BorrowFacts; profile Core; import Standard.Semantic.Vocabulary; struct Position{double x;double y;} using Held=ref Position; void Facts(){Assert.Concept<PlainData>(Held,"a reference is not raw transport data");}`
	if _, err := ParseWithBuiltSemanticModuleRoots("borrowfacts.concept", proof, []string{"../../libraries"}); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
		t.Fatalf("reference PlainData refutation: %v", err)
	}
	m, err := Parse("return_ref.concept", `profile Core; ref int Identity(ref int value){return ref value;}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateMachineIR(m); err == nil || !strings.Contains(err.Error(), "REFERENCE_RETURN_UNSUPPORTED") {
		t.Fatalf("reference return must remain a specific boundary: %v", err)
	}
}

func TestEVT2g3bGeneratedAndExactTransport(t *testing.T) {
	schemaPath := "../../libraries/Standard/Backend/BridgeSchema6.concept"
	source, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := ParseWithBuiltSemanticModuleRoots(schemaPath, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateMachineBridgeCodecs(schema)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{"machineir_bridge6_codec_generated.go": generated.Go, "../../libraries/Standard/Backend/BridgeCodec6.concept": generated.Concept, "../../docs/design/EVT2-MACHINEIR-BRIDGE6.schema.json": generated.Metadata} {
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("regenerate %s: %v", path, err)
		}
	}
	if generated.Hash != BorrowMachineBridgeSchemaHash || BorrowMachineBridgeVersion != 6 || ObjectMachineBridgeSchemaHash != "7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60" {
		t.Fatal("wire schema identity drift")
	}
	const minimal = `module RefWire; profile Core; int Read(ref const int value){return value;}`
	m, err := Parse("refwire.concept", minimal)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	exact := []byte(hex.EncodeToString(data) + "\n")
	path := filepath.Join("testdata", "evt2g3b_ref.hex")
	if os.Getenv("CONCEPT_UPDATE_CHECKED_OUTPUTS") == "1" {
		if err := os.WriteFile(path, exact, 0644); err != nil {
			t.Fatal(err)
		}
	}
	pinned, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(bytes.TrimSpace(pinned), bytes.TrimSpace(exact)) {
		t.Fatalf("exact reference wire bytes changed: %v", err)
	}
	sum := sha256.Sum256(data)
	t.Logf("CMIRAMD6 %s; minimal exact artifact %d bytes sha256=%x", generated.Hash, len(data), sum)
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <windows.h>\n#include <stdio.h>\n#include <string.h>\nstatic const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};\n#define F(n) concept_standard__backend__amd64_##n\nint main(void){unsigned char bytes[4096];concept_native_image image;concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(r.tag){fprintf(stderr,\"%s\\n\",F(describe_native_error)(r.payload.error.error));return 1;}unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 2;memcpy(memory,bytes,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 3;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);typedef int32_t(*Read)(const int32_t*);Read fn=NULL;void *p=memory;memcpy(&fn,&p,sizeof fn);int32_t value=-37;int result=fn(&value);VirtualFree(memory,0,MEM_RELEASE);return result==-37?0:4;}\n")
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		for _, verify := range []bool{false, true} {
			runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "minimal_reference.c", c.String(), "-pedantic-errors", "-O2")
		}
	}
}

func TestEVT2g3bSourceNegatives(t *testing.T) {
	for _, name := range []string{"const_store", "const_escalation", "wrong_target", "local_escape", "not_live", "known_destroy", "temporary_escape", "conflicting_destructive_access"} {
		t.Run(name, func(t *testing.T) {
			path := "../../language/evt2g3b/invalid/" + name + ".concept"
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(path, string(source))
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) {
				t.Fatalf("expected checked source diagnostic: %v", err)
			}
			want := map[string]string{"const_store": "MUTATION_THROUGH_REF_CONST", "const_escalation": "REF_MUTABLE_FROM_CONST", "wrong_target": "REF_TYPE_MISMATCH", "local_escape": "REF_ESCAPE_LOCAL", "not_live": "CV4560", "known_destroy": "STORAGE_REFERENCE_AFTER_DESTROY", "temporary_escape": "REF_REQUIRES_PLACE", "conflicting_destructive_access": "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW"}[name]
			if diagnostic.SemanticCategory() != want {
				t.Fatalf("want %s, got %s: %v", want, diagnostic.SemanticCategory(), err)
			}
			t.Logf("%s: %s", diagnostic.Code, diagnostic.SemanticCategory())
		})
	}
}

func TestEVT2g3bBridgeAdmission(t *testing.T) {
	path := "../../language/evt2g3b/valid/references.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	clone := func() BorrowModule {
		data, _ := json.Marshal(machine.Borrow)
		var out BorrowModule
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	ordinal := func(name string) int {
		for i, f := range machine.Borrow.Objects.Functions {
			if f.Name == name {
				return i
			}
		}
		t.Fatalf("missing %s", name)
		return -1
	}
	local := ordinal("Local")
	read := ordinal("Read")
	element := ordinal("Element")
	early := ordinal("Early")
	type specimen struct {
		name, want string
		data       []byte
	}
	cases := []specimen{}
	add := func(name, want string, change func(*BorrowModule)) {
		next := clone()
		change(&next)
		if err := VerifyMachineIR(MachineModule{Borrow: &next}); err == nil {
			t.Fatalf("Go accepted %s", name)
		}
		data, err := encodeBorrowMachineBridge(next)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, specimen{name, want, data})
	}
	add("target", "CMIRAMD6_REFERENCE_TARGET_TYPE", func(next *BorrowModule) { next.Authority[local].References[0].Target = 255 })
	add("semantic_target", "CMIRAMD6_REFERENCE_TARGET_TYPE", func(next *BorrowModule) { next.Authority[local].References[1].SemanticTarget = strings.Repeat("f", 64) })
	add("semantic_call_target", "CMIRAMD6_REFERENCE_CALL_SIGNATURE", func(next *BorrowModule) { next.Authority[read].References[0].SemanticTarget = strings.Repeat("f", 64) })
	add("root", "CMIRAMD6_REFERENCE_ROOT", func(next *BorrowModule) { next.Authority[local].References[0].Root = 63 })
	add("layout", "CMIRAMD6_REFERENCE_TARGET_TYPE", func(next *BorrowModule) { next.Authority[local].References[0].LayoutIdentity = "stale" })
	add("lifetime", "CMIRAMD6_REFERENCE_LIFETIME", func(next *BorrowModule) { next.Authority[local].References[0].Lifetime = 127 })
	add("scope_parent", "CMIRAMD6_REFERENCE_LIFETIME", func(next *BorrowModule) { next.Authority[local].Scopes[1].Parent = 1 })
	add("provenance", "CMIRAMD6_REFERENCE_PROVENANCE", func(next *BorrowModule) { next.Authority[local].References[1].Provenance = "other-object" })
	add("const_upgrade", "CMIRAMD6_MUTABLE_AUTHORITY_REQUIRED", func(next *BorrowModule) { next.Authority[read].References[1].Mutable = true })
	add("const_parameter", "CMIRAMD6_MUTABLE_AUTHORITY_REQUIRED", func(next *BorrowModule) {
		next.Authority[read].References[0].Mutable = true
		for i := range next.Authority[read].References {
			next.Authority[read].References[i].Mutable = true
		}
	})
	add("field_offset", "CMIRAMD6_LAYOUT_GEOMETRY", func(next *BorrowModule) {
		for b := range next.Objects.Functions[local].Blocks {
			for i, in := range next.Objects.Functions[local].Blocks[b].Instructions {
				if in.Op == "field_address" {
					next.Objects.Functions[local].Blocks[b].Instructions[i].Offset++
					return
				}
			}
		}
	})
	add("stride", "CMIRAMD6_LAYOUT_GEOMETRY", func(next *BorrowModule) {
		for b := range next.Objects.Functions[element].Blocks {
			for i, in := range next.Objects.Functions[element].Blocks[b].Instructions {
				if in.Op == "index_address" {
					next.Objects.Functions[element].Blocks[b].Instructions[i].Stride++
					return
				}
			}
		}
	})
	add("not_live", "CMIRAMD6_REFERENCE_TARGET_NOT_LIVE", func(next *BorrowModule) {
		for i, in := range next.Objects.Functions[local].Blocks[0].Instructions {
			if in.Op == "initialize_slot" {
				next.Objects.Functions[local].Blocks[0].Instructions[i].Op = "store_slot"
				return
			}
		}
	})
	add("call_type", "CMIRAMD6_REFERENCE_CALL_SIGNATURE", func(next *BorrowModule) {
		i := ordinal("Calls")
		for p, in := range next.Objects.Functions[i].Blocks[0].Instructions {
			if in.Op == "call" {
				next.Objects.Functions[i].Blocks[0].Instructions[p].Calls[0].Target = "missing"
				return
			}
		}
	})
	add("reference_result", "CMIRAMD6_VALUE_TYPE", func(next *BorrowModule) {
		next.Objects.Functions[local].Blocks[0].Term.Value = next.Authority[local].References[1].Value
	})
	add("site_missing", "CMIRAMD6_MALFORMED", func(next *BorrowModule) {
		next.Authority[local].Sites = next.Authority[local].Sites[:len(next.Authority[local].Sites)-1]
	})
	add("scope_use", "CMIRAMD6_REFERENCE_LIFETIME", func(next *BorrowModule) {
		for i, site := range next.Authority[early].Sites {
			if site.Block == 1 && next.Objects.Functions[early].Blocks[site.Block].Instructions[site.Instruction].Op == "call" {
				next.Authority[early].Sites[i].Scope = 0
				return
			}
		}
	})
	add("reference_count", "CMIRAMD6_CAPACITY_EXCEEDED", func(next *BorrowModule) {
		for len(next.Authority[local].References) < 513 {
			next.Authority[local].References = append(next.Authority[local].References, next.Authority[local].References[0])
		}
	})
	add("scope_count", "CMIRAMD6_CAPACITY_EXCEEDED", func(next *BorrowModule) {
		for len(next.Authority[local].Scopes) < 129 {
			next.Authority[local].Scopes = append(next.Authority[local].Scopes, next.Authority[local].Scopes[0])
		}
	})
	// Invalid tags/counts cannot be produced by the generated checked writer.
	var fields []machineBridgeWireField
	w := &machineBridgeWriter{trace: &fields}
	if err := w.writeWireBorrowModule(*machine.Borrow); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []struct {
		name, record, field, want string
		value                     uint32
	}{{"root_tag", "WireBorrowReference", "RootKind", "CMIRAMD6_MALFORMED", 99}, {"authority_bool", "WireBorrowReference", "Mutable", "CMIRAMD6_MALFORMED", 2}, {"escape_tag", "WireBorrowReference", "Escape", "CMIRAMD6_MALFORMED", 1}, {"address_space", "WireBorrowReference", "Space", "CMIRAMD6_MALFORMED", 1}, {"reference_capacity", "WireBorrowFunction", "References", "CMIRAMD6_MALFORMED", 1048577}} {
		data := bytes.Clone(valid)
		offset := BorrowMachineBridgeHeaderSize + bridgeFieldOffset(t, fields, raw.record, raw.field)
		binary.LittleEndian.PutUint32(data[offset:], raw.value)
		if _, err := DecodeMachineBridge(data); err == nil {
			t.Fatalf("Go accepted %s", raw.name)
		}
		cases = append(cases, specimen{raw.name, raw.want, data})
	}
	for _, h := range []struct {
		name, want string
		offset     int
	}{{"version", "CMIRAMD6_VERSION_MISMATCH", 8}, {"schema", "CMIRAMD6_SCHEMA_HASH_MISMATCH", 12}} {
		data := bytes.Clone(valid)
		data[h.offset] ^= 1
		cases = append(cases, specimen{h.name, h.want, data})
	}
	cases = append(cases, specimen{"truncated", "CMIRAMD6_MALFORMED", valid[:len(valid)-1]})
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char input[262144],out[262144];static concept_native_image image,before;int main(void){FILE *file;size_t n;concept_span_byte dst={out,sizeof out};\n")
	dir := t.TempDir()
	good := filepath.Join(dir, "valid.cmir")
	if err := os.WriteFile(good, valid, 0600); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&c, "file=fopen(%q,\"rb\");if(!file)return 90;n=fread(input,1,sizeof input,file);fclose(file);concept_readonly_span_byte src={input,n};for(int run=0;run<100;run++){concept_result_int_borrow_error r=F(validated_borrow_bridge_round_trip)(src,dst);if(r.tag){fprintf(stderr,\"valid: %%s\\n\",F(describe_borrow_error)(r.payload.error.error));return 91;}if(r.payload.ok.value!=(int)n||memcmp(input,out,n))return 92;}\n", filepath.ToSlash(good))
	for i, tc := range cases {
		file := filepath.Join(dir, tc.name+".cmir")
		if err := os.WriteFile(file, tc.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&c, "file=fopen(%q,\"rb\");if(!file)return 93;n=fread(input,1,sizeof input,file);fclose(file);src=(concept_readonly_span_byte){input,n};memset(out,0xa5,sizeof out);memset(&image,0x5a,sizeof image);before=image;concept_result_int_borrow_error r%d=F(validated_borrow_bridge_round_trip)(src,dst);if(!r%d.tag||strcmp(F(describe_borrow_error)(r%d.payload.error.error),%q)){fprintf(stderr,\"%s: %%s\\n\",r%d.tag?F(describe_borrow_error)(r%d.payload.error.error):\"ACCEPTED\");return %d;}concept_result_int_native_error emitted%d=F(emit_module)(src,dst,&image);if(!emitted%d.tag||memcmp(&before,&image,sizeof image))return 94;for(size_t j=0;j<sizeof out;j++){if(out[j]!=0xa5)return 95;}\n", filepath.ToSlash(file), i, i, i, tc.want, tc.name, i, i, i+1, i, i)
	}
	abiSource := `module BorrowABI; profile Core; struct R{int value;} int Read(ref const R r){return r.value;} R Identity(R value){return value;}`
	abiModule, err := Parse("borrowabi.concept", abiSource)
	if err != nil {
		t.Fatal(err)
	}
	abiMachine, err := GenerateMachineIR(abiModule)
	if err != nil {
		t.Fatal(err)
	}
	abiData, err := EncodeMachineBridge(abiMachine)
	if err != nil {
		t.Fatal(err)
	}
	abiPath := filepath.Join(dir, "aggregate_abi.cmir")
	if err := os.WriteFile(abiPath, abiData, 0644); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&c, "file=fopen(%q,\"rb\");if(!file)return 96;n=fread(input,1,sizeof input,file);fclose(file);src=(concept_readonly_span_byte){input,n};concept_result_void_borrow_error admitted=F(validate_borrow_machine_bridge)(src);if(admitted.tag)return 97;concept_result_int_native_error abi=F(emit_module)(src,dst,&image);if(abi.tag||!image.finalized)return 98;\n", filepath.ToSlash(abiPath))
	fmt.Fprintf(&c, "puts(\"100 artifact-only roundtrips and %d malformed reference cases passed; checked aggregate ABI now emits\");return 0;}\n", len(cases))
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "reference_admission.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g3bNativeReferences(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("actual Win64 execution requires Windows AMD64")
	}
	path := "../../language/evt2g3b/valid/references.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	data.WriteString("static const unsigned char artifact[]={")
	for _, b := range artifact {
		fmt.Fprintf(&data, "%d,", b)
	}
	data.WriteString("};\n")
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", f.Name, i)
	}
	dir := t.TempDir()
	dump := filepath.ToSlash(filepath.Join(dir, "native.bin"))
	harness := strings.Replace(referenceNativeHarness, "/* ARTIFACT */", data.String(), 1)
	harness = strings.Replace(harness, "/* IMAGE_DUMP */", fmt.Sprintf("%q", dump), 1)
	var first []byte
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			runFoundationNativeHarnessOutput(t, outputs, "references_artifact.c", "#define REFERENCE_NATIVE_ONLY 1\n"+harness, "-pedantic-errors", "-O2")
			actual, err := os.ReadFile(filepath.FromSlash(dump))
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = actual
			} else if !bytes.Equal(first, actual) {
				t.Fatal("Normal/Verify native images differ")
			}
			policy := ConservativeCompilationPolicy()
			if verify {
				policy = VerifyCompilationPolicy()
			}
			oracle, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			for name, body := range oracle {
				if strings.HasSuffix(name, ".generated.c") {
					outputs["oracle.c"] = body
				} else {
					outputs[name] = body
				}
			}
			runFoundationNativeHarnessOutput(t, outputs, "references_oracle.c", harness, "-pedantic-errors", "-O2")
		})
	}
}

func TestEVT2g3bOriginalBoundaryNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Win64 native execution requires Windows AMD64")
	}
	path := "../../language/evt2g3/boundaries/source_references.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	ordinal := -1
	for i, f := range machine.Borrow.Objects.Functions {
		if f.Name == "Main" {
			ordinal = i
		}
	}
	if ordinal < 0 {
		t.Fatal("Main missing")
	}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <windows.h>\n#include <stdio.h>\n#include <string.h>\nstatic const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};\n#define F(n) concept_standard__backend__amd64_##n\nint main(void){unsigned char bytes[4096];concept_native_image image;concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(r.tag)return 1;unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 2;memcpy(memory,bytes,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 3;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);typedef double(*Main)(void);Main fn=NULL;")
	fmt.Fprintf(&c, "void *p=memory+image.symbols.data[%d].offset;memcpy(&fn,&p,sizeof fn);double actual=fn();VirtualFree(memory,0,MEM_RELEASE);if(actual!=9.0)return 4;printf(\"original ref Position boundary returns 9: functions=%%d image=%%d bytes\\n\",image.symbolCount,image.byteCount);return 0;}\n", ordinal)
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "original_borrow.c", c.String(), "-pedantic-errors", "-O2")
	}
}

const referenceNativeHarness = `
#include "amd64.generated.h"
#ifndef REFERENCE_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
/* ARTIFACT */
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
typedef double(*Unary)(double);typedef double(*Element)(int,double);typedef double(*Early)(_Bool,double);typedef int(*Integer)(int);typedef float(*Float)(float);
typedef void(*Add)(int32_t*,int32_t);typedef int(*Stack)(int,int,int,int,int32_t*,int,int,double*);
typedef double(*Nullary)(void);
#define GETTER(t) static t Get##t(unsigned char *memory,int ordinal,concept_native_image *image){t fn=NULL;void *p=memory+image->symbols.data[ordinal].offset;memcpy(&fn,&p,sizeof fn);return fn;}
GETTER(Unary) GETTER(Element) GETTER(Early) GETTER(Integer) GETTER(Float) GETTER(Add) GETTER(Stack)
GETTER(Nullary)
#define FN(n,t) Get##t(memory,ORD_##n,&image)
static unsigned char code[262144],again[262144];static concept_native_image image,other;
static int plan_text(concept_readonly_span_byte input,int ordinal,char *text){concept_backend_function f;concept_object_placement p;concept_allocation a;concept_finalized_frame frame;if(F(plan_object_native_function)(input,ordinal,&f,&p,&a,&frame).tag)return 0;int n=sprintf(text,"frame=%d slots=%d regs=%d",frame.byteCount,frame.slotCount,f.virtualRegCount);for(int s=0;s<frame.slotCount;s++){concept_frame_slot v=frame.slots.data[s];n+=sprintf(text+n," s%d:%d:%d:%d:%d",v.category.tag,v.identity,v.offset,v.size,v.alignment);}for(int v=0;v<f.virtualRegCount;v++){concept_scalar_location x=a.scalarLocations.data[v];n+=sprintf(text+n," v%d:%d:%d:%d:%d:%d",v,x.kind.tag,x.gpr.tag,x.xmm.tag,a.start.data[v],a.end.data[v]);}return 1;}
int main(int argc,char **argv){
 concept_readonly_span_byte input={artifact,sizeof artifact};concept_span_byte output={code,sizeof code};
 concept_result_void_borrow_error checked=F(validate_borrow_machine_bridge)(input);if(checked.tag){fprintf(stderr,"%s\n",F(describe_borrow_error)(checked.payload.error.error));return 1;}
 clock_t start=clock();concept_result_int_native_error result=F(emit_module)(input,output,&image);if(result.tag){fprintf(stderr,"%s\n",F(describe_native_error)(result.payload.error.error));return 2;}REQUIRE(image.finalized);
 printf("combined planning/encoding %.3f ms\n",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 FILE *dump=fopen(/* IMAGE_DUMP */,"wb");REQUIRE(dump);REQUIRE(fwrite(code,1,image.byteCount,dump)==(size_t)image.byteCount);fclose(dump);
 unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,code,image.byteCount);DWORD old=0;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);FN(Element,Element)(argv[1][0]=='-'?-1:3,1.0);return 99;}
 REQUIRE(FN(Local,Unary)(4.0)==9.0);REQUIRE(FN(Fields,Unary)(4.0)==11.0);REQUIRE(FN(IntegerField,Integer)(37)==40);REQUIRE(FN(FloatField,Float)(2.5f)==7.0f);
 REQUIRE(FN(NestedField,Unary)(4.0)==9.0);REQUIRE(FN(Element,Element)(1,4.0)==7.0);REQUIRE(FN(Aliases,Unary)(4.0)==15.0);REQUIRE(FN(CopyVersusReference,Unary)(4.0)==13.0);
 REQUIRE(FN(Calls,Unary)(4.0)==9.0);REQUIRE(FN(AcrossCall,Unary)(4.0)==10.0);REQUIRE(FN(Early,Early)(1,4.0)==7.0&&FN(Early,Early)(0,4.0)==6.0);REQUIRE(FN(StackCaller,Integer)(37)==40);REQUIRE(FN(Immutable,Unary)(4.0)==6.0);
 REQUIRE(FN(WholeObject,Unary)(4.0)==15.0);REQUIRE(FN(ScopeStable,Unary)(4.0)==9.0);
 REQUIRE(FN(SelectedElement,Nullary)()==6.0);
 /* Independently compiled C calls exercise native pointer parameters, including
    pointer positions 4 and 7. This does not qualify aggregate external ABI. */
 int32_t count=37;double number=4.0;FN(Add,Add)(&count,3);REQUIRE(count==40);REQUIRE(FN(StackScalar,Stack)(2,3,4,5,&count,6,7,&number)==42&&count==42&&number==9.0);
#ifndef REFERENCE_NATIVE_ONLY
 for(int run=0;run<100;run++){double x=(double)run/8.0-5.0;int index=run%3;
 REQUIRE(FN(WholeObject,Unary)(x)==concept_evt2g3b__references_whole_object(x));REQUIRE(FN(ScopeStable,Unary)(x)==concept_evt2g3b__references_scope_stable(x));
 REQUIRE(FN(Local,Unary)(x)==concept_evt2g3b__references_local(x));REQUIRE(FN(Fields,Unary)(x)==concept_evt2g3b__references_fields(x));REQUIRE(FN(NestedField,Unary)(x)==concept_evt2g3b__references_nested_field(x));REQUIRE(FN(Element,Element)(index,x)==concept_evt2g3b__references_element(index,x));REQUIRE(FN(Aliases,Unary)(x)==concept_evt2g3b__references_aliases(x));REQUIRE(FN(CopyVersusReference,Unary)(x)==concept_evt2g3b__references_copy_versus_reference(x));REQUIRE(FN(Calls,Unary)(x)==concept_evt2g3b__references_calls(x));REQUIRE(FN(AcrossCall,Unary)(x)==concept_evt2g3b__references_across_call(x));REQUIRE(FN(Early,Early)(run%2,x)==concept_evt2g3b__references_early(run%2,x));REQUIRE(FN(StackCaller,Integer)(run)==concept_evt2g3b__references_stack_caller(run));}
#endif
 start=clock();for(int run=0;run<10000000;run++){count=0;FN(Add,Add)(&count,1);REQUIRE(count==1);}printf("10000000 native reference calls including checked body %.3f ms\n",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 VirtualFree(memory,0,MEM_RELEASE);
 start=clock();for(int ordinal=0;ordinal<image.symbolCount;ordinal++){concept_backend_function f;concept_object_placement p;concept_allocation a;concept_finalized_frame frame;REQUIRE(!F(plan_object_native_function)(input,ordinal,&f,&p,&a,&frame).tag);printf("frame %.*s bytes=%d slots=%d regs=%d code=%d\n",f.name.length,(char*)artifact+f.name.offset,frame.byteCount,f.slotCount,f.virtualRegCount,image.symbols.data[ordinal].size);}
 printf("all-function validation/planning %.3f ms\n",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 char before[32768],after[32768];REQUIRE(plan_text(input,ORD_Early,before));
 for(int run=0;run<100;run++){concept_span_byte dst={again,sizeof again};result=F(emit_module)(input,dst,&other);REQUIRE(!result.tag&&other.byteCount==image.byteCount&&!memcmp(code,again,image.byteCount));REQUIRE(plan_text(input,ORD_Early,after)&&!strcmp(before,after));}
 for(int bad=0;bad<2;bad++){char path[MAX_PATH],command[MAX_PATH+32];REQUIRE(GetModuleFileNameA(NULL,path,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %s",path,bad?"-1":"3");STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;REQUIRE(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));DWORD waited=WaitForSingleObject(process.hProcess,30000);if(waited!=WAIT_OBJECT_0){TerminateProcess(process.hProcess,99);WaitForSingleObject(process.hProcess,1000);}DWORD status=0;REQUIRE(GetExitCodeProcess(process.hProcess,&status)&&waited==WAIT_OBJECT_0&&status==0xc000001dUL);CloseHandle(process.hThread);CloseHandle(process.hProcess);}
 printf("native references functions=%d image=%d: actual execution, ABI, bounds, 100 bytes/plans passed\n",image.symbolCount,image.byteCount);return 0;
}
`
