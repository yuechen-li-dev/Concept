package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"
)

const bridgeNativeByteHarness = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
int main(int argc,char**argv) {
 if(argc!=4)return 90;
 FILE*in=fopen(argv[1],"rb");if(!in)return 91;
 fseek(in,0,SEEK_END);long n=ftell(in);rewind(in);if(n<=0)return 92;
 unsigned char*data=malloc((size_t)n);if(!data || fread(data,1,(size_t)n,in)!=(size_t)n)return 93;fclose(in);
 FILE*out=fopen(argv[3],"wb");if(!out)return 94;
 concept_readonly_span_byte bridge={data,(size_t)n};
 for(int ordinal=0;ordinal<atoi(argv[2]);++ordinal) {
  unsigned char code[65536]={0};concept_span_byte dest={code,sizeof code};
  concept_result_int_backend_error result=concept_standard__backend__amd64_emit_function(bridge,ordinal,dest);
  if(result.tag!=0){fprintf(stderr,"ordinal=%d error=%u\n",ordinal,result.payload.error.error.tag);return 95;}
  for(int run=0;run<100;++run) {
   unsigned char next[65536]={0};concept_span_byte again={next,sizeof next};
   concept_result_int_backend_error r=concept_standard__backend__amd64_emit_function(bridge,ordinal,again);
   if(r.tag || r.payload.ok.value!=result.payload.ok.value || memcmp(code,next,(size_t)r.payload.ok.value))return 96;
  }
  uint32_t length=(uint32_t)result.payload.ok.value;
  if(fwrite(&length,4,1,out)!=1 || fwrite(code,1,length,out)!=length)return 97;
 }
 fclose(out);free(data);return 0;
}`

func bridgeNativeEmitter(t *testing.T, source []byte) string {
	t.Helper()
	path := "../../libraries/Standard/Backend/AMD64.concept"
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(dir, "bytes.c")
	if err := os.WriteFile(harness, []byte(bridgeNativeByteHarness), 0644); err != nil {
		t.Fatal(err)
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "bytes.exe")
	objects := nativeFixtureObjects(t, outputs, compiler, "-pedantic-errors")
	args := append([]string{"-std=c11", "-pedantic-errors", "-I", dir}, objects...)
	args = append(args, harness, "-o", exe)
	if out, err := nativeCommand(t, compiler, withHostLinkArgs(args...)...).CombinedOutput(); err != nil {
		t.Fatalf("native codec parity build: %v\n%s", err, out)
	}
	return exe
}
func bridgeNativeBytes(t *testing.T, exe string, data []byte, count int) ([]byte, time.Duration) {
	t.Helper()
	dir := t.TempDir()
	input := filepath.Join(dir, "input.cmir")
	output := filepath.Join(dir, "native.bin")
	if err := os.WriteFile(input, data, 0644); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if out, err := nativeCommand(t, exe, input, strconv.Itoa(count), output).CombinedOutput(); err != nil {
		t.Fatalf("native bytes: %v\n%s", err, out)
	}
	duration := time.Since(started)
	result, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return result, duration
}

// Frozen outputs were captured from the exact pre-migration manual backend
// after live shadow agreement. No manual codec or fallback ships in tests.
func TestR9a2BridgeNativeFrozenOracle(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Backend/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	derivedExe := bridgeNativeEmitter(t, source)
	corpus := bridgeParityCorpus(t)
	var names []string
	for name := range corpus {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			m := corpus[name]
			derived, err := EncodeMachineBridge(m)
			if err != nil {
				t.Fatal(err)
			}
			expected := bridgeFrozenOracle(t, name, ".amd64")
			actual, newTime := bridgeNativeBytes(t, derivedExe, derived, len(m.Functions))
			if !bytes.Equal(expected, actual) {
				t.Fatalf("native code byte mismatch %s", name)
			}
			t.Logf("%s: %d functions %d native bytes; 100 emissions/function=%s, frozen manual bytes identical", name, len(m.Functions), len(actual), newTime)
		})
	}
}
