package concept

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2f4ScalarSSEPrimitives(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("actual Win64 XMM entrypoints require Windows AMD64")
	}
	var normal string
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			out := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "sse_host.c", scalarSSEHarness, "-pedantic-errors", "-O2", "-fno-fast-math")
			var lines []string
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "bytes ") {
					lines = append(lines, strings.TrimSpace(line))
				}
			}
			observed := strings.Join(lines, "\n")
			if verify && observed != normal {
				t.Fatal("Normal/Verify scalar bytes differ")
			}
			normal = observed
		})
	}
}

const scalarSSEHarness = `#include "amd64.generated.h"
#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <math.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"SSE check failed line %d: %s\n",__LINE__,#c);return 1;}}while(0)
static unsigned char bytes[256],baseline[256];
static concept_byte_writer writer;
static concept_scalar_representation rep(int wide){return (concept_scalar_representation){wide?11:10};}
static concept_scalar_sseoperand operand(int x,int wide){return (concept_scalar_sseoperand){0,{{0},{1},{0},{x},-1},{wide?11:10},-1,wide?8:4};}
static int emit(int op,int wide){writer=(concept_byte_writer){{bytes,sizeof bytes},0};return F(emit_scalar_sse)(&writer,(concept_scalar_opcode){op},rep(wide),operand(0,wide),operand(1,wide)).tag;}
static void *publish(void){void *p=VirtualAlloc(NULL,(size_t)writer.offset,MEM_RESERVE|MEM_COMMIT,PAGE_READWRITE);if(!p)return NULL;memcpy(p,bytes,(size_t)writer.offset);DWORD previous;if(!VirtualProtect(p,(size_t)writer.offset,PAGE_EXECUTE_READ,&previous)||!FlushInstructionCache(GetCurrentProcess(),p,(size_t)writer.offset)){VirtualFree(p,0,MEM_RELEASE);return NULL;}return p;}
static int finish(void){return F(write_byte)(&writer,195).tag;}
static uint32_t bits32(float x){uint32_t b;memcpy(&b,&x,4);return b;}
static uint64_t bits64(double x){uint64_t b;memcpy(&b,&x,8);return b;}
static float float32(uint32_t b){float x;memcpy(&x,&b,4);return x;}
static double float64(uint64_t b){double x;memcpy(&x,&b,8);return x;}
typedef float(*F32)(float,float);typedef double(*F64)(double,double);typedef int(*C32)(float,float);typedef int(*C64)(double,double);
static float oracle32(int op,float a,float b){switch(op){case 20:return a+b;case 21:return a-b;case 22:return a*b;default:return a/b;}}
static double oracle64(int op,double a,double b){switch(op){case 20:return a+b;case 21:return a-b;case 22:return a*b;default:return a/b;}}
static int compare32(int op,float a,float b){switch(op){case 25:return a==b;case 26:return a!=b;case 27:return a<b;case 28:return a<=b;case 29:return a>b;default:return a>=b;}}
static int compare64(int op,double a,double b){switch(op){case 25:return a==b;case 26:return a!=b;case 27:return a<b;case 28:return a<=b;case 29:return a>b;default:return a>=b;}}
int main(void){
 const int ops[]={4,20,21,22,23,25};const int codes[]={0x10,0x58,0x5c,0x59,0x5e,0x2e};
 for(int wide=0;wide<2;wide++)for(int i=0;i<6;i++){
  REQUIRE(!emit(ops[i],wide));unsigned char expected[4]={(unsigned char)(wide?0xf2:0xf3),0x0f,(unsigned char)codes[i],0xc1};int n=4;
  if(i==5){expected[0]=0x66;if(!wide){memmove(expected,expected+1,3);n=3;}}
  REQUIRE(writer.offset==n&&!memcmp(bytes,expected,(size_t)n));memcpy(baseline,bytes,(size_t)n);
  for(int run=0;run<100;run++){REQUIRE(!emit(ops[i],wide));REQUIRE(writer.offset==n&&!memcmp(bytes,baseline,(size_t)n));}
  printf("bytes %d/%d ",wide,ops[i]);for(int j=0;j<n;j++)printf("%02x",bytes[j]);puts("");
 }
 for(int wide=0;wide<2;wide++)for(int op=20;op<=23;op++){
  REQUIRE(!emit(op,wide)&&!finish());void *p=publish();REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);
  const double a[]={7.5,-13.0,0.0,-0.0,1.0,INFINITY,-INFINITY,0.0,INFINITY,3.0,INFINITY},b[]={2.0,3.0,-INFINITY,INFINITY,0.0,0.0,INFINITY,0.0,INFINITY,-0.0,2.0};
  for(int i=0;i<11;i++){if(wide){double actual=d(a[i],b[i]),expected=oracle64(op,a[i],b[i]);REQUIRE((isnan(actual)&&isnan(expected))||bits64(actual)==bits64(expected));}else{float actual=f((float)a[i],(float)b[i]),expected=oracle32(op,(float)a[i],(float)b[i]);REQUIRE((isnan(actual)&&isnan(expected))||bits32(actual)==bits32(expected));}}
  if(wide&&op==20){clock_t start=clock();volatile double sum=0;for(int run=0;run<100000;run++)sum+=d(7.5,2.0);REQUIRE(sum==950000.0);printf("timing native arithmetic 100000 calls %.3f ms\n",1000.0*(clock()-start)/CLOCKS_PER_SEC);}
  REQUIRE(VirtualFree(p,0,MEM_RELEASE));
 }
 const uint32_t domain32[]={0xc0000000,0,0x80000000,0x40000000,0x7f800000,0xff800000,0x7fc01234};
 const uint64_t domain64[]={UINT64_C(0xc000000000000000),0,UINT64_C(0x8000000000000000),UINT64_C(0x4000000000000000),UINT64_C(0x7ff0000000000000),UINT64_C(0xfff0000000000000),UINT64_C(0x7ff8000000001234)};
 int comparisons=0;
 for(int wide=0;wide<2;wide++)for(int op=25;op<=30;op++){
  REQUIRE(!emit(op,wide));REQUIRE(!F(write_scalar_compare_bool)(&writer,(concept_scalar_opcode){op}).tag&&!finish());void *p=publish();REQUIRE(p);C32 f=NULL;C64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);
  for(int a=0;a<7;a++)for(int b=0;b<7;b++){if(wide){double x=float64(domain64[a]),y=float64(domain64[b]);REQUIRE(d(x,y)==compare64(op,x,y));}else{float x=float32(domain32[a]),y=float32(domain32[b]);REQUIRE(f(x,y)==compare32(op,x,y));}comparisons++;}REQUIRE(VirtualFree(p,0,MEM_RELEASE));
 }
 for(int wide=0;wide<2;wide++){
  writer=(concept_byte_writer){{bytes,sizeof bytes},0};REQUIRE(!F(write_scalar_negation)(&writer,(concept_xmmregister){0},rep(wide)).tag);
  const unsigned char neg32[]={0x66,0x0f,0x7e,0xc0,0x0f,0xba,0xf8,0x1f,0x66,0x0f,0x6e,0xc0};
  const unsigned char neg64[]={0x66,0x48,0x0f,0x7e,0xc0,0x48,0x0f,0xba,0xf8,0x3f,0x66,0x48,0x0f,0x6e,0xc0};
  REQUIRE(writer.offset==(int)(wide?sizeof neg64:sizeof neg32)&&!memcmp(bytes,wide?neg64:neg32,(size_t)writer.offset));REQUIRE(!finish());void *p=publish();REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);
  for(int i=0;i<7;i++){if(wide)REQUIRE(bits64(d(float64(domain64[i]),0))==(domain64[i]^UINT64_C(0x8000000000000000)));else REQUIRE(bits32(f(float32(domain32[i]),0))==(domain32[i]^0x80000000u));}REQUIRE(VirtualFree(p,0,MEM_RELEASE));
  for(int i=0;i<8;i++){uint64_t raw=wide?(i==7?UINT64_C(0x3ff0000000000001):domain64[i]):(i==7?0x3dcccccdu:domain32[i]);unsigned char literal[8];for(int j=0;j<8;j++)literal[j]=(unsigned char)(raw>>(8*j));
   writer=(concept_byte_writer){{bytes,sizeof bytes},0};concept_readonly_span_byte bits={literal,wide?8:4};REQUIRE(!F(write_scalar_constant)(&writer,bits,(concept_xmmregister){0},rep(wide)).tag&&!finish());p=publish();REQUIRE(p);memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);if(wide)REQUIRE(bits64(d(0,0))==raw);else REQUIRE(bits32(f(0,0))==(uint32_t)raw);REQUIRE(VirtualFree(p,0,MEM_RELEASE));
  }
 }
 // High XMM identities are encoder-only; the allocator pool stays XMM0..5.
 writer=(concept_byte_writer){{bytes,sizeof bytes},0};REQUIRE(!F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(1),operand(8,1),operand(9,1)).tag);REQUIRE(writer.offset==5&&!memcmp(bytes,"\xf2\x45\x0f\x58\xc1",5));
 // RSP spill/load bytes execute, keeping the return-address slot untouched.
 for(int wide=0;wide<2;wide++){
  writer=(concept_byte_writer){{bytes,sizeof bytes},0};REQUIRE(!F(write_stack_adjustment)(&writer,1,24).tag);
  concept_scalar_sseoperand stack=operand(0,wide);stack.memory=1;stack.location.kind.tag=3;stack.location.slot=0;stack.offset=8;
  REQUIRE(!F(emit_scalar_sse)(&writer,(concept_scalar_opcode){5},rep(wide),stack,operand(0,wide)).tag);
  REQUIRE(!F(emit_scalar_sse)(&writer,(concept_scalar_opcode){4},rep(wide),operand(0,wide),stack).tag);REQUIRE(!F(write_stack_adjustment)(&writer,0,24).tag&&!finish());
  REQUIRE(bytes[4]==(wide?0xf2:0xf3)&&bytes[5]==0x0f&&bytes[6]==0x11&&bytes[7]==0x44&&bytes[8]==0x24&&bytes[9]==8);
  void *p=publish();REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);if(wide)REQUIRE(bits64(d(float64(domain64[6]),0))==domain64[6]);else REQUIRE(bits32(f(float32(domain32[6]),0))==domain32[6]);REQUIRE(VirtualFree(p,0,MEM_RELEASE));
 }
 writer=(concept_byte_writer){{bytes,sizeof bytes},0};memset(bytes,0xa5,sizeof bytes);
 concept_scalar_sseoperand bad=operand(1,0);bad.representation.tag=11;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),bad).tag&&writer.offset==0&&bytes[0]==0xa5);
 bad=operand(1,0);bad.location.registerClass.tag=0;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),bad).tag&&writer.offset==0);
 bad=operand(1,0);bad.memory=1;bad.location.kind.tag=3;bad.offset=2147483647;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),bad).tag&&writer.offset==0);
 bad=operand(1,0);bad.width=8;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),bad).tag);
 bad=operand(1,0);bad.memory=1;bad.location.kind.tag=3;bad.offset=-1;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),bad).tag);
 REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){31},rep(0),operand(0,0),operand(1,0)).tag);
 writer.bytes.length=1;REQUIRE(F(emit_scalar_sse)(&writer,(concept_scalar_opcode){20},rep(0),operand(0,0),operand(1,0)).tag&&writer.offset==0&&bytes[0]==0xa5);
 clock_t started=clock();for(int run=0;run<10000;run++){REQUIRE(!F(select_scalar_sse)((concept_scalar_opcode){20},rep(run%2)).tag);}printf("timing selection 10000 %.3f ms\n",1000.0*(clock()-started)/CLOCKS_PER_SEC);
 started=clock();for(int run=0;run<10000;run++){REQUIRE(!emit(20,run%2));}printf("timing encoding 10000 %.3f ms\n",1000.0*(clock()-started)/CLOCKS_PER_SEC);
 unsigned char finite[]={0xcd,0xcc,0xcc,0x3d};concept_readonly_span_byte material={finite,4};started=clock();for(int run=0;run<10000;run++){writer=(concept_byte_writer){{bytes,sizeof bytes},0};REQUIRE(!F(write_scalar_constant)(&writer,material,(concept_xmmregister){0},rep(0)).tag);}printf("timing constant materialization 10000 %.3f ms\n",1000.0*(clock()-started)/CLOCKS_PER_SEC);
 writer=(concept_byte_writer){{bytes,sizeof bytes},-1};REQUIRE(F(write_scalar_negation)(&writer,(concept_xmmregister){0},rep(0)).tag);
 concept_readonly_span_byte wrong_bits={finite,3};writer.offset=0;REQUIRE(F(write_scalar_constant)(&writer,wrong_bits,(concept_xmmregister){0},rep(0)).tag&&writer.offset==0);
 REQUIRE(comparisons==588);printf("native primitive comparisons=%d, exact constants/negation/spills and strict arithmetic passed\n",comparisons);return 0;
}
`

