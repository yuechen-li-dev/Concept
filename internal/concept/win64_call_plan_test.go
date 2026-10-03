package concept

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestEVT2e3Win64PlansMovesAndLiveness(t *testing.T) {
	fixture := "testdata/evt2e3_calls.concept"
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
	if MachineBridgeVersion != 3 || MachineBridgeSchemaHash != "3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312" {
		t.Fatal("ABI derivation changed semantic bridge")
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
	harness := strings.Replace(win64CallPlanHarness, "/* ARTIFACT */", data.String(), 1)
	var normalPlans string
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		outputs := backendTestOutputs(t, verify)
		t.Run(mode, func(t *testing.T) {
			output := runFoundationNativeHarnessOutput(t, outputs, "win64_plans.c", harness, "-pedantic-errors")
			var plans []string
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "call site=") {
					plans = append(plans, strings.TrimSpace(line))
				}
			}
			if len(plans) != 15 {
				t.Fatalf("missing plan observations: %d", len(plans))
			}
			canonical := strings.Join(plans, "\n")
			if !verify {
				normalPlans = canonical
			} else if canonical != normalPlans {
				t.Fatal("Normal/Verify ABI plans, preservation or debug output disagree")
			}
		})
	}
}

// This C11 host supplies fixtures, replays move values and renders observations.
// It contains no ABI placement, clobber, preservation or cycle-breaking policy.
const win64CallPlanHarness = `#include "amd64.generated.h"
#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#define F(name) concept_standard__backend__amd64_##name
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"failed line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
#define concept_register_make_rax() ((concept_register){.tag=0})
#define concept_register_make_rbx() ((concept_register){.tag=1})
#define concept_register_make_rcx() ((concept_register){.tag=2})
#define concept_register_make_rdx() ((concept_register){.tag=3})
#define concept_register_make_rsp() ((concept_register){.tag=7})
#define concept_register_make_r8() ((concept_register){.tag=8})
#define concept_register_make_r9() ((concept_register){.tag=9})
#define concept_register_make_r12() ((concept_register){.tag=12})
#define concept_call_value_type_make_bool() ((concept_call_value_type){.tag=1})
#define concept_call_value_type_make_i32() ((concept_call_value_type){.tag=6})
#define concept_call_value_type_make_u64() ((concept_call_value_type){.tag=9})
#define concept_abiclass_make_integer() ((concept_abiclass){.tag=0})
#define concept_abiclass_make_bool() ((concept_abiclass){.tag=1})
#define concept_abiclass_make_address() ((concept_abiclass){.tag=2})
#define concept_abiclass_make_float() ((concept_abiclass){.tag=3})
#define concept_preservation_class_make_safe_in_callee_saved() ((concept_preservation_class){.tag=1})
#define concept_opcode_make_call() ((concept_opcode){.tag=16})
/* ARTIFACT */
static concept_call_request request(int count,int result) {
 concept_call_request r=F(empty_call_request)();r.argumentCount=count;r.instruction=4;
 for(int i=0;i<count;++i) r.arguments.data[i]=(concept_abivalue){i,concept_call_value_type_make_i32(),concept_abiclass_make_integer(),4};
 r.hasResult=result!=0;
 if(result)r.result=(concept_abivalue){100,concept_call_value_type_make_i32(),concept_abiclass_make_integer(),4};
 return r;
}
static int describe(char *text,const concept_call_plan *p) {
 int n=sprintf(text,"call site=%d argc=%d return=%d:v%d:w%d shadow=%d stack=%d flags=%d temps=%d moves=%d",p->instruction,p->argumentCount,p->hasResult,p->result.value,p->result.width,p->shadowSpaceBytes,p->stackArgumentCount,p->flagsClobbered,p->temporaryCount,p->moveCount);
 for(int i=0;i<p->argumentCount;++i){concept_argument_placement a=p->arguments.data[i];n+=sprintf(text+n," arg%d=v%d:%u:w%d:%u:%d:%d",a.index,a.source.value,a.source.type.tag,a.source.width,a.physical.tag,a.onStack,a.stackSlot);}
 for(int i=0;i<p->moveCount;++i){concept_argument_move m=p->moves.data[i];n+=sprintf(text+n," move=%u:%u:%d>%u:%u:%d:w%d",m.sourceKind.tag,m.source.tag,m.sourceSlot,m.destinationKind.tag,m.destination.tag,m.destinationSlot,m.width);}
 return n;
}
// Independent value oracle: moves are executed as width-limited assignments.
static int replay(const concept_call_plan *p,const concept_array_8_register *sources) {
 uint64_t reg[16],original[16],stack[4]={0},temp[4]={0};
 for(int i=0;i<16;++i)reg[i]=original[i]=UINT64_C(0xabcd010203040500)+(uint64_t)i;
 for(int i=0;i<p->moveCount;++i){concept_argument_move m=p->moves.data[i];uint64_t value=m.sourceKind.tag==0?reg[m.source.tag]:temp[m.sourceSlot];uint64_t mask=m.width==8?UINT64_MAX:((UINT64_C(1)<<(m.width*8))-1);value&=mask;
  if(m.destinationKind.tag==0)reg[m.destination.tag]=value;else if(m.destinationKind.tag==1)stack[m.destinationSlot]=value;else temp[m.destinationSlot]=value;
 }
 for(int i=0;i<p->argumentCount;++i){concept_argument_placement a=p->arguments.data[i];uint64_t mask=a.source.width==8?UINT64_MAX:((UINT64_C(1)<<(a.source.width*8))-1);uint64_t actual=a.onStack?stack[a.stackSlot]:reg[a.physical.tag];if((actual&mask)!=(original[sources->data[i].tag]&mask))return 0;}
 return 1;
}
int main(void){
 concept_abiregister_tables tables=F(win64registers)();
 REQUIRE(tables.integerArguments.data[0].tag==2&&tables.integerArguments.data[1].tag==3&&tables.integerArguments.data[2].tag==8&&tables.integerArguments.data[3].tag==9);
 const unsigned caller[]={0,2,3,8,9,10,11},callee[]={1,6,4,5,12,13,14,15};
 for(int i=0;i<7;++i)REQUIRE(tables.callerSaved.data[i].tag==caller[i]);
 for(int i=0;i<8;++i)REQUIRE(tables.calleeSaved.data[i].tag==callee[i]);
 REQUIRE(!F(is_value_register)(concept_register_make_rsp()));
 concept_call_plan p=F(empty_call_plan)();
 concept_array_8_register sources={0};
 for(int i=0;i<8;++i)sources.data[i]=concept_register_make_rax();
 for(int count=0;count<=8;++count){concept_call_request r=request(count,1);REQUIRE(!F(plan_win64call)(&r,&p).tag);REQUIRE(!F(validate_call_plan)(&p).tag);REQUIRE(p.shadowSpaceBytes==32&&p.resultPhysical.tag==0&&p.hasResult&&p.result.width==4);
  REQUIRE(p.stackArgumentCount==(count>4?count-4:0));
  for(int i=0;i<count;++i){REQUIRE(p.arguments.data[i].source.value==i);if(i<4)REQUIRE(p.arguments.data[i].physical.tag==tables.integerArguments.data[i].tag);else REQUIRE(p.arguments.data[i].onStack&&p.arguments.data[i].stackSlot==i-4);}
  if(count==0||count==4||count==5||count==6){char text[4096];describe(text,&p);puts(text);}
 }
 concept_call_request r=request(1,0);REQUIRE(!F(plan_win64call)(&r,&p).tag&&!p.hasResult&&p.result.value==-1);
 r.arguments.data[0]=(concept_abivalue){0,concept_call_value_type_make_bool(),concept_abiclass_make_bool(),1};REQUIRE(!F(plan_win64call)(&r,&p).tag&&p.arguments.data[0].source.width==1);
 r.arguments.data[0]=(concept_abivalue){0,concept_call_value_type_make_u64(),concept_abiclass_make_address(),8};REQUIRE(!F(plan_win64call)(&r,&p).tag&&p.arguments.data[0].source.category.tag==2&&p.arguments.data[0].physical.tag==2);
 for(unsigned type=2;type<=9;++type){r.arguments.data[0].category=concept_abiclass_make_integer();r.arguments.data[0].type.tag=type;r.arguments.data[0].width=type<=3?1:type<=5?2:type<=7?4:8;REQUIRE(!F(plan_win64call)(&r,&p).tag);REQUIRE(p.arguments.data[0].source.type.tag==type&&p.arguments.data[0].source.width==r.arguments.data[0].width);}
 for(unsigned category=3;category<=4;++category){r=request(1,0);r.arguments.data[0].category.tag=category;REQUIRE(F(plan_win64call)(&r,&p).tag);}
 for(unsigned form=1;form<=3;++form){r=request(0,0);r.form.tag=form;REQUIRE(F(plan_win64call)(&r,&p).tag);}
 r=request(0,0);r.convention.tag=99;REQUIRE(F(plan_win64call)(&r,&p).tag);
 r=request(0,1);r.result.category=concept_abiclass_make_float();REQUIRE(F(plan_win64call)(&r,&p).tag);
 r=request(0,0);r.argumentCount=9;REQUIRE(F(plan_win64call)(&r,&p).tag);
 r=request(5,1);REQUIRE(!F(plan_win64call)(&r,&p).tag);
 concept_call_plan bad=p;bad.arguments.data[1].physical=bad.arguments.data[0].physical;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.arguments.data[0].source.value=-1;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.arguments.data[0].physical=concept_register_make_rbx();REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.resultPhysical=concept_register_make_rdx();REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.shadowSpaceBytes=0;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.arguments.data[4].stackSlot=1;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.flagsClobbered=0;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.stackArgumentBytes=0;REQUIRE(F(validate_call_plan)(&bad).tag);
 bad=p;bad.arguments.data[0].physical.tag=99;REQUIRE(F(validate_call_plan)(&bad).tag);
 // Acyclic.
 r=request(2,1);REQUIRE(!F(plan_win64call)(&r,&p).tag);sources.data[0]=concept_register_make_rax();sources.data[1]=concept_register_make_rbx();REQUIRE(!F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);REQUIRE(p.moveCount==2&&p.temporaryCount==0&&p.moves.data[0].source.tag==0&&p.moves.data[0].destination.tag==2);REQUIRE(!F(validate_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&replay(&p,&sources));
 // Two-way and three-way cycles, mixed widths and repeat-source fanout.
 for(int count=2;count<=3;++count){r=request(count,1);REQUIRE(!F(plan_win64call)(&r,&p).tag);for(int i=0;i<count;++i)sources.data[i]=tables.integerArguments.data[(i+1)%count];REQUIRE(!F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);REQUIRE(p.temporaryCount==1&&p.moveCount==count+1&&p.moves.data[0].destinationKind.tag==2&&p.moves.data[0].width==8);REQUIRE(!F(validate_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&replay(&p,&sources));char baseline[4096],next[4096];describe(baseline,&p);puts(baseline);for(int run=0;run<100;++run){concept_call_plan q=F(empty_call_plan)();REQUIRE(!F(plan_win64call)(&r,&q).tag&&!F(plan_argument_moves)(&q,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);describe(next,&q);REQUIRE(!strcmp(baseline,next));}bad=p;bad.moveCount=0;REQUIRE(F(validate_argument_moves)(&bad,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);}
 r=request(2,0);r.arguments.data[0]=(concept_abivalue){0,concept_call_value_type_make_bool(),concept_abiclass_make_bool(),1};r.arguments.data[1]=(concept_abivalue){1,concept_call_value_type_make_u64(),concept_abiclass_make_integer(),8};REQUIRE(!F(plan_win64call)(&r,&p).tag);sources.data[0]=concept_register_make_rdx();sources.data[1]=concept_register_make_rcx();REQUIRE(!F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&!F(validate_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&replay(&p,&sources));
 r=request(5,0);r.arguments.data[4].value=1;REQUIRE(!F(plan_win64call)(&r,&p).tag);sources.data[0]=concept_register_make_rdx();sources.data[1]=concept_register_make_rcx();sources.data[2]=concept_register_make_r8();sources.data[3]=concept_register_make_r9();sources.data[4]=concept_register_make_rcx();REQUIRE(!F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&p.moves.data[0].destinationKind.tag==1&&!F(validate_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&replay(&p,&sources));
 // Exhaust every source combination in the four ABI argument registers.
 clock_t moveStart=clock();
 for(int a=0;a<4;++a)for(int b=0;b<4;++b)for(int c=0;c<4;++c)for(int d=0;d<4;++d){int ids[]={a,b,c,d};r=request(4,0);for(int i=0;i<4;++i){sources.data[i]=tables.integerArguments.data[ids[i]];r.arguments.data[i].value=ids[i];}REQUIRE(!F(plan_win64call)(&r,&p).tag&&!F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&!F(validate_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag&&replay(&p,&sources));}
 printf("256 source/cycle/fanout move plans %.3f ms\n",1000.0*(clock()-moveStart)/CLOCKS_PER_SEC);
 r=request(2,0);REQUIRE(!F(plan_win64call)(&r,&p).tag);sources.data[0]=sources.data[1]=concept_register_make_rcx();REQUIRE(F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);sources.data[0]=concept_register_make_rsp();REQUIRE(F(plan_argument_moves)(&p,(concept_ref_const_array_1_register){.data=sources.data,.shape={8}}).tag);
 concept_preservation invalid={0,4,0,concept_register_make_rax(),5,0,10,concept_preservation_class_make_safe_in_callee_saved()};REQUIRE(F(validate_preservation)(invalid).tag);
 concept_readonly_span_byte input={artifact,sizeof artifact};
 int ordinals[]={ORD_Calls0,ORD_Calls1,ORD_Calls4,ORD_Calls5,ORD_Calls6,ORD_CallsVoid,ORD_CallsBool,ORD_Repeated,ORD_Across};
 clock_t planningStart=clock();
 for(int specimen=0;specimen<9;++specimen){char baseline[8192]={0};for(int run=0;run<100;++run){concept_backend_function function=F(empty_function)();REQUIRE(!F(decode_function)(input,ordinals[specimen],&function).tag&&!F(layout_lowered_blocks)(&function).tag);concept_allocation allocation=F(empty_allocation)();REQUIRE(!F(allocate_registers)(&function,&allocation).tag&&!F(plan_function_calls)(&function,&allocation).tag);concept_liveness live=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&function,&live).tag);
  int call=-1;for(int i=0;i<function.instructionCount;++i)if(function.instructions.data[i].call.target.length)call=i;REQUIRE(call>=0);
  concept_preservation records[512];concept_span_preservation out={records,512};int count=0;REQUIRE(!F(plan_allocated_call)(&function,&allocation,&live,call,&p,out,&count).tag);char next[8192];int n=describe(next,&p);for(int i=0;i<count;++i)n+=sprintf(next+n," live=%d:%d:%d:%u:%u:%d:%d",records[i].value,records[i].width,records[i].address,records[i].physical.tag,records[i].classification.tag,records[i].start,records[i].end);for(int i=0;i<15;++i)n+=sprintf(next+n," saved=%d",allocation.usedCalleeSaved.data[i]);
  if(run==0){strcpy(baseline,next);puts(next);}else REQUIRE(!strcmp(baseline,next));
  if(specimen==4)REQUIRE(p.stackArgumentCount==2&&allocation.usedCalleeSaved.data[F(register_index)(concept_register_make_rbx())]);
  if(specimen==5)REQUIRE(!p.hasResult);
  if(specimen==6)REQUIRE(p.result.width==1&&p.arguments.data[0].source.width==1);
  if(specimen==8){REQUIRE(count>0&&allocation.preservationCount>0);for(int i=0;i<count;++i)REQUIRE(records[i].value!=p.result.value&&records[i].classification.tag==0);int id=records[0].value;allocation.physical.data[id]=concept_register_make_r12();REQUIRE(!F(plan_function_calls)(&function,&allocation).tag);REQUIRE(allocation.usedCalleeSaved.data[F(register_index)(concept_register_make_r12())]);REQUIRE(!F(plan_allocated_call)(&function,&allocation,&live,call,&p,out,&count).tag);int found=0;for(int i=0;i<count;++i)if(records[i].value==id){REQUIRE(records[i].classification.tag==1);found=1;}REQUIRE(found);concept_span_preservation empty={records,0};REQUIRE(F(plan_allocated_call)(&function,&allocation,&live,call,&p,empty,&count).tag);}
  if(specimen==8&&run==0){clock_t begin=clock();for(int k=0;k<1000;++k){concept_liveness measured=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&function,&measured).tag);}double aware=1000.0*(clock()-begin)/CLOCKS_PER_SEC;int argc=function.instructions.data[call].call.argumentCount;function.instructions.data[call].call.argumentCount=0;begin=clock();for(int k=0;k<1000;++k){concept_liveness measured=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&function,&measured).tag);}double control=1000.0*(clock()-begin)/CLOCKS_PER_SEC;function.instructions.data[call].call.argumentCount=argc;printf("1000 liveness passes call-aware=%.3f ms call-uses-disabled-control=%.3f ms delta=%.3f ms\n",aware,control,aware-control);}
  unsigned char bytes[1024];memset(bytes,0x5a,sizeof bytes);concept_span_byte target={bytes,sizeof bytes};REQUIRE(F(emit_function)(input,ordinals[specimen],target).tag);for(unsigned i=0;i<sizeof bytes;++i)REQUIRE(bytes[i]==0x5a);
 }}
 printf("900 artifact ABI/allocation/liveness plans %.3f ms\n",1000.0*(clock()-planningStart)/CLOCKS_PER_SEC);
 concept_backend_function flags=F(empty_function)();flags.blockCount=1;flags.instructionCount=2;flags.blocks.data[0].instructionCount=2;flags.blocks.data[0].flagsUse=0;flags.instructions.data[0]=F(empty_instruction)();flags.instructions.data[0].flagsDef=0;flags.instructions.data[1]=F(empty_instruction)();flags.instructions.data[1].op=concept_opcode_make_call();REQUIRE(F(validate_call_flags)(&flags).tag);flags.blocks.data[0].flagsUse=-1;REQUIRE(!F(validate_call_flags)(&flags).tag);
 clock_t abiStart=clock();r=request(6,1);for(int i=0;i<10000;++i)REQUIRE(!F(plan_win64call)(&r,&p).tag);printf("10000 ABI plans %.3f ms; sizes request=%zu plan=%zu placement=%zu move=%zu preservation=%zu backend=%zu allocation=%zu\n",1000.0*(clock()-abiStart)/CLOCKS_PER_SEC,sizeof r,sizeof p,sizeof(concept_argument_placement),sizeof(concept_argument_move),sizeof(concept_preservation),sizeof(concept_backend_function),sizeof(concept_allocation));
 return 0;
}
`
