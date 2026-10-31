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

// This executes residual runtime code. Verdict evaluation remains in Stage-0.
func TestEVT2h6ResidualNativeExecution(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64")
	}
	paths := []string{verdictProtocolPath, "../../language/evt1/semantic-vocabulary/valid/typed_verdict_values.concept", "../../language/evt2h6/valid/other_application.concept", "../../language/evt2h6/valid/artifact_consumer.concept"}
	var expected []byte
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var m Module
			if strings.Contains(path, "artifact_consumer") {
				_, producer := verdictSource(t, verdictProtocolPath)
				artifact := buildSemanticArtifact(t, verdictProtocolPath, string(producer), nil)
				m, err = ParseWithSemanticModules(path, string(source), map[string][]byte{"Research.Protocol": artifact})
			} else {
				m, err = Parse(path, string(source))
			}
			if err != nil {
				t.Fatal(err)
			}
			raw := verdictRuntime(t, m)
			dir := t.TempDir()
			input := filepath.Join(dir, "checked.cmir3")
			dump := filepath.Join(dir, "native.bin")
			if err := os.WriteFile(input, raw, 0600); err != nil {
				t.Fatal(err)
			}
			// Same checked declarations and runtime bodies with only the assertion
			// statement removed. No second embedded source program or evaluator.
			plain := m
			plain.Functions = append([]FunctionDecl(nil), m.Functions...)
			for i := range plain.Functions {
				body := *plain.Functions[i].Body
				body.Statements = nil
				for _, s := range plain.Functions[i].Body.Statements {
					if expr, ok := s.(*ExprStmt); ok {
						if call, ok := expr.Value.(*CallExpr); ok && call.ConceptGoal != "" {
							continue
						}
					}
					body.Statements = append(body.Statements, s)
				}
				plain.Functions[i].Body = &body
			}
			plainPath := filepath.Join(dir, "plain.cmir3")
			if err := os.WriteFile(plainPath, verdictRuntime(t, plain), 0600); err != nil {
				t.Fatal(err)
			}
			entry := 0
			name := "Use"
			for i, f := range m.Functions {
				if f.Name == "Main" {
					entry = i
					name = "Main"
				}
			}
			host := fmt.Sprintf(verdictResidualHost, filepath.ToSlash(input), filepath.ToSlash(plainPath), filepath.ToSlash(dump), entry, verdictTrials(), evt1FunctionSymbol(evt1SemanticSymbolBase(m), name))
			var first []byte
			for _, verify := range []bool{false, true} {
				outputs := backendTestOutputs(t, verify)
				t.Log(runFoundationNativeHarnessOutput(t, outputs, "verdict_residual.c", host, "-pedantic-errors", "-O2"))
				code, err := os.ReadFile(dump)
				if err != nil {
					t.Fatal(err)
				}
				if first == nil {
					first = code
				} else if !bytes.Equal(first, code) {
					t.Fatal("Normal/Verify residual image differs")
				}
				// The consumer has an extra imported ordinary Use function. Compare
				// complete image bytes only for the single-function specimens.
				if len(m.Functions) == 1 {
					if expected == nil {
						expected = code
					} else if !bytes.Equal(code, expected) {
						t.Fatal("proof cases or Normal/Verify changed residual code")
					}
				}
				t.Logf("verify=%t image=%d sha256=%x", verify, len(code), sha256.Sum256(code))
				want := "43a5857c93722cd54cfc1cdd4b33a39c67d0c81036396703dc76d97a83ead5d7"
				if len(m.Functions) == 2 {
					want = "4561bd362d67bd9cc489ab4a03c7ae7c30b0a6e1279e057a76fa7069da32cac6"
				}
				if got := fmt.Sprintf("%x", sha256.Sum256(code)); got != want {
					t.Fatalf("residual native pin changed: %s", got)
				}
				policy := ConservativeCompilationPolicy()
				if verify {
					policy = VerifyCompilationPolicy()
				}
				oracle, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
				if err != nil {
					t.Fatal(err)
				}
				for file, data := range oracle {
					if strings.HasSuffix(file, ".generated.c") {
						outputs["oracle.c"] = data
					} else {
						outputs[file] = data
					}
				}
				t.Log(runFoundationNativeHarnessOutput(t, outputs, "verdict_oracle.c", "#define VERDICT_ORACLE 1\n"+host, "-pedantic-errors", "-O2"))
			}
		})
	}
}

const verdictResidualHost = `#include "amd64.generated.h"
#include <windows.h>
#include <stdio.h>
#include <string.h>
#ifdef VERDICT_ORACLE
#include "oracle.c"
#endif
#define F(n) concept_standard__backend__amd64_##n
#define CHECK(c) do { if(!(c)){fprintf(stderr,"line %%d: %%s\n",__LINE__,#c);return __LINE__;} } while(0)
static unsigned char input[1000000],plain[1000000],code[65536],other[65536];
static concept_native_image image,baseline;
int main(void){
 FILE*f=fopen(%q,"rb");CHECK(f);size_t n=fread(input,1,sizeof input,f);fclose(f);
 f=fopen(%q,"rb");CHECK(f);size_t p=fread(plain,1,sizeof plain,f);fclose(f);
 CHECK(!F(emit_module)((concept_readonly_span_byte){input,n},(concept_span_byte){code,sizeof code},&image).tag);
 CHECK(!F(emit_module)((concept_readonly_span_byte){plain,p},(concept_span_byte){other,sizeof other},&baseline).tag);
 CHECK(image.byteCount==baseline.byteCount&&!memcmp(code,other,image.byteCount)&&image.fixupCount==0);
 f=fopen(%q,"wb");CHECK(f&&fwrite(code,1,image.byteCount,f)==(size_t)image.byteCount);fclose(f);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memory);memcpy(memory,code,image.byteCount);DWORD old;CHECK(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 typedef int(*Use)(void);Use use;void*entry=memory+image.symbols.data[%d].offset;memcpy(&use,&entry,sizeof use);CHECK(use()==7);
 for(int trial=0;trial<%d;trial++){CHECK(!F(emit_module)((concept_readonly_span_byte){input,n},(concept_span_byte){other,sizeof other},&baseline).tag);CHECK(!memcmp(&image,&baseline,sizeof image)&&!memcmp(code,other,image.byteCount));}
#ifdef VERDICT_ORACLE
 CHECK(use()==%s());puts("strict C11/native residual parity; Verdict evaluated at comptime only");
#else
 puts("source-free Cathedral residual execution; no predicate C body linked");
#endif
 printf("residual functions=%%d bytes=%%d fixups=%%d\n",image.symbolCount,image.byteCount,image.fixupCount);
 CHECK(VirtualFree(memory,0,MEM_RELEASE));return 0;
}`
