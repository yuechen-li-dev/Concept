package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2x6NativeCTraceParity(t *testing.T) {
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) { runPushdownNativeTraceParity(t, verify) })
	}
}

func runPushdownNativeTraceParity(t *testing.T, verify bool) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("executable-memory qualification requires Windows AMD64")
	}
	module, source := pushdownFixture(t)
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	layout := lir.Functions[1].Activation.Layout
	if layout.Size != 140 || layout.SlotSize != 16 || layout.SlotsOffset != 12 || layout.MaxFrameSize != 12 || layout.Alignment != 4 {
		t.Fatalf("fixture layout drift: %+v", layout)
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range machine.Functions {
		t.Logf("%s: %d virtual registers, %d blocks", fn.Name, len(fn.VRegs), len(fn.Blocks))
	}
	for i := 0; i < 100; i++ {
		next, err := GenerateLIR(module)
		if err != nil || !reflect.DeepEqual(next, lir) || next.String() != lir.String() {
			t.Fatalf("LIR run %d: %v", i, err)
		}
		nm, err := LowerLirToAmd64Machine(next)
		if err != nil || nm.String() != machine.String() {
			t.Fatalf("MachineIR run %d: %v", i, err)
		}
		nb, err := EncodeMachineBridge(nm)
		if err != nil || !bytes.Equal(nb, bridge) {
			t.Fatalf("CMIR run %d: %v", i, err)
		}
	}
	backendPath := filepath.Join("..", "..", "libraries", "Standard", "Backend", "AMD64.concept")
	backendSource, err := os.ReadFile(backendPath)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := ParseWithBuiltSemanticModuleRoots(backendPath, string(backendSource), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	policy := ConservativeCompilationPolicy()
	if verify {
		policy = VerifyCompilationPolicy()
	}
	outputs, err := GenerateForTargetWithPolicy(backend, backendSource, GenericC11Target(), policy)
	if err != nil {
		t.Fatal(err)
	}
	oracleSource := append(append([]byte(nil), source...), []byte("\nint Main() { instance Worker a(0); Step(a, Parent); return 0; }\n")...)
	oracleModule, err := Parse("pushdown.concept", string(oracleSource))
	if err != nil {
		t.Fatal(err)
	}
	oracleOutputs, err := GenerateForTargetWithPolicy(oracleModule, oracleSource, GenericC11Target(), policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pushdown.generated.h", "pushdown.generated.c"} {
		outputs[key] = oracleOutputs[key]
	}
	// Observe the existing C boundary, without implementing machine semantics.
	body := "static int oracle_yielded;\n" + strings.ReplaceAll(string(outputs["pushdown.generated.c"]), "return; /* CONCEPT_STEP_YIELDED", "oracle_yielded=1; return; /* CONCEPT_STEP_YIELDED")
	outputs["oracle_runtime.inc"] = []byte(body)
	delete(outputs, "pushdown.generated.c")
	pairPath := filepath.Join("..", "..", "language", "evt2", "machines", "parent_child.concept")
	pairSource, err := os.ReadFile(pairPath)
	if err != nil {
		t.Fatal(err)
	}
	pairModule, err := Parse("pair.concept", string(pairSource))
	if err != nil {
		t.Fatal(err)
	}
	pairLIR, err := GenerateLIR(pairModule)
	if err != nil {
		t.Fatal(err)
	}
	if pairLIR.Functions[1].Activation.Layout.Size != 108 || pairLIR.Functions[1].Activation.Layout.SlotSize != 12 {
		t.Fatal("source stride-12 fixture layout drift")
	}
	pairMachine, err := LowerLirToAmd64Machine(pairLIR)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pairMachine.String(), "IMUL") {
		t.Fatal("source stride 12 did not use shared MachineIR legalization")
	}
	pairWire, err := EncodeMachineBridge(pairMachine)
	if err != nil {
		t.Fatal(err)
	}
	pairOracleSource := append(append([]byte(nil), pairSource...), []byte("\nint Main() { instance Pair a(0); Step(a, Parent); return 0; }\n")...)
	pairOracleModule, err := Parse("pair.concept", string(pairOracleSource))
	if err != nil {
		t.Fatal(err)
	}
	pairOutputs, err := GenerateForTargetWithPolicy(pairOracleModule, pairOracleSource, GenericC11Target(), policy)
	if err != nil {
		t.Fatal(err)
	}
	outputs["pair.generated.h"] = pairOutputs["pair.generated.h"]
	outputs["pair_runtime.inc"] = []byte(strings.ReplaceAll(string(pairOutputs["pair.generated.c"]), "return; /* CONCEPT_STEP_YIELDED", "oracle_yielded=1; return; /* CONCEPT_STEP_YIELDED"))
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "pushdown.cmir")
	if err := os.WriteFile(artifact, bridge, 0644); err != nil {
		t.Fatal(err)
	}
	pairArtifact := filepath.Join(dir, "pair.cmir")
	if err := os.WriteFile(pairArtifact, pairWire, 0644); err != nil {
		t.Fatal(err)
	}
	variantArtifact := func(name, sourceText string) string {
		t.Helper()
		vm, err := Parse(name+".concept", sourceText)
		if err != nil {
			t.Fatal(err)
		}
		mm, err := GenerateMachineIR(vm)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := EncodeMachineBridge(mm)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name+".cmir")
		if err := os.WriteFile(path, wire, 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	rootPop := variantArtifact("root-pop", strings.ReplaceAll(string(source), "machine.preserved; complete;", "machine.preserved; pop;"))
	initFailure := variantArtifact("init-failure", strings.ReplaceAll(string(source), "int count = 4;", "int count = value + 1;"))
	harness := filepath.Join(dir, "pushdown_harness.c")
	if err := os.WriteFile(harness, []byte(pushdownNativeHarness), 0644); err != nil {
		t.Fatal(err)
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Fatal("native qualification needs a C compiler")
	}
	executable := filepath.Join(dir, "pushdown-native.exe")
	if out, err := nativeCommand(t, compiler, "-std=c11", "-I", dir, filepath.Join(dir, "amd64.generated.c"), harness, "-o", executable).CombinedOutput(); err != nil {
		t.Fatalf("native build: %v\n%s", err, out)
	}
	if out, err := nativeCommand(t, executable, artifact, "trace").CombinedOutput(); err != nil {
		t.Fatalf("native parity: %v\n%s", err, out)
	} else {
		t.Logf("C/native trace (100 identical runs; two interleaved instances):\n%s", out)
	}
	if out, err := nativeCommand(t, executable, rootPop, "root-pop").CombinedOutput(); err != nil {
		t.Fatalf("native root pop: %v\n%s", err, out)
	}
	if out, err := nativeCommand(t, executable, pairArtifact, "pair-trace").CombinedOutput(); err != nil {
		t.Fatalf("source stride-12 C/native parity: %v\n%s", err, out)
	} else {
		t.Logf("source stride-12 C/native trace:\n%s", out)
	}
	for _, scenario := range []string{"depth-zero", "depth-over", "tag", "state", "nested-tag", "nested-state", "overflow", "init-failure"} {
		t.Run(scenario, func(t *testing.T) {
			snapshot := filepath.Join(dir, scenario+".bin")
			input := artifact
			if scenario == "init-failure" {
				input = initFailure
			}
			cmd := nativeCommand(t, executable, input, scenario, snapshot)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("corruption unexpectedly executed: %s", out)
			}
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 77 {
				t.Fatalf("expected trapped snapshot: %v\n%s", err, out)
			}
			data, err := os.ReadFile(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != 344 || !bytes.Equal(data[:172], data[172:]) {
				t.Fatalf("trap mutated frame or sentinels: %x", data)
			}
		})
	}
}

