package concept

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Execute the arm-dominance specimen too: one match contains an if, the other
// is inside an if. Runtime assertions are independent C/native differential
// probes over the same authoritative source used by the malformed CFG test.
func TestEVT2h3ConditionalNativeExecution(t *testing.T) {
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64")
	}
	path := "../../language/evt2h3/valid/arm_regions.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	var first []byte
	for run := 0; run < 100; run++ {
		fresh, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
		machine, err := GenerateMachineIR(fresh)
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeMachineBridge(machine)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = data
		} else if !bytes.Equal(first, data) {
			t.Fatal("conditional CFG/artifact differs")
		}
	}
	dir := t.TempDir()
	artifact := filepath.Join(dir, "regions.cmir9")
	if err := os.WriteFile(artifact, first, 0600); err != nil {
		t.Fatal(err)
	}
	host := strings.Replace(matchRegionsHost, "/* ARTIFACT */", fmt.Sprintf("#define ARTIFACT %q", filepath.ToSlash(artifact)), 1)
	var native []byte
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		oracle, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		outputs := backendTestOutputs(t, verify)
		for name, body := range oracle {
			if strings.HasSuffix(name, ".generated.c") {
				outputs["oracle.c"] = body
			} else {
				outputs[name] = body
			}
		}
		dump := filepath.Join(dir, fmt.Sprintf("regions-%t.bin", verify))
		t.Log(runFoundationNativeHarnessOutput(t, outputs, "match_regions.c", strings.Replace(host, "/* DUMP */", fmt.Sprintf("#define DUMP %q", filepath.ToSlash(dump)), 1), "-pedantic-errors", "-O2"))
		code, err := os.ReadFile(dump)
		if err != nil {
			t.Fatal(err)
		}
		if native == nil {
			native = code
		} else if !bytes.Equal(native, code) {
			t.Fatal("conditional Normal/Verify bytes differ")
		}
		t.Logf("conditional Verify=%t bytes=%d sha256=%x", verify, len(code), sha256.Sum256(code))
	}
}

const matchRegionsHost = `#include "amd64.generated.h"
#include "oracle.c"
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
/* ARTIFACT */
/* DUMP */
static unsigned char input[65536],code[65536],next[65536];static concept_native_image image,other;
int main(void){FILE*f=fopen(ARTIFACT,"rb");if(!f)return 1;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte src={input,n};
 if(F(validate_pattern_machine_bridge)(src).tag)return 2;
 concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(emitted.tag||!image.finalized||image.symbolCount!=2)return 3;
 for(int run=0;run<100;run++){concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){next,sizeof next},&other);if(r.tag||memcmp(&image,&other,sizeof image)||memcmp(code,next,image.byteCount))return 4;}
 f=fopen(DUMP,"wb");if(!f||fwrite(code,1,image.byteCount,f)!=(size_t)image.byteCount)return 5;fclose(f);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 6;memcpy(memory,code,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 7;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 typedef int(*Entry)(uint32_t,_Bool);Entry regions,conditional;void*p=memory+image.symbols.data[0].offset;memcpy(&regions,&p,sizeof regions);p=memory+image.symbols.data[1].offset;memcpy(&conditional,&p,sizeof conditional);
 for(int variant=0;variant<2;variant++)for(int condition=0;condition<2;condition++){concept_choice choice=variant?concept_choice_make_second():concept_choice_make_first();if(regions(variant,condition)!=concept_testing__match_regions_regions(choice,condition)||conditional(variant,condition)!=concept_testing__match_regions_conditional(choice,condition))return 8;}
 VirtualFree(memory,0,MEM_RELEASE);puts("match inside if / if inside arm: C11 parity and 100 fresh emissions passed");return 0;}`