func TestEVT2f4SourceNativeScalar(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Win64 native execution")
	}
	source, err := os.ReadFile("../../language/evt2f/valid/native_scalar.concept")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := Parse("native_scalar.concept", string(source))
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
	for i, f := range machine.Scalar.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", f.Name, i)
	}
	harness := strings.Replace(scalarSSEHarness, "int main(void){", "static int primitives_main(void){", 1) + data.String() + sourceNativeSSEHarness
	var normal string
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			out := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "source_sse_host.c", harness, "-pedantic-errors", "-O2", "-fno-fast-math")
			var lines []string
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "source bytes ") {
					lines = append(lines, strings.TrimSpace(line))
				}
			}
			current := strings.Join(lines, "\n")
			if verify && current != normal {
				t.Fatal("mode image bytes differ")
			}
			normal = current
		})
	}
}

const sourceNativeSSEHarness = `
typedef float(*P32)(float,float,float,float,float,float,float,float);
typedef double(*P64)(double,double,double,double,double,double,double,double);
static unsigned char native_code[16384],native_again[16384];
static void *publishedModuleBase;
static void *source_function(int ordinal){
 static concept_native_image image;static int ready=0;
 concept_readonly_span_byte input={artifact,sizeof artifact};concept_span_byte output={native_code,sizeof native_code};
 if(!ready){clock_t started=clock();concept_result_int_native_error r=F(emit_module)(input,output,&image);if(r.tag){fprintf(stderr,"module emission error %d\n",r.payload.error.error.tag);return NULL;}memcpy(native_again,native_code,(size_t)image.byteCount);
 for(int run=0;run<100;run++){concept_native_image other;concept_result_int_native_error again=F(emit_module)(input,output,&other);if(again.tag||!other.finalized||other.fixupCount!=0||other.symbolCount!=image.symbolCount||other.byteCount!=image.byteCount||memcmp(native_code,native_again,(size_t)image.byteCount))return NULL;
 for(int i=0;i<image.symbolCount;i++){concept_native_symbol a=image.symbols.data[i],b=other.symbols.data[i];if(a.offset!=b.offset||a.size!=b.size||a.ordinal!=b.ordinal||a.identity.offset!=b.identity.offset||a.identity.length!=b.identity.length||a.name.offset!=b.name.offset||a.name.length!=b.name.length)return NULL;}}
 printf("timing 101 module emissions %.3f ms; bytes=%d symbols=%d\n",1000.0*(clock()-started)/CLOCKS_PER_SEC,image.byteCount,image.symbolCount);printf("source bytes module size=%d symbols=%d ",image.byteCount,image.symbolCount);for(int i=0;i<image.byteCount;i++)printf("%02x",native_code[i]);puts("");ready=1;}
 void *p=VirtualAlloc(NULL,(size_t)image.byteCount,MEM_RESERVE|MEM_COMMIT,PAGE_READWRITE);if(!p)return NULL;memcpy(p,native_code,(size_t)image.byteCount);DWORD previous;if(!VirtualProtect(p,(size_t)image.byteCount,PAGE_EXECUTE_READ,&previous)||!FlushInstructionCache(GetCurrentProcess(),p,(size_t)image.byteCount))return NULL;publishedModuleBase=p;return (unsigned char*)p+image.symbols.data[ordinal].offset;
}
int main(void){
 const int ord32[]={ORD_Add32,ORD_Sub32,ORD_Mul32,ORD_Div32,ORD_Neg32,ORD_Multi32};
 const int ord64[]={ORD_Add64,ORD_Sub64,ORD_Mul64,ORD_Div64,ORD_Neg64,ORD_Multi64};
 for(int w=0;w<2;w++)for(int op=0;op<6;op++){void *p=source_function(w?ord64[op]:ord32[op]);REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);if(w){double expected=op<4?oracle64(op+20,7.5,2.0):op==4?-7.5:(7.5-2.0)/(7.5+2.0);REQUIRE(bits64(d(7.5,2.0))==bits64(expected));}else{float expected=op<4?oracle32(op+20,7.5f,2.0f):op==4?-7.5f:(7.5f-2.0f)/(7.5f+2.0f);REQUIRE(bits32(f(7.5f,2.0f))==bits32(expected));}REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));}
 const int cmp32[]={ORD_Eq32,ORD_Ne32,ORD_Lt32,ORD_Le32,ORD_Gt32,ORD_Ge32};const int cmp64[]={ORD_Eq64,ORD_Ne64,ORD_Lt64,ORD_Le64,ORD_Gt64,ORD_Ge64};
 const uint32_t domain32[]={0xc0000000,0,0x80000000,0x40000000,0x7f800000,0xff800000,0x7fc01234};const uint64_t domain64[]={UINT64_C(0xc000000000000000),0,UINT64_C(0x8000000000000000),UINT64_C(0x4000000000000000),UINT64_C(0x7ff0000000000000),UINT64_C(0xfff0000000000000),UINT64_C(0x7ff8000000001234)};
 for(int w=0;w<2;w++)for(int op=0;op<6;op++){void *p=source_function(w?cmp64[op]:cmp32[op]);REQUIRE(p);C32 f=NULL;C64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);for(int i=0;i<7;i++)for(int j=0;j<7;j++){if(w)REQUIRE(d(float64(domain64[i]),float64(domain64[j]))==compare64(op+25,float64(domain64[i]),float64(domain64[j])));else REQUIRE(f(float32(domain32[i]),float32(domain32[j]))==compare32(op+25,float32(domain32[i]),float32(domain32[j])));}REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));}
 for(int w=0;w<2;w++){void *p=source_function(w?ORD_Constant64:ORD_Constant32);REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);if(w)REQUIRE(bits64(d(0,0))==UINT64_C(0x3ff0000000000001));else REQUIRE(bits32(f(0,0))==0x3dcccccdu);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(w?ORD_Pressure64:ORD_Pressure32);REQUIRE(p);P32 a=NULL;P64 b=NULL;memcpy(&a,&p,sizeof a);memcpy(&b,&p,sizeof b);if(w)REQUIRE(b(1,2,3,4,5,6,7,8)==36);else REQUIRE(a(1,2,3,4,5,6,7,8)==36);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(w?ORD_Branch64:ORD_Branch32);REQUIRE(p);memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);if(w){REQUIRE(d(2,3)==2&&d(4,3)==3);}else{REQUIRE(f(2,3)==2&&f(4,3)==3);}REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));}

 float array32[]={1,2,3,4};double array64[]={1,2,3,4};
 typedef float(*L32)(float*,int);typedef double(*L64)(double*,int);typedef void(*S32)(float*,int,float);typedef void(*S64)(double*,int,double);
 void *p=source_function(ORD_Load32);REQUIRE(p);L32 load32=NULL;memcpy(&load32,&p,sizeof load32);for(int i=0;i<4;i++)REQUIRE(load32(array32,i)==array32[i]);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Load64);REQUIRE(p);L64 load64=NULL;memcpy(&load64,&p,sizeof load64);for(int i=0;i<4;i++)REQUIRE(load64(array64,i)==array64[i]);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Store32);REQUIRE(p);S32 store32=NULL;memcpy(&store32,&p,sizeof store32);store32(array32,2,float32(0x7fc01234));REQUIRE(bits32(array32[2])==0x7fc01234);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Store64);REQUIRE(p);S64 store64=NULL;memcpy(&store64,&p,sizeof store64);store64(array64,1,float64(UINT64_C(0x7ff8000000001234)));REQUIRE(bits64(array64[1])==UINT64_C(0x7ff8000000001234));REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Local32);REQUIRE(p);F32 local32=NULL;memcpy(&local32,&p,sizeof local32);REQUIRE(bits32(local32(7.5f,2.0f))==bits32((7.5f-2.0f)/(7.5f+2.0f)));REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Local64);REQUIRE(p);F64 local64=NULL;memcpy(&local64,&p,sizeof local64);REQUIRE(bits64(local64(7.5,2.0))==bits64((7.5-2.0)/(7.5+2.0)));REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));

 p=source_function(ORD_Interpret32);REQUIRE(p);F32 q32=NULL;memcpy(&q32,&p,sizeof q32);REQUIRE(bits32(q32(-0.0f,0))==0x80000000u);REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));
 p=source_function(ORD_Interpret64);REQUIRE(p);F64 q64=NULL;memcpy(&q64,&p,sizeof q64);REQUIRE(bits64(q64(-0.0,0))==UINT64_C(0x8000000000000000));REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));

 const int br32[]={ORD_BranchEq32,ORD_BranchNe32,ORD_BranchLt32,ORD_BranchLe32,ORD_BranchGt32,ORD_BranchGe32};
 const int br64[]={ORD_BranchEq64,ORD_BranchNe64,ORD_BranchLt64,ORD_BranchLe64,ORD_BranchGt64,ORD_BranchGe64};
 for(int w=0;w<2;w++)for(int op=0;op<6;op++){p=source_function(w?br64[op]:br32[op]);REQUIRE(p);F32 f=NULL;F64 d=NULL;memcpy(&f,&p,sizeof f);memcpy(&d,&p,sizeof d);for(int i=0;i<7;i++)for(int j=0;j<7;j++){if(w){double x=float64(domain64[i]),y=float64(domain64[j]);uint64_t expected=compare64(op+25,x,y)?domain64[i]:domain64[j];REQUIRE(bits64(d(x,y))==expected);}else{float x=float32(domain32[i]),y=float32(domain32[j]);uint32_t expected=compare32(op+25,x,y)?domain32[i]:domain32[j];REQUIRE(bits32(f(x,y))==expected);}}REQUIRE(VirtualFree(publishedModuleBase,0,MEM_RELEASE));}
 puts("source CMIRAMD4 native arithmetic/588 comparisons/588 branch comparisons/constants/pressure/array/local/quantity passed");return 0;
}
`
