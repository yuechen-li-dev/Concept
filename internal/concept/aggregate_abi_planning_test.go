package concept

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func aggregateABIFixture(t *testing.T) (MachineModule, []byte) {
	t.Helper()
	path := "../../language/evt2g4/valid/aggregate_abi.concept"
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
	if machine.Borrow == nil || string(data[:8]) != "CMIRAMD6" {
		t.Fatal("expected unchanged checked CMIRAMD6")
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	return machine, data
}

func TestEVT2g4ArtifactOnlyPlanning(t *testing.T) {
	machine, data := aggregateABIFixture(t)
	dir := t.TempDir()
	artifact := filepath.Join(dir, "checked.cmir")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	var definitions strings.Builder
	fmt.Fprintf(&definitions, "#define ARTIFACT %q\n#define FUNCTION_COUNT %d\n", filepath.ToSlash(artifact), len(machine.Borrow.Objects.Functions))
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&definitions, "#define ORD_%s %d\n", f.Name, i)
	}
	host := strings.Replace(aggregateABIPlanningHost, "/* DEFINITIONS */", definitions.String(), 1)
	var normal string
	for _, verify := range []bool{false, true} {
		output := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "artifact_only_aggregate_planning.c", host, "-pedantic-errors", "-O2")
		var stable []string
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(line, "timing ") {
				t.Log(line)
			} else {
				stable = append(stable, line)
			}
		}
		output = strings.Join(stable, "\n")
		if !verify {
			normal = output
		} else if output != normal {
			t.Fatal("Normal/Verify ABI plans differ")
		}
	}
	t.Logf("unchanged CMIRAMD6 bytes=%d; source-free structural planning only; no aggregate native ABI execution claim", len(data))
}

// This independently compiled target IR is a test oracle, never a production
// classifier. These literal expected signatures come from Microsoft's default
// x64 calling convention and are compared to Cathedral's fixture table.
func TestEVT2g4IndependentWin64Compiler(t *testing.T) {
	compiler, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal("independent Win64 clang probe required:", err)
	}
	dir := t.TempDir()
	source := "../../language/evt2g4/probes/win64_aggregate.c"
	ir := filepath.Join(dir, "probe.ll")
	cmd := exec.Command(compiler, "--target=x86_64-pc-windows-msvc", "-std=c11", "-pedantic-errors", "-S", "-emit-llvm", "-O0", source, "-o", ir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Win64 C oracle: %v\n%s", err, out)
	}
	data, err := os.ReadFile(ir)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, returned, arg string }{
		{"RoundOne", "i8", "i8"}, {"RoundTwo", "i16", "i16"}, {"RoundFour", "i32", "i32"}, {"RoundEight", "i64", "i64"},
		{"RoundThree", "void", "ptr"}, {"RoundFive", "void", "ptr"}, {"RoundSix", "void", "ptr"}, {"RoundSeven", "void", "ptr"},
		{"RoundTwelve", "void", "ptr"}, {"RoundSixteen", "void", "ptr"}, {"RoundTwentyFour", "void", "ptr"}, {"RoundPadded", "void", "ptr"},
		{"RoundNested", "i64", "i64"}, {"RoundArray", "i64", "i64"},
	} {
		var signature string
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "define ") && strings.Contains(line, "@"+tc.name+"(") {
				signature = line
				break
			}
		}
		parts := strings.SplitN(signature, "@"+tc.name+"(", 2)
		if len(parts) != 2 || !strings.HasSuffix(strings.TrimSpace(parts[0]), tc.returned) || !strings.HasPrefix(parts[1], tc.arg+" ") {
			t.Fatalf("%s classification: %s", tc.name, signature)
		}
		if tc.returned == "void" && !strings.Contains(signature, "sret(%struct.") {
			t.Fatalf("hidden return missing: %s", signature)
		}
	}
	text := string(data)
	if !strings.Contains(text, "target triple = \"x86_64-pc-windows-msvc") || !strings.Contains(text, "sret(%struct.TwentyFour)") || !strings.Contains(text, "alloca %struct.TwentyFour, align 8") {
		t.Fatal("Win64 hidden-result / caller temporary probe missing")
	}
	t.Log("14 independent C record classifications match. This clang probe uses natural align 8 for a 24-byte caller temporary; Cathedral enforces Microsoft's documented align 16. No external temporary-alignment conformance claim; generated native ABI execution remains EVT2g5.")
}

