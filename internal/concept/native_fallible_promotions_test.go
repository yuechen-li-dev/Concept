package concept

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keep the actual historical files executable when a formerly unsupported
// boundary is promoted. These are oracle/ABI assertions, not duplicate programs.
func TestEVT2h7HistoricalPromotions(t *testing.T) {
	requireNativeQualification(t)
	cases := []struct{ path, name, signature, checks string }{
		{"../../language/evt2h4/unsupported/escalation.concept", "Forward", "int(*Call)(NValue)", `CHECK(call((NValue){0,7})==concept_forward((concept_option_int){.tag=0,.payload.some.value=7}));`},
		{"../../language/evt2h4/unsupported/propagation.concept", "Forward", "NValue(*Call)(NValue)", `for(int tag=0;tag<2;tag++){NValue found=call((NValue){tag,7});concept_option_int oracle=concept_forward((concept_option_int){.tag=tag,.payload.some.value=7});CHECK(found.tag==oracle.tag);if(!tag)CHECK(found.value==oracle.payload.some.value);}`},
		{"../../language/evt2h5/unsupported/bang.concept", "Need", "int(*Call)(NValue)", `CHECK(call((NValue){0,7})==concept_need((concept_result_int_bool){.tag=0,.payload.ok.value=7}));`},
		{"../../language/evt2h5/unsupported/try.concept", "Need", "int(*Call)(void)", `CHECK(call()==0&&call()==concept_need());`},
	}
	for _, verify := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%t/%s", verify, tc.path), func(t *testing.T) {
				source, err := os.ReadFile(tc.path)
				if err != nil {
					t.Fatal(err)
				}
				m, err := Parse(tc.path, string(source))
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
				dir := t.TempDir()
				artifact := filepath.Join(dir, "input.cmir10")
				if err := os.WriteFile(artifact, data, 0600); err != nil {
					t.Fatal(err)
				}
				ordinal := -1
				for i, f := range machine.Fallible.Base.Base.Base.Objects.Functions {
					if f.Name == tc.name {
						ordinal = i
					}
				}
				if ordinal < 0 {
					t.Fatal(tc.name)
				}
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
				base := strings.TrimSuffix(filepath.Base(tc.path), ".concept")
				checks := strings.NewReplacer("concept_forward", "concept_"+base+"_forward", "concept_need", "concept_"+base+"_need").Replace(tc.checks)
				host := fmt.Sprintf(promotionHost, filepath.ToSlash(artifact), tc.signature, ordinal, checks)
				t.Log(runFoundationNativeHarnessOutput(t, outputs, "promotion.c", host, "-pedantic-errors", "-O2"))
			})
		}
	}
}

const promotionHost = `#include "amd64.generated.h"
#include "oracle.c"
#include <windows.h>
#include <stdio.h>
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
#define CHECK(c) do{if(!(c))return __LINE__;}while(0)
typedef struct{int tag,value;}NValue;
static unsigned char artifact[1000000],code[262144];static concept_native_image image;
int main(void){FILE*f=fopen("%s","rb");CHECK(f);size_t n=fread(artifact,1,sizeof artifact,f);fclose(f);CHECK(!F(emit_module)((concept_readonly_span_byte){artifact,n},(concept_span_byte){code,sizeof code},&image).tag);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memory);memcpy(memory,code,image.byteCount);DWORD old;CHECK(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 typedef %s;Call call;void*entry=memory+image.symbols.data[%d].offset;memcpy(&call,&entry,sizeof call);%s
 CHECK(VirtualFree(memory,0,MEM_RELEASE));puts("historical unsupported specimen: native/C11 promotion verified");return 0;}`