const pushdownNativeHarness = `#include "amd64.generated.h"
#include "oracle_runtime.inc"
#define concept_panic pair_concept_panic
#define concept_abort_invalid_automata_state pair_abort_invalid_automata_state
#define concept_abort_automata_stack pair_abort_automata_stack
#define concept_abort_automata_completion pair_abort_automata_completion
#include "pair_runtime.inc"
#undef concept_panic
#undef concept_abort_invalid_automata_state
#undef concept_abort_automata_stack
#undef concept_abort_automata_completion
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
typedef struct { unsigned char before[16], frame[140], after[16]; } guarded;
typedef void (*init_fn)(void*, int32_t);
typedef uint32_t (*step_fn)(void*);
static guarded failed, prior;
static const char* snapshot;
static LONG CALLBACK trap_snapshot(EXCEPTION_POINTERS* p) {
  if (p->ExceptionRecord->ExceptionCode != EXCEPTION_ILLEGAL_INSTRUCTION) return EXCEPTION_CONTINUE_SEARCH;
  FILE* f=fopen(snapshot,"wb"); if (!f) ExitProcess(78);
  fwrite(&prior,1,sizeof prior,f); fwrite(&failed,1,sizeof failed,f); fclose(f); ExitProcess(77);
  return EXCEPTION_CONTINUE_SEARCH;
}
static uint32_t word(const unsigned char* f,int offset) { uint32_t v;memcpy(&v,f+offset,4);return v; }
static void put(unsigned char* f,int offset,uint32_t v) {memcpy(f+offset,&v,4);}
static int sentinels(const guarded* g) {for(int i=0;i<16;i++) if(g->before[i]!=0xA5||g->after[i]!=0xA5)return 0;return 1;}
static void* make_code(concept_readonly_span_byte bridge,int ordinal) {
  unsigned char code[32768]={0};concept_span_byte output={code,sizeof code};
  concept_result_int_backend_error result=concept_standard__backend__amd64_emit_function(bridge,ordinal,output);
  if(result.tag!=0){fprintf(stderr,"backend function %d error %u\n",ordinal,result.payload.error.error.tag);
    concept_backend_function f=concept_standard__backend__amd64_empty_function();
    concept_standard__backend__amd64_decode_function(bridge,ordinal,&f);
    concept_allocation a=concept_standard__backend__amd64_empty_allocation();
    concept_standard__backend__amd64_layout_lowered_blocks(&f);
    concept_standard__backend__amd64_compute_intervals(&f,&a);
    for(int pos=0;pos<f.instructionCount;pos++) {int live=0;for(int id=0;id<f.virtualRegCount;id++)if(a.start.data[id]<=pos&&a.end.data[id]>pos)live++;if(live>5){fprintf(stderr,"pressure at %d:",pos);for(int id=0;id<f.virtualRegCount;id++)if(a.start.data[id]<=pos&&a.end.data[id]>pos)fprintf(stderr," v%d[%d,%d]",id,a.start.data[id],a.end.data[id]);fprintf(stderr,"\n");break;}}
    return NULL;}
  concept_backend_function function=concept_standard__backend__amd64_empty_function();
  if(concept_standard__backend__amd64_decode_function(bridge,ordinal,&function).tag!=0||concept_standard__backend__amd64_layout_lowered_blocks(&function).tag!=0)return NULL;
  concept_allocation original=concept_standard__backend__amd64_empty_allocation();
  if(concept_standard__backend__amd64_allocate_registers(&function,&original).tag!=0)return NULL;
  for(int i=0;i<100;i++) {unsigned char next[32768]={0};concept_span_byte dst={next,sizeof next};concept_result_int_backend_error again=concept_standard__backend__amd64_emit_function(bridge,ordinal,dst);if(again.tag!=0||again.payload.ok.value!=result.payload.ok.value||memcmp(code,next,(size_t)result.payload.ok.value))return NULL;
    concept_allocation assignment=concept_standard__backend__amd64_empty_allocation();if(concept_standard__backend__amd64_allocate_registers(&function,&assignment).tag!=0)return NULL;
    for(int id=0;id<function.virtualRegCount;id++)if(assignment.physical.data[id].tag!=original.physical.data[id].tag||assignment.assigned.data[id]!=original.assigned.data[id])return NULL;
  }
  void* p=VirtualAlloc(NULL,(size_t)result.payload.ok.value,MEM_RESERVE|MEM_COMMIT,PAGE_READWRITE);if(!p)return NULL;
  memcpy(p,code,(size_t)result.payload.ok.value);DWORD old;
  if(!VirtualProtect(p,(size_t)result.payload.ok.value,PAGE_EXECUTE_READ,&old)||!FlushInstructionCache(GetCurrentProcess(),p,(size_t)result.payload.ok.value))return NULL;
  return p;
}
static int compare(int i,uint32_t result,guarded* g,concept_worker_instance* c,int print) {
  unsigned char* n=g->frame;uint32_t depth=word(n,0);
  if(!sentinels(g)||depth!=c->depth||n[4]!=c->completed||word(n,8)!=(uint32_t)c->shared.value||word(n,20)!=(uint32_t)c->parent_frames[0].preserved) return 1;
  if(result!=(uint32_t)(c->completed?2:oracle_yielded?1:0))return 2;
  int tag=-1,state=-1,count=-1;
  for(uint32_t slot=0;slot<depth;slot++) {
    int off=12+16*(int)slot;uint32_t t=word(n,off),s=word(n,off+4);
    if(t!=c->machine_tags[slot])return 3;
    if(t==0) {if(s!=c->parent_frames[slot].current_state||word(n,off+8)!=(uint32_t)c->parent_frames[slot].preserved)return 4;}
    else if(t==1) {if(s!=c->child_frames[slot].current_state||word(n,off+8)!=(uint32_t)c->child_frames[slot].count)return 5;}
    else if(t==2) {if(s!=c->grand_child_frames[slot].current_state||n[off+8]!=c->grand_child_frames[slot].touched||word(n,off+12)!=(uint32_t)c->grand_child_frames[slot].amount)return 6;}
    else return 7;
    if(slot==depth-1){tag=(int)t;state=(int)s;if(t==1)count=(int)word(n,off+8);}
  }
  if(print)printf("%d %u %u %d %d %d %d %d\n",i,result,depth,tag,state,(int)word(n,20),count,(int)word(n,8));
  return 0;
}
int main(int argc,char** argv) {
  if(argc<3)return 90;SetErrorMode(SEM_NOGPFAULTERRORBOX|SEM_FAILCRITICALERRORS);
  FILE* file=fopen(argv[1],"rb");if(!file)return 91;fseek(file,0,SEEK_END);long n=ftell(file);rewind(file);
  unsigned char* bytes=malloc((size_t)n);if(!bytes||fread(bytes,1,(size_t)n,file)!=(size_t)n)return 92;fclose(file);
  concept_readonly_span_byte bridge={bytes,(size_t)n};void* ic=make_code(bridge,0);void* sc=make_code(bridge,1);if(!ic||!sc)return 93;
  init_fn init=NULL;step_fn step=NULL;memcpy(&init,&ic,sizeof init);memcpy(&step,&sc,sizeof step);
  if(!strcmp(argv[2],"pair-trace")) {
    typedef struct {unsigned char before[16],frame[108],after[16];} pair_guarded;
    for(int repeat=0;repeat<100;repeat++) {
      pair_guarded g;memset(&g,0xA5,sizeof g);init(g.frame,0);concept_pair_instance c={0};concept_pair_init(&c,0);
      const int depths[]={2,2,1,0,0}, values[]={0,5,11,18,18};
      for(int i=0;i<5;i++) {
        oracle_yielded=0;concept_pair_step_top(&c);uint32_t result=step(g.frame);
        if(word(g.frame,0)!=c.depth||word(g.frame,0)!=(uint32_t)depths[i]||g.frame[4]!=c.completed||word(g.frame,8)!=(uint32_t)c.shared.value||word(g.frame,8)!=(uint32_t)values[i]||result!=(uint32_t)(c.completed?2:oracle_yielded?1:0))return 130+i;
        for(int j=0;j<16;j++)if(g.before[j]!=0xA5||g.after[j]!=0xA5)return 140;
        for(int j=0;j<c.depth;j++) {
          int off=12+j*12;uint32_t tag=word(g.frame,off);if(tag!=c.machine_tags[j])return 141;
          if(tag==0) {if(word(g.frame,off+4)!=c.parent_frames[j].current_state||word(g.frame,off+8)!=(uint32_t)c.parent_frames[j].preserved)return 142;}
          if(tag==1) {if(word(g.frame,off+4)!=c.child_frames[j].current_state||word(g.frame,off+8)!=(uint32_t)c.child_frames[j].count)return 143;}
        }
        if(repeat==0)printf("pair %d %u %u %d\n",i,result,word(g.frame,0),(int)word(g.frame,8));
      }
    }
    VirtualFree(ic,0,MEM_RELEASE);VirtualFree(sc,0,MEM_RELEASE);free(bytes);return 0;
  }
  if(strcmp(argv[2],"trace") && strcmp(argv[2],"root-pop")) {
    if(argc!=4)return 94;memset(&failed,0xA5,sizeof failed);init(failed.frame,0);
    if(!strcmp(argv[2],"depth-zero"))put(failed.frame,0,0);
    else if(!strcmp(argv[2],"depth-over"))put(failed.frame,0,9);
    else if(!strcmp(argv[2],"tag"))put(failed.frame,12,99);
    else if(!strcmp(argv[2],"state"))put(failed.frame,16,99);
    else if(!strcmp(argv[2],"nested-tag")||!strcmp(argv[2],"nested-state")){for(int i=0;i<4;i++)step(failed.frame);put(failed.frame,!strcmp(argv[2],"nested-tag")?44:48,99);}
    else if(!strcmp(argv[2],"overflow")){put(failed.frame,0,8);put(failed.frame,12+7*16,0);put(failed.frame,16+7*16,0);put(failed.frame,20+7*16,7);}
    else if(!strcmp(argv[2],"init-failure"))put(failed.frame,8,2147483647u);
    else return 95;
    prior=failed;snapshot=argv[3];AddVectoredExceptionHandler(1,trap_snapshot);step(failed.frame);return 96;
  }
  for(int repeat=0;repeat<100;repeat++) {
    guarded a,b;memset(&a,0xA5,sizeof a);memset(&b,0xA5,sizeof b);init(a.frame,0);init(b.frame,100);
    concept_worker_instance ca={0},cb={0};concept_worker_init(&ca,0);concept_worker_init(&cb,100);
    for(int i=0;i<14;i++) {
      oracle_yielded=0;concept_worker_step_top(&ca);uint32_t result=step(a.frame);int error=compare(i,result,&a,&ca,repeat==0);if(error){fprintf(stderr,"a step %d mismatch %d\n",i,error);return 100+error;}
      if(i%2==0) {oracle_yielded=0;concept_worker_step_top(&cb);result=step(b.frame);error=compare(i,result,&b,&cb,0);if(error)return 110+error;}
    }
    guarded completed=a;if(step(a.frame)!=2||memcmp(&a,&completed,sizeof a))return 120;
    if(word(b.frame,0)!=2||word(b.frame,8)!=116||word(a.frame,8)!=39)return 121;
  }
  VirtualFree(ic,0,MEM_RELEASE);VirtualFree(sc,0,MEM_RELEASE);free(bytes);return 0;
}
`