func TestEVT2g4Boundaries(t *testing.T) {
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char output[65536];static concept_native_image image,before;\n")
	for i, name := range []string{"hidden_argument_capacity", "stack_probe"} {
		path := "../../language/evt2g4/boundaries/" + name + ".concept"
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
		fmt.Fprintf(&c, "static unsigned char artifact%d[]={", i)
		for _, b := range data {
			fmt.Fprintf(&c, "%d,", b)
		}
		c.WriteString("};\n")
	}
	c.WriteString("int main(void){concept_span_byte dst={output,sizeof output};memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;\n")
	for i, want := range []string{"AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED", "AMD64_STACK_PROBE_REQUIRED"} {
		fmt.Fprintf(&c, "{concept_readonly_span_byte src={artifact%d,sizeof artifact%d};concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(!r.tag||strcmp(F(describe_native_error)(r.payload.error.error),%q)){fprintf(stderr,\"%%s\\n\",r.tag?F(describe_native_error)(r.payload.error.error):\"ACCEPTED\");return 1;}if(memcmp(&image,&before,sizeof image))return 2;for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 3;}\n", i, i, want)
	}
	c.WriteString("puts(\"hidden physical-argument capacity and stack-probing boundaries reject before output/image mutation\");return 0;}\n")
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "aggregate_boundaries.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g4ScalarLeafFallback(t *testing.T) {
	// The shared allocator correction also takes the real scalar encoder path.
	m, err := Parse("leaf.concept", `module Leaf; profile Core;
int Sum(int a, int b, int c, int d, int e, int f) {
    return a + b + c + d + e + f;
}
double ScalarLane(double value) {
    return value;
}`)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if machine.Scalar == nil {
		t.Fatal("expected scalar CMIRAMD4 allocator lane")
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <windows.h>\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString(`};
int main(void){unsigned char bytes[8192];concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};concept_native_image image;
concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(r.tag){fprintf(stderr,"%s\n",F(describe_native_error)(r.payload.error.error));return 1;}
concept_backend_function function;concept_allocation allocation=F(empty_allocation)();concept_finalized_frame frame=F(empty_finalized_frame)();
if(F(decode_scalar_function)(src,0,&function).tag||F(allocate_registers)(&function,&allocation).tag||F(qualify_scalar_native_frame)(&function,&allocation,&frame).tag)return 2;
if(frame.regions.data[3].size<8)return 3;
unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 4;memcpy(memory,bytes,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 5;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
typedef int(*Sum)(int,int,int,int,int,int);Sum sum=NULL;void *entry=memory;memcpy(&sum,&entry,sizeof sum);int result=sum(1,2,3,4,5,6);VirtualFree(memory,0,MEM_RELEASE);if(result!=21)return 6;
puts("shared leaf GPR fallback executes Sum(1..6)=21 with a saved register");return 0;}
`)
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "scalar_leaf_fallback.c", c.String(), "-pedantic-errors", "-O2")
	}
}

