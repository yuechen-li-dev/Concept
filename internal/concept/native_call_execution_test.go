package concept

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2e5NativeInternalCalls(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Win64 executable-memory ABI qualification requires Windows AMD64")
	}
	path := "../../libraries/Standard/Backend/AMD64.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	fixture := "testdata/evt2e5_calls.concept"
	input, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := Parse(fixture, string(input))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(checked)
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
	for i, f := range machine.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", f.Name, i)
	}
	harness := strings.Replace(nativeCallsHarness, "/* ARTIFACT */", data.String(), 1)
	var normal string
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		mode := "Normal"
		if verify {
			policy = VerifyCompilationPolicy()
			mode = "Verify"
		}
		outputs, err := GenerateForTargetWithPolicy(backend, source, GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		oracle, err := GenerateForTargetWithPolicy(checked, input, GenericC11Target(), policy)
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
		t.Run(mode, func(t *testing.T) {
			output := runFoundationNativeHarnessOutput(t, outputs, "calls_host.c", harness, "-pedantic-errors", "-O2")
			var observations []string
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "image ") || strings.HasPrefix(line, "symbol ") || strings.HasPrefix(line, "fixup ") {
					observations = append(observations, strings.TrimSpace(line))
				}
			}
			canonical := strings.Join(observations, "\n")
			if !verify {
				normal = canonical
			} else if normal != canonical {
				t.Fatal("Normal/Verify layout/fixups/bytes differ")
			}
		})
	}
}