const aggregateABIPlanningHost = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
/* DEFINITIONS */
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(x) do { if(!(x)){fprintf(stderr,"line %d: %s\n",__LINE__,#x);return 1;} }while(0)
static unsigned char artifact[200000],output[65536];
static concept_backend_function function;
static concept_object_placement placement;
static concept_allocation allocation;
static concept_finalized_frame frame;
static concept_aggregate_function_plan plan;
static unsigned long long digest;
/* Canonical debug text compares every inspected field, without C padding. */
static char debugText[65536],firstText[FUNCTION_COUNT][65536];
static size_t debugLength;
static int debugOverflow;
static void word(int n){digest^=(unsigned)n;digest*=1099511628211ULL;if(debugOverflow)return;int count=snprintf(debugText+debugLength,sizeof debugText-debugLength,"%d,",n);if(count<0||(size_t)count>=sizeof debugText-debugLength){debugOverflow=1;return;}debugLength+=(size_t)count;}
static void location(concept_scalar_location l){word(l.kind.tag);word(l.registerClass.tag);word(l.gpr.tag);word(l.xmm.tag);word(l.slot);}
static void type(concept_aggregate_abitype t){word(t.type);word(t.size);word(t.alignment);word(t.category.tag);word(t.carrier.tag);for(int i=0;i<t.identity.length;i++)word(artifact[t.identity.offset+i]);for(int i=0;i<t.layout.length;i++)word(artifact[t.layout.offset+i]);}
static unsigned long long fingerprint(void){
 digest=1469598103934665603ULL;debugLength=0;debugOverflow=0;debugText[0]=0;type(plan.result);word(plan.hiddenPointer);word(plan.parameterCount);word(plan.callCount);word(plan.transferCount);word(plan.returnCount);
 for(int i=0;i<plan.parameterCount;i++){concept_aggregate_argument_plan a=plan.parameters.data[i];type(a.type);word(a.semanticIndex);word(a.physicalIndex);word(a.value);word(a.carrier);word(a.storage);word(a.start);word(a.end);}
 word(frame.byteCount);word(frame.padding);word(frame.slotCount);
 for(int i=0;i<7;i++){word(frame.regions.data[i].category.tag);word(frame.regions.data[i].offset);word(frame.regions.data[i].size);}
 for(int i=0;i<frame.slotCount;i++){concept_frame_slot s=frame.slots.data[i];word(s.category.tag);word(s.identity);word(s.offset);word(s.size);word(s.alignment);}
 for(int i=0;i<plan.callCount;i++){concept_aggregate_call_plan c=plan.calls.data[i];word(c.block);word(c.instruction);word(c.position);word(c.resultValue);word(c.resultStorage);word(c.hiddenPointer);type(c.result);word(c.argumentCount);word(c.physical.shadowSpaceBytes);word(c.physical.stackArgumentBytes);word(c.physical.moveCount);word(c.physical.temporaryCount);
  for(int j=0;j<c.argumentCount;j++){concept_aggregate_argument_plan a=c.arguments.data[j];type(a.type);word(a.value);word(a.carrier);word(a.storage);word(a.physicalIndex);word(a.start);word(a.end);location(c.physical.scalar.destinations.data[a.physicalIndex]);}
  for(int j=0;j<c.physical.moveCount;j++){concept_scalar_move m=c.physical.scalar.moves.data[j];location(m.source);location(m.destination);word(m.representation.tag);}
 }
 for(int i=0;i<plan.transferCount;i++){concept_aggregate_transfer a=plan.transfers.data[i];word(a.kind.tag);word(a.type);word(a.value);word(a.sourceSlot);word(a.storage);word(a.carrier);word(a.point);word(a.end);}
 return digest;
}
static int run_plan(concept_readonly_span_byte src,int ordinal){concept_result_void_aggregate_abierror r=F(plan_aggregate_function)(src,ordinal,&function,&placement,&allocation,&frame,&plan);if(r.tag){fprintf(stderr,"ordinal %d: %s\n",ordinal,F(describe_aggregate_abierror)(r.payload.error.error));return 1;}return 0;}
static int field_copy(concept_bridge_reader *reader,concept_wire_object_module document,int typeID,unsigned char *to,const unsigned char *from,int depth){
 if(depth>16)return 1;
 concept_result_wire_object_type_aggregate_abierror r=F(aggregate_plan_type)(reader,document,typeID);if(r.tag)return 1;concept_wire_object_type t=r.payload.ok.value;
 if(t.Kind.tag==12){for(int i=0;i<t.Fields.count;i++){concept_result_wire_object_field_aggregate_abierror f=F(aggregate_plan_field)(reader,t,i);if(f.tag)return 1;if(field_copy(reader,document,f.payload.ok.value.Type,to+f.payload.ok.value.Offset,from+f.payload.ok.value.Offset,depth+1))return 1;}}
 else if(t.Kind.tag==13){for(int i=0;i<t.Extent;i++)if(field_copy(reader,document,t.Element,to+i*t.Stride,from+i*t.Stride,depth+1))return 1;}
 else memcpy(to,from,(size_t)t.Size);
 return 0;
}
static int validate_plan(concept_bridge_reader *reader,concept_wire_object_module document,int ordinal,const char *expected){
 concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(reader,document,ordinal);if(owner.tag)return 1;
 concept_result_void_aggregate_abierror r=F(validate_aggregate_function_plan)(reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan);
 if(!r.tag||strcmp(F(describe_aggregate_abierror)(r.payload.error.error),expected)){fprintf(stderr,"corrupt plan wanted %s got %s\n",expected,r.tag?F(describe_aggregate_abierror)(r.payload.error.error):"ACCEPTED");return 1;}return 0;
}
int main(void){
 FILE *file=fopen(ARTIFACT,"rb");REQUIRE(file);size_t size=fread(artifact,1,sizeof artifact,file);fclose(file);REQUIRE(size>44 && size<sizeof artifact);
 concept_readonly_span_byte src={artifact,size};concept_span_byte dst={output,sizeof output};
 concept_result_wire_object_module_aggregate_abierror decoded=F(admitted_aggregate_document)(src);REQUIRE(!decoded.tag);concept_wire_object_module document=decoded.payload.ok.value;concept_bridge_reader reader={src,0};
 const int sizes[]={1,2,4,8,3,5,6,7,12,16,24,16,8,8,4};
 const int argumentClasses[]={3,3,3,3,4,4,4,4,4,4,4,4,3,3,3};
 const int resultClasses[]={5,5,5,5,6,6,6,6,6,6,6,6,5,5,6};
 for(int i=0;i<15;i++){REQUIRE(!run_plan(src,i));REQUIRE(plan.parameterCount==1);REQUIRE(plan.parameters.data[0].type.size==sizes[i]);REQUIRE(plan.parameters.data[0].type.category.tag==argumentClasses[i]);REQUIRE(plan.result.category.tag==resultClasses[i]);}
 unsigned long long first[FUNCTION_COUNT];
 for(int run=0;run<100;run++)for(int i=0;i<FUNCTION_COUNT;i++){
  REQUIRE(!run_plan(src,i));unsigned long long hash=fingerprint();REQUIRE(!debugOverflow);if(!run){first[i]=hash;memcpy(firstText[i],debugText,debugLength+1);printf("function=%d frame=%d slots=%d calls=%d returns=%d transfers=%d plan=%016llx\n",i,frame.byteCount,frame.slotCount,plan.callCount,plan.returnCount,plan.transferCount,hash);}else {REQUIRE(first[i]==hash);REQUIRE(!strcmp(firstText[i],debugText));}
 }
 REQUIRE(!run_plan(src,ORD_Shifted));REQUIRE(plan.result.category.tag==6 && plan.hiddenPointer>=0);
 REQUIRE(plan.parameters.data[0].physicalIndex==1 && plan.parameters.data[1].physicalIndex==2 && plan.parameters.data[2].physicalIndex==3 && plan.parameters.data[3].physicalIndex==4);
 concept_scalar_location shifted=F(win64scalar_argument)(plan.parameters.data[1].physicalIndex,plan.parameters.data[1].type.carrier);REQUIRE(shifted.registerClass.tag==1 && shifted.xmm.tag==2);
 shifted=F(win64scalar_argument)(plan.parameters.data[3].physicalIndex,plan.parameters.data[3].type.carrier);REQUIRE(shifted.kind.tag==1 && shifted.slot==0);
 REQUIRE(!run_plan(src,ORD_Driver));REQUIRE(plan.callCount==5);int indirect=0,hidden=0;
 for(int i=0;i<plan.callCount;i++){concept_aggregate_call_plan c=plan.calls.data[i];REQUIRE(c.physical.shadowSpaceBytes==32);if(c.result.category.tag==6){hidden++;REQUIRE(c.hiddenPointer>=0 && c.resultStorage>=0);}
  for(int j=0;j<c.argumentCount;j++){concept_aggregate_argument_plan a=c.arguments.data[j];if(a.type.category.tag==4){indirect++;REQUIRE(a.storage!=placement.values.data[a.value]);REQUIRE(function.slots.data[a.storage].alignment==16 && frame.slotOffsets.data[a.storage]%16==0);REQUIRE(a.start<c.position && a.end>c.position);}}
 }
 REQUIRE(indirect>=3 && hidden==3);REQUIRE(frame.regions.data[1].size==24);
 REQUIRE(!run_plan(src,ORD_Blend));REQUIRE(plan.parameters.data[0].type.category.tag==3 && plan.parameters.data[0].physicalIndex==1);REQUIRE(plan.parameters.data[1].type.category.tag==4 && plan.parameters.data[1].physicalIndex==2);REQUIRE(plan.parameters.data[3].type.category.tag==2 && plan.parameters.data[3].physicalIndex==4);
 REQUIRE(!run_plan(src,ORD_Early));REQUIRE(plan.returnCount==2);int hiddenReturns=0;for(int i=0;i<plan.transferCount;i++)if(plan.transfers.data[i].kind.tag==8){hiddenReturns++;REQUIRE(plan.transfers.data[i].carrier==plan.hiddenPointer);}REQUIRE(hiddenReturns==2);
 REQUIRE(!run_plan(src,ORD_Chain));REQUIRE(plan.callCount==2 && plan.calls.data[0].resultStorage>=0);REQUIRE(plan.calls.data[1].arguments.data[0].type.category.tag==4);
 REQUIRE(!run_plan(src,ORD_Isolation));REQUIRE(plan.callCount==1);concept_aggregate_argument_plan isolated=plan.calls.data[0].arguments.data[0];REQUIRE(isolated.type.category.tag==4 && isolated.storage!=placement.values.data[isolated.value]);
 /* Replay the actual planned typed transfers and finalized frame offsets. */
 unsigned char callerMemory[4096],calleeMemory[4096];memset(callerMemory,0xa5,sizeof callerMemory);memset(calleeMemory,0x5a,sizeof calleeMemory);
 double initial[3]={1,2,3};int sourceOffset=frame.slotOffsets.data[placement.values.data[isolated.value]],tempOffset=frame.slotOffsets.data[isolated.storage];memcpy(callerMemory+sourceOffset,initial,sizeof initial);
 int copies=0;for(int i=0;i<plan.transferCount;i++){concept_aggregate_transfer x=plan.transfers.data[i];if(x.kind.tag==2){REQUIRE(x.storage==isolated.storage && x.sourceSlot==placement.values.data[isolated.value]);REQUIRE(!field_copy(&reader,document,x.type,callerMemory+frame.slotOffsets.data[x.storage],callerMemory+frame.slotOffsets.data[x.sourceSlot],0));copies++;}}REQUIRE(copies==1);
 REQUIRE(!run_plan(src,ORD_MutateValue));copies=0;int parameterOffset=-1;
 for(int i=0;i<plan.transferCount;i++){concept_aggregate_transfer x=plan.transfers.data[i];if(x.kind.tag==1){parameterOffset=frame.slotOffsets.data[x.storage];REQUIRE(!field_copy(&reader,document,x.type,calleeMemory+parameterOffset,callerMemory+tempOffset,0));copies++;}}REQUIRE(copies==1 && parameterOffset>=0);
 double changed=91;memcpy(calleeMemory+parameterOffset,&changed,sizeof changed);REQUIRE(!memcmp(callerMemory+sourceOffset,initial,sizeof initial));REQUIRE(!memcmp(callerMemory+tempOffset,initial,sizeof initial));
 REQUIRE(!run_plan(src,ORD_RoundPadded));REQUIRE(plan.parameters.data[0].type.size==16);unsigned char padded[16]={0};padded[0]=3;memset(padded+8,0x19,8);memset(calleeMemory,0x5a,sizeof calleeMemory);copies=0;
 for(int i=0;i<plan.transferCount;i++){concept_aggregate_transfer x=plan.transfers.data[i];if(x.kind.tag==1){unsigned char *destination=calleeMemory+frame.slotOffsets.data[x.storage];REQUIRE(!field_copy(&reader,document,x.type,destination,padded,0));REQUIRE(destination[0]==3 && !memcmp(destination+8,padded+8,8));for(int j=1;j<8;j++)REQUIRE(destination[j]==0x5a);copies++;}}REQUIRE(copies==1);
 /* Every malformed derived plan is checked against the admitted artifact. */
 REQUIRE(!run_plan(src,ORD_Shifted));plan.parameters.data[1].physicalIndex=1;REQUIRE(!validate_plan(&reader,document,ORD_Shifted,"AMD64_AGGREGATE_ABI_INVALID_POSITION"));
 REQUIRE(!run_plan(src,ORD_Shifted));plan.result.category.tag=5;REQUIRE(!validate_plan(&reader,document,ORD_Shifted,"AMD64_AGGREGATE_ABI_INVALID_DIRECT_WIDTH"));
 REQUIRE(!run_plan(src,ORD_RoundFour));plan.result.category.tag=6;plan.result.carrier.tag=12;REQUIRE(!validate_plan(&reader,document,ORD_RoundFour,"AMD64_AGGREGATE_ABI_RETURN_INELIGIBLE"));
 REQUIRE(!run_plan(src,ORD_Shifted));plan.hiddenPointer=-1;REQUIRE(!validate_plan(&reader,document,ORD_Shifted,"AMD64_AGGREGATE_ABI_INVALID_HIDDEN_RESULT"));
 REQUIRE(!run_plan(src,ORD_Driver));plan.calls.data[0].physical.scalar.destinations.data[2].xmm.tag=1;REQUIRE(!validate_plan(&reader,document,ORD_Driver,"AMD64_AGGREGATE_ABI_INVALID_POSITION"));
 REQUIRE(!run_plan(src,ORD_Isolation));plan.calls.data[0].arguments.data[0].storage=placement.values.data[plan.calls.data[0].arguments.data[0].value];REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_INVALID_TEMPORARY"));
 REQUIRE(!run_plan(src,ORD_Isolation));plan.calls.data[0].arguments.data[0].type.category.tag=2;REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_STALE_LAYOUT"));
 REQUIRE(!run_plan(src,ORD_Isolation));plan.calls.data[0].arguments.data[0].end=plan.calls.data[0].position;REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_TEMPORARY_LIFETIME"));
 REQUIRE(!run_plan(src,ORD_Isolation));plan.transfers.data[0].sourceSlot=plan.transfers.data[0].storage;REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_TEMPORARY_LIFETIME"));
 REQUIRE(!run_plan(src,ORD_Isolation));plan.callCount=33;REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED"));
 REQUIRE(!run_plan(src,ORD_Driver));frame.slots.data[frame.slotCount-1].offset=0;REQUIRE(!validate_plan(&reader,document,ORD_Driver,"AMD64_AGGREGATE_ABI_INVALID_FRAME"));
 REQUIRE(!run_plan(src,ORD_Driver));plan.calls.data[0].physical.stackArgumentBytes=0;REQUIRE(!validate_plan(&reader,document,ORD_Driver,"AMD64_AGGREGATE_ABI_INVALID_POSITION"));
 REQUIRE(!run_plan(src,ORD_Driver));plan.calls.data[0].hiddenPointer=-1;REQUIRE(!validate_plan(&reader,document,ORD_Driver,"AMD64_AGGREGATE_ABI_INVALID_HIDDEN_RESULT"));
 REQUIRE(!run_plan(src,ORD_Isolation));function.slots.data[placement.argumentTemps.data[0]].alignment=8;REQUIRE(!validate_plan(&reader,document,ORD_Isolation,"AMD64_AGGREGATE_ABI_TEMPORARY_ALIGNMENT"));
 REQUIRE(!run_plan(src,ORD_RoundFour));plan.parameters.data[0].type.layout=plan.parameters.data[0].type.identity;REQUIRE(!validate_plan(&reader,document,ORD_RoundFour,"AMD64_AGGREGATE_ABI_STALE_LAYOUT"));
 concept_result_aggregate_abitype_aggregate_abierror missing=F(classify_win64object)(&reader,document,-1,(concept_object_value_class){1},0);REQUIRE(missing.tag && !strcmp(F(describe_aggregate_abierror)(missing.payload.error.error),"AMD64_AGGREGATE_ABI_MISSING_LAYOUT"));
 REQUIRE(!run_plan(src,ORD_DirectChain));REQUIRE(plan.callCount==2 && plan.calls.data[0].result.category.tag==5);int captures=0;for(int i=0;i<plan.transferCount;i++)if(plan.transfers.data[i].kind.tag==5)captures++;REQUIRE(captures==2);
 REQUIRE(!run_plan(src,ORD_StackArguments));REQUIRE(plan.callCount==1);REQUIRE(plan.calls.data[0].arguments.data[4].type.category.tag==3 && plan.calls.data[0].arguments.data[4].physicalIndex==4);REQUIRE(plan.calls.data[0].arguments.data[5].type.category.tag==4 && plan.calls.data[0].arguments.data[5].physicalIndex==5);
 REQUIRE(!run_plan(src,ORD_CallMixed));REQUIRE(plan.calls.data[0].physical.argumentCount==8 && plan.calls.data[0].physical.stackArgumentBytes==32);REQUIRE(plan.calls.data[0].arguments.data[6].type.category.tag==2 && plan.calls.data[0].arguments.data[7].type.category.tag==2);
 concept_result_wire_object_function_aggregate_abierror driver=F(aggregate_plan_owner)(&reader,document,ORD_Driver);REQUIRE(!driver.tag);
 clock_t begin=clock();int classified=0;for(int run=0;run<100;run++)for(int i=0;i<document.Types.count;i++){concept_result_wire_object_type_aggregate_abierror raw=F(aggregate_plan_type)(&reader,document,i);REQUIRE(!raw.tag);if(raw.payload.ok.value.Kind.tag==12){concept_result_aggregate_abitype_aggregate_abierror r=F(classify_win64object)(&reader,document,i,(concept_object_value_class){1},0);REQUIRE(!r.tag);classified++;}}
 double classificationMs=1000.0*(double)(clock()-begin)/CLOCKS_PER_SEC;
 begin=clock();for(int run=0;run<100;run++){concept_result_void_backend_error r=F(project_object_function_abi)(&reader,document,driver.payload.ok.value,&function,&placement,1);REQUIRE(!r.tag);}double projectionMs=1000.0*(double)(clock()-begin)/CLOCKS_PER_SEC;
 begin=clock();for(int run=0;run<100;run++){concept_result_void_aggregate_abierror r=F(prepare_aggregate_function)(&reader,document,driver.payload.ok.value,&function,&placement,&allocation,&frame,&plan);REQUIRE(!r.tag);}double planningMs=1000.0*(double)(clock()-begin)/CLOCKS_PER_SEC;
 printf("timing classifications=%d classification-ms=%.3f projection-with-temporaries-100-ms=%.3f full-call-frame-plans-100-ms=%.3f resident-plan-bytes=%zu active-call-bytes=%zu Driver-frame=%d\n",classified,classificationMs,projectionMs,planningMs,sizeof plan,(size_t)plan.callCount*sizeof(concept_aggregate_call_plan),frame.byteCount);
 concept_native_image image,before;memset(&image,0x5a,sizeof image);before=image;memset(output,0xa5,sizeof output);
 concept_result_int_native_error emitted=F(emit_module)(src,dst,&image);REQUIRE(!emitted.tag && image.finalized && image.symbolCount==FUNCTION_COUNT);
 image=before;memset(output,0xa5,sizeof output);concept_span_byte tiny={output,1};emitted=F(emit_module)(src,tiny,&image);REQUIRE(emitted.tag && !memcmp(&image,&before,sizeof image));for(size_t i=0;i<sizeof output;i++)REQUIRE(output[i]==0xa5);
 puts("100 artifact-only ABI/frame/move/transfer plans per function; native emission now succeeds; capacity failure is atomic");return 0;
}
`