const nativeCallsHarness = `#include "amd64.generated.h"
#include "oracle.c"
#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#ifdef _WIN32
#include <windows.h>
#endif
#define F(name) concept_standard__backend__amd64_##name
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"failed line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
#define REJECT(expr,name) do{concept_result_void_native_error rejected=(expr);REQUIRE(rejected.tag&&!strcmp(F(describe_native_error)(rejected.payload.error.error),name));}while(0)
/* ARTIFACT */
static unsigned char code[65536],again[65536];
static concept_native_image image,other;
static void *entry(void *base,int ordinal){return (unsigned char*)base+image.symbols.data[ordinal].offset;}
static int describe(char *s,const concept_native_image *p){
 int n=sprintf(s,"image bytes=%d symbols=%d calls=%d finalized=%d",p->byteCount,p->symbolCount,p->fixupCount,p->finalized);
 for(int i=0;i<p->symbolCount;++i){concept_native_symbol a=p->symbols.data[i];n+=sprintf(s+n," s=%d:%d:%d:%d:%d:%d:%d:%d",a.ordinal,a.offset,a.size,a.calls,a.identity.offset,a.identity.length,a.name.offset,a.name.length);}
 for(int i=0;i<p->fixupCount;++i){concept_call_fixup a=p->fixups.data[i];n+=sprintf(s+n," f=%u:%d:%d:%d:%d:%d:%d:%d:%d:%d",a.kind.tag,a.sourceOrdinal,a.source.offset,a.source.length,a.target.offset,a.target.length,a.callOffset,a.patchOffset,a.displacement,a.resolved);}
 return n;
}
#ifdef _WIN32
typedef int (*Fn0)(void);typedef int (*Fn1)(int);typedef int (*Fn2)(int,int);typedef int (*Fn3)(int,int,int);typedef int (*Fn4)(int,int,int,int);typedef int (*Fn5)(int,int,int,int,int);typedef int (*Fn8)(int,int,int,int,int,int,int,int);
#define PTR(type,name,ord) type name=NULL;void *name##_entry=entry(executable,ord);memcpy(&name,&name##_entry,sizeof name)
// Independent ABI wrapper: retain host nonvolatile registers, seed eight
// sentinels, call generated code with four scalar args and check RSP/canary.
__attribute__((naked)) static uint64_t sentinel(void *target,int a,int b,int c,int d) {
 __asm__(".intel_syntax noprefix\n"
 "push rbx\n push rbp\n push rsi\n push rdi\n push r12\n push r13\n push r14\n push r15\n sub rsp,72\n"
 "mov r10,rcx\n mov ecx,edx\n mov edx,r8d\n mov r8d,r9d\n mov r9d,DWORD PTR [rsp+176]\n"
 "mov QWORD PTR [rsp+40],rsp\n mov QWORD PTR [rsp+48],0x7654321\n"
 "mov ebx,0x1111\n mov ebp,0x2222\n mov esi,0x3333\n mov edi,0x4444\n mov r12d,0x5555\n mov r13d,0x6666\n mov r14d,0x7777\n mov r15d,0x8888\n"
 "call r10\n mov DWORD PTR [rsp+56],eax\n cmp rsp,QWORD PTR [rsp+40]\n jne 1f\n cmp QWORD PTR [rsp+48],0x7654321\n jne 1f\n"
 "cmp rbx,0x1111\n jne 1f\n cmp rbp,0x2222\n jne 1f\n cmp rsi,0x3333\n jne 1f\n cmp rdi,0x4444\n jne 1f\n"
 "cmp r12,0x5555\n jne 1f\n cmp r13,0x6666\n jne 1f\n cmp r14,0x7777\n jne 1f\n cmp r15,0x8888\n jne 1f\n"
 "mov eax,DWORD PTR [rsp+56]\n jmp 2f\n 1: mov rax,-1\n 2: add rsp,72\n pop r15\n pop r14\n pop r13\n pop r12\n pop rdi\n pop rsi\n pop rbp\n pop rbx\n ret\n .att_syntax prefix\n");
}
#endif
int main(void) {
 concept_readonly_span_byte input={artifact,sizeof artifact};concept_span_byte output={code,sizeof code};
 concept_byte_writer alias={{again,sizeof again},0};concept_register rbp={6};
 REQUIRE(!F(write_mov_immediate)(&alias,rbp,0xd3,1).tag&&alias.offset==3&&again[0]==0x40&&again[1]==0xb5&&again[2]==0xd3);
 alias.offset=0;REQUIRE(!F(write_stack_adjustment)(&alias,1,136).tag&&alias.offset==7&&!memcmp(again,"\x48\x81\xec\x88\x00\x00\x00",7));
 alias.offset=0;concept_register r10={10};REQUIRE(!F(write_rspreg)(&alias,0,r10,136,2).tag&&alias.offset==9&&!memcmp(again,"\x66\x44\x89\x94\x24\x88\x00\x00\x00",9));
 clock_t start=clock();concept_result_int_native_error result=F(emit_module)(input,output,&image);
 if(result.tag)fprintf(stderr,"emission: %s\n",F(describe_native_error)(result.payload.error.error));
 REQUIRE(!result.tag && image.finalized && image.fixupCount>0);
 double encoding=1000.0*(clock()-start)/CLOCKS_PER_SEC;
 concept_bridge_reader reader={input,0};REQUIRE(!F(validate_native_image)(&reader,&image,output).tag);
 char baseline[32768],next[32768];describe(baseline,&image);puts(baseline);
 printf("image hex=");for(int i=0;i<image.byteCount;++i)printf("%02x",code[i]);puts("");
 printf("metadata symbol=%zu fixup=%zu image=%zu\n",sizeof(concept_native_symbol),sizeof(concept_call_fixup),sizeof(concept_native_image));
 int forward=0,backward=0;
 for(int i=0;i<image.symbolCount;++i)printf("symbol %d offset=%d size=%d\n",i,image.symbols.data[i].offset,image.symbols.data[i].size);
 for(int i=0;i<image.fixupCount;++i){concept_call_fixup f=image.fixups.data[i];printf("fixup source=%d site=%d delta=%d bytes=%02x%02x%02x%02x%02x\n",f.sourceOrdinal,f.callOffset,f.displacement,code[f.callOffset],code[f.callOffset+1],code[f.callOffset+2],code[f.callOffset+3],code[f.callOffset+4]);forward+=f.displacement>0;backward+=f.displacement<0;}
 REQUIRE(forward && backward);
 // Exact first forward call: no arguments, sub rsp,40; E8 to next body.
 REQUIRE(code[0]==0x48&&code[1]==0x83&&code[2]==0xec&&code[3]==0x28);
 REQUIRE(image.fixups.data[0].callOffset==4 && code[4]==0xe8 && image.fixups.data[0].displacement==11 && code[5]==11 && code[6]==0 && code[7]==0 && code[8]==0);
 concept_span_byte repeated={again,sizeof again};
 for(int run=0;run<100;++run){REQUIRE(!F(emit_module)(input,repeated,&other).tag);describe(next,&other);REQUIRE(!strcmp(baseline,next) && !memcmp(code,again,(size_t)image.byteCount));}
 // Patch admission rejects absent symbol, duplicate identity, bad opcode,
 // duplicate patch, resolved input, missing resolution and changed bytes.
 other=image;other.finalized=0;for(int i=0;i<other.fixupCount;++i){concept_call_fixup f=other.fixups.data[i];for(int j=0;j<4;++j)again[f.patchOffset+j]=0;f.resolved=0;other.fixups.data[i]=f;}
 concept_native_image pending=other;REQUIRE(!F(resolve_native_calls)(&reader,&other,repeated).tag);
 start=clock();for(int run=0;run<1000;++run){other=pending;memcpy(again,code,(size_t)image.byteCount);for(int i=0;i<other.fixupCount;++i)memset(again+other.fixups.data[i].patchOffset,0,4);REQUIRE(!F(resolve_native_calls)(&reader,&other,repeated).tag);}printf("timing fixup 1000 passes %.3f ms\n",1000.0*(clock()-start)/CLOCKS_PER_SEC);
 REJECT(F(resolve_native_calls)(&reader,&other,repeated),"AMD64_NATIVE_INVALID_INPUT");
 other=pending;other.symbols.data[1].identity=other.symbols.data[0].identity;REJECT(F(resolve_native_calls)(&reader,&other,repeated),"AMD64_DUPLICATE_SYMBOL");
 memcpy(again,code,(size_t)image.byteCount);for(int i=0;i<pending.fixupCount;++i)memset(again+pending.fixups.data[i].patchOffset,0,4);
 other=pending;other.fixups.data[0].target.length=0;REJECT(F(resolve_native_calls)(&reader,&other,repeated),"AMD64_UNRESOLVED_SYMBOL");REQUIRE(!other.finalized);
 other=pending;other.fixupCount--;REJECT(F(resolve_native_calls)(&reader,&other,repeated),"AMD64_UNRESOLVED_CALL_FIXUP");REQUIRE(!other.finalized);
 other=image;other.finalized=0;other.fixups.data[0].target.length=0;REQUIRE(F(find_native_symbol)(&reader,&other,other.fixups.data[0].target).tag);
 other=image;other.fixups.data[0].resolved=0;REJECT(F(validate_native_image)(&reader,&other,output),"AMD64_UNRESOLVED_CALL_FIXUP");
 other=image;other.fixupCount--;REJECT(F(validate_native_image)(&reader,&other,output),"AMD64_UNRESOLVED_CALL_FIXUP");
 other=image;other.symbolCount=129;REQUIRE(F(validate_native_image)(&reader,&other,output).tag);
 other=image;other.fixups.data[0].source=other.symbols.data[1].identity;REQUIRE(F(validate_native_image)(&reader,&other,output).tag);
 other=image;other.fixups.data[1]=other.fixups.data[0];REQUIRE(F(validate_native_image)(&reader,&other,output).tag);
 int patch=image.fixups.data[0].callOffset;unsigned char saved=code[patch];code[patch]=0x90;REQUIRE(F(validate_native_image)(&reader,&image,output).tag);code[patch]=saved;
 other=image;other.fixups.data[1].callOffset=other.fixups.data[0].callOffset;other.fixups.data[1].patchOffset=other.fixups.data[0].patchOffset;other.finalized=0;REQUIRE(F(resolve_native_calls)(&reader,&other,output).tag);
 REQUIRE(F(call_displacement)(2147483652ULL,0).payload.ok.value==2147483647);
 REQUIRE(F(call_displacement)(0,2147483643ULL).payload.ok.value==(-2147483647-1));
 REQUIRE(F(call_displacement)(2147483653ULL,0).tag && F(call_displacement)(0,2147483644ULL).tag && F(call_displacement)(0,UINT64_MAX).tag);
 concept_span_byte small={again,1};REQUIRE(F(emit_module)(input,small,&other).tag && !other.finalized);
#ifdef _WIN32
 void *executable=VirtualAlloc(NULL,(size_t)image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(executable);
 memcpy(executable,code,(size_t)image.byteCount);DWORD old=0;
 REQUIRE(VirtualProtect(executable,(size_t)image.byteCount,PAGE_EXECUTE_READ,&old));REQUIRE(FlushInstructionCache(GetCurrentProcess(),executable,(size_t)image.byteCount));
 PTR(Fn0,fwd,ORD_Forward);PTR(Fn0,back,ORD_Back);PTR(Fn1,one,ORD_One);PTR(Fn4,four,ORD_Call4);PTR(Fn5,five,ORD_Call5);PTR(Fn8,eight,ORD_Call8);
 PTR(Fn3,across,ORD_Across);PTR(Fn4,pressure,ORD_Pressure);PTR(Fn3,chain,ORD_Chain);PTR(Fn8,multi,ORD_Multi);PTR(Fn2,repeat,ORD_Repeat);PTR(Fn1,early,ORD_Early);PTR(Fn1,fact,ORD_Fact);
 typedef void(*Void)(int);PTR(Void,touch,ORD_CallVoid);typedef bool(*Bool)(bool);PTR(Bool,boolean,ORD_CallBool);
 typedef uint8_t(*Byte)(uint8_t);PTR(Byte,byte,ORD_CallNarrow8);typedef uint16_t(*Word)(uint16_t);PTR(Word,word,ORD_CallNarrow16);typedef uint64_t(*Wide)(uint64_t);PTR(Wide,wide,ORD_CallWide);
 PTR(Byte,keep8,ORD_Keep8);PTR(Word,keep16,ORD_Keep16);PTR(Wide,keep64,ORD_Keep64);PTR(Bool,keepbool,ORD_KeepBool);
 start=clock();
 for(int run=0;run<100;++run){
  REQUIRE(fwd()==concept_calls_forward() && back()==concept_calls_back());
  REQUIRE(one(-37)==concept_calls_one(-37));REQUIRE(four(1,-2,7,11)==concept_calls_call4(1,-2,7,11));
  REQUIRE(five(1,-2,7,11,23)==concept_calls_call5(1,-2,7,11,23));REQUIRE(eight(1,2,4,8,16,32,64,128)==concept_calls_call8(1,2,4,8,16,32,64,128));
  REQUIRE(across(11,23,41)==concept_calls_across(11,23,41));REQUIRE(pressure(1,2,4,8)==concept_calls_pressure(1,2,4,8));
  REQUIRE(chain(2,3,7)==concept_calls_chain(2,3,7));REQUIRE(multi(1,2,4,8,16,32,64,128)==concept_calls_multi(1,2,4,8,16,32,64,128));
  REQUIRE(repeat(7,11)==concept_calls_repeat(7,11));REQUIRE(early(-4)==concept_calls_early(-4) && early(4)==concept_calls_early(4));REQUIRE(fact(5)==concept_calls_fact(5));
  touch(11);REQUIRE(!boolean(false)&&boolean(true));REQUIRE(byte(0xd3)==concept_calls_call_narrow8(0xd3));REQUIRE(word(0xd357)==concept_calls_call_narrow16(0xd357));REQUIRE(wide(UINT64_C(0xabcd010203040506))==concept_calls_call_wide(UINT64_C(0xabcd010203040506)));
  REQUIRE(keep8(0xd3)==concept_calls_keep8(0xd3)&&keep16(0xd357)==concept_calls_keep16(0xd357)&&keep64(UINT64_C(0xabcd010203040506))==concept_calls_keep64(UINT64_C(0xabcd010203040506))&&!keepbool(false)&&keepbool(true));
 }
 printf("native 100 passes C11 parity %.3f ms encoding %.3f ms\n",1000.0*(clock()-start)/CLOCKS_PER_SEC,encoding);
 REQUIRE(VirtualFree(executable,0,MEM_RELEASE));
 // This wrapper protects its own frame while probing actual generated code.
 executable=VirtualAlloc(NULL,(size_t)image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(executable);memcpy(executable,code,(size_t)image.byteCount);
 REQUIRE(VirtualProtect(executable,(size_t)image.byteCount,PAGE_EXECUTE_READ,&old)&&FlushInstructionCache(GetCurrentProcess(),executable,(size_t)image.byteCount));
 REQUIRE(sentinel(entry(executable,ORD_Pressure),1,2,4,8)==(uint32_t)concept_calls_pressure(1,2,4,8));
 REQUIRE(sentinel(entry(executable,ORD_FullSaved),0,0,0,0)==255);
 REQUIRE(sentinel(entry(executable,ORD_Chain),2,3,7,0)==(uint32_t)concept_calls_chain(2,3,7));
 REQUIRE(sentinel(entry(executable,ORD_Early),4,0,0,0)==11 && sentinel(entry(executable,ORD_Early),-4,0,0,0)==(uint32_t)-4);
 REQUIRE(VirtualFree(executable,0,MEM_RELEASE));
 // Independent leaf entry-RSP probe, placed at the Concept-selected Zero
 // body offset in a separate writable image. No call byte is changed.
 static const unsigned char rsp_probe[]={0x48,0x89,0xe0,0x83,0xe0,0x0f,0xc3};
 REQUIRE(image.symbols.data[ORD_Zero].size>=7);
 executable=VirtualAlloc(NULL,(size_t)image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(executable);memcpy(executable,code,(size_t)image.byteCount);memcpy(entry(executable,ORD_Zero),rsp_probe,sizeof rsp_probe);
 REQUIRE(VirtualProtect(executable,(size_t)image.byteCount,PAGE_EXECUTE_READ,&old)&&FlushInstructionCache(GetCurrentProcess(),executable,(size_t)image.byteCount));
 PTR(Fn0,align,ORD_Forward);REQUIRE(align()==8);REQUIRE(VirtualFree(executable,0,MEM_RELEASE));
 // Controlled allocations make physical cycles necessary while retaining
 // real checked literal definitions, frame actions, fixups and native calls.
 for(int cycle=2;cycle<=3;++cycle){
  concept_native_image probe=F(empty_native_image)();concept_byte_writer writer={{again,sizeof again},0};
  concept_encoding_plan branches=F(empty_encoding_plan)();concept_encoding_workspace workspace={{branches.blockOffsets.data,128},{branches.fixups.data,256},0,{probe.fixups.data,512},0,0,1};
  for(int ordinal=0;ordinal<2;++ordinal){
   concept_backend_function function=F(empty_function)();int selected=ordinal?(cycle==2?ORD_Cycle2:ORD_Cycle3):ORD_Ordered;
   REQUIRE(!F(decode_function)(input,selected,&function).tag&&!F(layout_lowered_blocks)(&function).tag);
   concept_allocation allocation=F(empty_allocation)();REQUIRE(!F(allocate_registers_with_policy)(&function,&allocation,1).tag);
   if(ordinal){const unsigned regs2[]={3,2,8,9},regs3[]={3,8,2,9};for(int p=0;p<function.instructionCount;++p)if(function.instructions.data[p].op.tag==16){for(int i=0;i<4;++i)allocation.physical.data[function.instructions.data[p].call.values.data[i]].tag=cycle==2?regs2[i]:regs3[i];}}
   concept_finalized_frame frame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&function,&allocation,&frame).tag);
   if(ordinal)REQUIRE(frame.tempSlots.data[0]>=0);
   workspace.functionOrdinal=ordinal;int begin=writer.offset;
   REQUIRE(!F(encode_function_into_writer)(&writer,&reader,&function,&allocation,&frame,&workspace).tag);
   probe.symbols.data[ordinal]=(concept_native_symbol){function.identity,function.name,ordinal,begin,writer.offset-begin,allocation.callCount};probe.symbolCount++;
  }
  probe.fixupCount=workspace.callCount;probe.byteCount=writer.offset;concept_span_byte view={again,sizeof again};REQUIRE(!F(resolve_native_calls)(&reader,&probe,view).tag && probe.finalized);
  executable=VirtualAlloc(NULL,(size_t)probe.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(executable);memcpy(executable,again,(size_t)probe.byteCount);
  REQUIRE(VirtualProtect(executable,(size_t)probe.byteCount,PAGE_EXECUTE_READ,&old)&&FlushInstructionCache(GetCurrentProcess(),executable,(size_t)probe.byteCount));
  void *address=(unsigned char*)executable+probe.symbols.data[1].offset;Fn0 fn=NULL;memcpy(&fn,&address,sizeof fn);
  REQUIRE(fn()==concept_calls_ordered(11,22,33,44));REQUIRE(VirtualFree(executable,0,MEM_RELEASE));
 }
 puts("native ABI sentinels, entry RSP=8 mod16 and two/three-way cycle calls passed");
#else
 fprintf(stderr,"native Win64 execution unavailable\n");return 77;
#endif
 return 0;
}
`
