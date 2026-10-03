package concept

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestEVT2e4FrameRealization(t *testing.T) {
	path := "../../libraries/Standard/Backend/AMD64.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	fixture := "testdata/evt2e4_frames.concept"
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
	harness := strings.Replace(frameRealizationHarness, "/* ARTIFACT */", data.String(), 1)
	var normal string
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		policy := ConservativeCompilationPolicy()
		if verify {
			mode = "Verify"
			policy = VerifyCompilationPolicy()
		}
		outputs, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(mode, func(t *testing.T) {
			output := runFoundationNativeHarnessOutput(t, outputs, "frames.c", harness, "-pedantic-errors")
			var observations []string
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "frame ") {
					observations = append(observations, strings.TrimSpace(line))
				}
			}
			if len(observations) != 8 {
				t.Fatalf("frame observations=%d", len(observations))
			}
			canonical := strings.Join(observations, "\n")
			if !verify {
				normal = canonical
			} else if normal != canonical {
				t.Fatal("Normal/Verify concrete frames/actions disagree")
			}
		})
	}
}

// Test host: checked-input orchestration and independent concrete value oracle.
// All storage, offsets, ordering and inserted actions come from Concept.
const frameRealizationHarness = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#define F(name) concept_standard__backend__amd64_##name
#define REQUIRE(c) do { if(!(c)){fprintf(stderr,"failed line %d: %s\n",__LINE__,#c);return __LINE__;} } while(0)
/* ARTIFACT */
static int dump(char *out,const concept_finalized_frame *f,const concept_call_site_actions *sites,int n) {
 int k=sprintf(out,"frame size=%d used=%d pad=%d returns=%d",f->byteCount,f->unalignedSize,f->padding,f->returnCount); for(int i=0;i<f->returnCount;++i)k+=sprintf(out+k," ret=%d",f->returnBlocks.data[i]);
 for(int i=0;i<7;++i)k+=sprintf(out+k," r=%u:%d:%d",f->regions.data[i].category.tag,f->regions.data[i].offset,f->regions.data[i].size);
 for(int i=0;i<f->slotCount;++i){concept_frame_slot s=f->slots.data[i];k+=sprintf(out+k," s=%u:%d:%d:%d:%d",s.category.tag,s.identity,s.offset,s.size,s.alignment);}
 for(int phase=0;phase<2;++phase){const concept_frame_action *a=phase?f->epilogue.data:f->prologue.data;int count=phase?f->epilogueCount:f->prologueCount;for(int i=0;i<count;++i)k+=sprintf(out+k," e%d=%u:%u:%u:%d:%d:%d",phase,a[i].kind.tag,a[i].source.tag,a[i].destination.tag,a[i].slot,a[i].offset,a[i].width);}
 for(int j=0;j<n;++j){k+=sprintf(out+k," call=%d:%d:%d",sites[j].abi.instruction,sites[j].count,sites[j].callIndex);for(int i=0;i<sites[j].count;++i){concept_frame_action a=sites[j].actions.data[i];k+=sprintf(out+k," a=%u:%u:%u:%d:%d:%d:%d",a.kind.tag,a.source.tag,a.destination.tag,a.slot,a.offset,a.width,a.value);}}
 return k;
}
static uint64_t mask(int w){return w==8?UINT64_MAX:(UINT64_C(1)<<(w*8))-1;}
// Byte-addressed stack oracle intentionally independent of slot-token replay.
static void store(unsigned char *s,int offset,uint64_t value,int width){for(int i=0;i<width;++i)s[offset+i]=(unsigned char)(value>>(8*i));}
static uint64_t load(const unsigned char *s,int offset,int width){uint64_t value=0;for(int i=0;i<width;++i)value|=(uint64_t)s[offset+i]<<(8*i);return value;}
static int replay(const concept_finalized_frame *f,const concept_call_site_actions *site,const concept_allocation *alloc,const concept_preservation *records,int count) {
 unsigned char stack[65536]={0}; uint64_t reg[16],original[16]; int rsp=0;
 for(int i=0;i<16;++i)reg[i]=original[i]=UINT64_C(0xabcd010203040500)+(uint64_t)i;
 for(int i=0;i<f->prologueCount;++i){concept_frame_action a=f->prologue.data[i];if(a.kind.tag==0)rsp-=a.offset;else if(a.kind.tag==1)store(stack,a.offset,reg[a.source.tag],a.width);else return 0;}
 if((8+rsp)%16!=0)return 0;
 for(int i=0;i<site->count;++i){concept_frame_action a=site->actions.data[i];
  if(a.kind.tag==3)store(stack,a.offset,reg[a.source.tag],a.width);
  else if(a.kind.tag==4)reg[a.destination.tag]=load(stack,a.offset,a.width);
  else if(a.kind.tag==5)reg[a.destination.tag]=reg[a.source.tag]&mask(a.width);
  else if(a.kind.tag==6){
   for(int j=0;j<site->abi.argumentCount;++j){concept_argument_placement p=site->abi.arguments.data[j];uint64_t v=p.onStack?load(stack,32+p.stackSlot*8,p.source.width):reg[p.physical.tag];if((v&mask(p.source.width))!=(original[alloc->physical.data[p.source.value].tag]&mask(p.source.width)))return 0;}
   const int caller[]={0,2,3,8,9,10,11};for(int j=0;j<7;++j)reg[caller[j]]=UINT64_C(0xdeadbeef);
   if(site->abi.hasResult)reg[0]=UINT64_C(0x12345678)&mask(site->abi.result.width);
  }else return 0;
 }
 for(int i=0;i<count;++i)if((reg[records[i].physical.tag]&mask(records[i].width))!=(original[records[i].physical.tag]&mask(records[i].width)))return 0;
 if(site->abi.hasResult && (reg[alloc->physical.data[site->abi.result.value].tag]&mask(site->abi.result.width))!=(UINT64_C(0x12345678)&mask(site->abi.result.width)))return 0;
 // Model ordinary function computation changing each saved GPR, then all
 // return paths use the verified epilogue and leave RAX's return value intact.
 for(int i=0;i<f->slotCount;++i)if(f->slots.data[i].category.tag==3){int idx=f->slots.data[i].identity;concept_abiregister_tables t=F(win64registers)();reg[t.calleeSaved.data[idx-7].tag]=0;}
 reg[0]=UINT64_C(0xabc123);
 for(int i=0;i<f->epilogueCount;++i){concept_frame_action a=f->epilogue.data[i];if(a.kind.tag==2)reg[a.destination.tag]=load(stack,a.offset,a.width);else if(a.kind.tag==7)rsp+=a.offset;else if(a.kind.tag!=8)return 0;}
 if(rsp || reg[0]!=UINT64_C(0xabc123))return 0;
 const int callee[]={1,6,4,5,12,13,14,15};for(int i=0;i<8;++i)if(reg[callee[i]]!=original[callee[i]])return 0;
 return 1;
}
int main(void) {
 concept_readonly_span_byte input={artifact,sizeof artifact};
 const int ords[]={ORD_Calls0,ORD_Calls5,ORD_Calls8,ORD_Across,ORD_Multi,ORD_Early,ORD_Pressure,ORD_VoidCall};
 int maximum=0;
 for(int specimen=0;specimen<8;++specimen){
  concept_backend_function function=F(empty_function)(); REQUIRE(!F(decode_function)(input,ords[specimen],&function).tag && !F(layout_lowered_blocks)(&function).tag);
  concept_allocation allocation=F(empty_allocation)(); REQUIRE(!F(allocate_registers)(&function,&allocation).tag && !F(plan_function_calls)(&function,&allocation).tag);
  concept_liveness live=F(empty_liveness)(); REQUIRE(!F(compute_liveness)(&function,&live).tag);
  concept_finalized_frame frame=F(empty_finalized_frame)(); concept_result_void_frame_error q=F(qualify_function_frame)(&function,&allocation,&frame); if(q.tag) fprintf(stderr,"specimen=%d frame error=%s hasCalls=%d input-shadow=%d allocated-shadow=%d\\n",specimen,F(describe_frame_error)(q.payload.error.error),function.hasCalls,function.frameShadowSpace,allocation.shadowSpaceBytes); REQUIRE(!q.tag);
  REQUIRE(frame.byteCount%16==8 && frame.regions.data[0].size==32);
  if(specimen==0)REQUIRE(frame.byteCount==40 && frame.regions.data[4].size==0);
  if(specimen==1)REQUIRE(frame.regions.data[1].size==8 && frame.slots.data[frame.outgoingSlots.data[0]].offset==32);
  if(specimen==2||specimen==4)REQUIRE(frame.regions.data[1].size==32 && frame.slots.data[frame.outgoingSlots.data[3]].offset==56);
  if(specimen==3)REQUIRE(frame.spillSlots.data[4]>=0 && frame.slots.data[frame.spillSlots.data[4]].size==4);
  if(specimen==5)REQUIRE(frame.returnCount>=2);
  if(specimen==6)REQUIRE(frame.regions.data[4].size>=8);
  if(frame.byteCount>maximum)maximum=frame.byteCount;
  concept_call_site_actions sites[8]; int ncalls=0;
  for(int p=0;p<function.instructionCount;++p)if(function.instructions.data[p].op.tag==16){
   concept_call_plan abi=F(empty_call_plan)();concept_preservation records[512];int count=0;concept_span_preservation view={records,512};
   REQUIRE(!F(plan_allocated_call)(&function,&allocation,&live,p,&abi,view,&count).tag);
   REQUIRE(!F(realize_allocated_call)(&function,&allocation,&live,p,&frame,&sites[ncalls]).tag && replay(&frame,&sites[ncalls],&allocation,records,count));ncalls++;
  }
  char baseline[32768],next[32768];dump(baseline,&frame,sites,ncalls);puts(baseline);
  for(int run=0;run<100;++run){concept_finalized_frame f=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&function,&allocation,&f).tag);int n=0;for(int p=0;p<function.instructionCount;++p)if(function.instructions.data[p].op.tag==16){REQUIRE(!F(realize_allocated_call)(&function,&allocation,&live,p,&f,&sites[n]).tag);n++;}dump(next,&f,sites,n);REQUIRE(!strcmp(baseline,next));}
  concept_finalized_frame bad=frame;bad.byteCount++;REQUIRE(F(validate_frame)(&bad,1).tag);
  bad=frame;bad.regions.data[1].offset=0;REQUIRE(F(validate_frame)(&bad,1).tag);
  bad=frame;bad.epilogueCount--;REQUIRE(F(validate_frame)(&bad,1).tag);
  if(specimen==5){bad=frame;bad.returnBlocks.data[0]=-1;REQUIRE(F(validate_function_frame)(&function,&allocation,&live,&bad).tag);}
  if(frame.slotCount){bad=frame;bad.slots.data[0].alignment=3;REQUIRE(F(validate_frame)(&bad,1).tag);}
  if(specimen==3){bad=frame;bad.spillSlots.data[4]=-1;concept_call_site_actions site;int p=sites[0].abi.instruction;REQUIRE(F(realize_allocated_call)(&function,&allocation,&live,p,&bad,&site).tag);}
  if(specimen==4){bad=frame;int first=frame.spillSlots.data[0],second=frame.spillSlots.data[1];bad.slots.data[second].offset=bad.slots.data[first].offset;concept_result_void_frame_error error=F(validate_frame)(&bad,1);REQUIRE(error.tag && error.payload.error.error.tag==3);}
  if(specimen==0){concept_backend_function badInput=function;badInput.frameAlignment=3;REQUIRE(F(realize_frame)(&badInput,&allocation,&live,&bad).tag);badInput=function;badInput.slotCount=65;REQUIRE(F(realize_frame)(&badInput,&allocation,&live,&bad).tag);}
  unsigned char bytes[4096];memset(bytes,0x5a,sizeof bytes);concept_span_byte out={bytes,sizeof bytes};REQUIRE(F(emit_function)(input,ords[specimen],out).tag);for(int i=0;i<4096;++i)REQUIRE(bytes[i]==0x5a);
 }
 // All 256 source combinations, including repeated identities and disjoint
 // cycles, pass through the actual frame finalizer and concrete replay.
 for(int code=0;code<256;++code){
  concept_backend_function f=F(empty_function)();REQUIRE(!F(decode_function)(input,ORD_Calls8,&f).tag && !F(layout_lowered_blocks)(&f).tag);
  concept_allocation a=F(empty_allocation)();REQUIRE(!F(allocate_registers)(&f,&a).tag);
  concept_abiregister_tables table=F(win64registers)();
  for(int i=0;i<4;++i)a.physical.data[i]=table.integerArguments.data[i];
  int position=-1;for(int i=0;i<f.instructionCount;++i)if(f.instructions.data[i].op.tag==16)position=i;
  REQUIRE(position>=0);f.instructions.data[position].call.argumentCount=4;
  int digits=code;for(int i=0;i<4;++i){f.instructions.data[position].call.values.data[i]=digits%4;digits/=4;}
  REQUIRE(!F(plan_function_calls)(&f,&a).tag);
  concept_liveness live=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&f,&live).tag);
  concept_finalized_frame frame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&f,&a,&frame).tag);
  concept_call_site_actions site;REQUIRE(!F(realize_allocated_call)(&f,&a,&live,position,&frame,&site).tag);
  concept_preservation records[512];int count=0;concept_call_plan plan=F(empty_call_plan)();concept_span_preservation view={records,512};
  REQUIRE(!F(plan_allocated_call)(&f,&a,&live,position,&plan,view,&count).tag && replay(&frame,&site,&a,records,count));
  REQUIRE(plan.temporaryCount<=2);
  // Explicit RCX<->RDX and RCX->RDX->R8->RCX identities.
  if(code==225||code==201)REQUIRE(plan.temporaryCount==1);
  if(code==177)REQUIRE(plan.temporaryCount==2);
  if(plan.temporaryCount){concept_finalized_frame bad=frame;bad.tempSlots.data[0]=-1;REQUIRE(F(realize_allocated_call)(&f,&a,&live,position,&bad,&site).tag);}
  if(code==225||code==201||code==177){
   char baseline[32768],next[32768];REQUIRE(!F(realize_allocated_call)(&f,&a,&live,position,&frame,&site).tag);dump(baseline,&frame,&site,1);
   for(int run=0;run<100;++run){concept_finalized_frame nextFrame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&f,&a,&nextFrame).tag && !F(realize_allocated_call)(&f,&a,&live,position,&nextFrame,&site).tag);dump(next,&nextFrame,&site,1);REQUIRE(!strcmp(baseline,next));}
  }
 }
 // Natural-width preservation storage, including canonical address width.
 for(int width=1;width<=8;width*=2){
  concept_backend_function f=F(empty_function)();REQUIRE(!F(decode_function)(input,ORD_Across,&f).tag && !F(layout_lowered_blocks)(&f).tag);
  concept_allocation a=F(empty_allocation)();REQUIRE(!F(allocate_registers)(&f,&a).tag);
  f.virtualRegs.data[4].width=width;f.virtualRegs.data[4].address=width==8;
  concept_finalized_frame frame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&f,&a,&frame).tag);
  REQUIRE(frame.slots.data[frame.spillSlots.data[4]].size==width && frame.slots.data[frame.spillSlots.data[4]].alignment==width);
  concept_liveness live=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&f,&live).tag);int p=-1;for(int i=0;i<f.instructionCount;++i)if(f.instructions.data[i].op.tag==16)p=i;
  concept_call_plan plan=F(empty_call_plan)();concept_preservation records[512];int count=0;concept_span_preservation view={records,512};REQUIRE(!F(plan_allocated_call)(&f,&a,&live,p,&plan,view,&count).tag);
  concept_call_site_actions site;REQUIRE(!F(realize_allocated_call)(&f,&a,&live,p,&frame,&site).tag && replay(&frame,&site,&a,records,count));
 }
 // One callee-save primitive, preserving the allocated value without a spill.
 concept_backend_function f=F(empty_function)();REQUIRE(!F(decode_function)(input,ORD_Across,&f).tag && !F(layout_lowered_blocks)(&f).tag);
 concept_allocation a=F(empty_allocation)();REQUIRE(!F(allocate_registers)(&f,&a).tag);
 a.physical.data[4]=(concept_register){.tag=1};
 concept_liveness live=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&f,&live).tag);
 concept_finalized_frame frame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&f,&a,&frame).tag);
 REQUIRE(frame.regions.data[3].size==12 && frame.spillSlots.data[4]==-1 && frame.prologueCount==2 && frame.epilogueCount==3);
 concept_finalized_frame bad=frame;bad.prologue.data[1].source=(concept_register){.tag=0};REQUIRE(F(validate_frame)(&bad,1).tag);
 bad=frame;bad.epilogue.data[0].destination=(concept_register){.tag=0};REQUIRE(F(validate_frame)(&bad,1).tag);
 bad=frame;bad.slots.data[1].offset=bad.slots.data[0].offset;REQUIRE(F(validate_frame)(&bad,1).tag);
 REQUIRE(F(add_frame_bytes)(65528,1).tag && F(add_frame_bytes)(2147483647,1).tag && F(align_frame_bytes)(0,3).tag);
 int used=0;bad=F(empty_finalized_frame)();bad.slotCount=512;REQUIRE(F(append_frame_slot)(&bad,(concept_frame_region_kind){.tag=4},0,4,4,&used).tag);
 int position=-1;for(int i=0;i<f.instructionCount;++i)if(f.instructions.data[i].op.tag==16)position=i;
 concept_call_plan plan=F(empty_call_plan)();concept_preservation records[512];int count=0;concept_span_preservation view={records,512};
 REQUIRE(!F(plan_allocated_call)(&f,&a,&live,position,&plan,view,&count).tag);
 concept_call_site_actions site;REQUIRE(!F(realize_allocated_call)(&f,&a,&live,position,&frame,&site).tag && replay(&frame,&site,&a,records,count));
 concept_call_site_actions broken=site;broken.actions.data[broken.callIndex].kind=(concept_frame_action_kind){.tag=3};REQUIRE(F(replay_call_actions)(&broken,view,count,&a,&frame).tag);
 broken=site;broken.count=32;for(int i=site.count;i<32;++i)broken.actions.data[i]=site.actions.data[site.callIndex];REQUIRE(F(replay_call_actions)(&broken,view,count,&a,&frame).tag);
 broken=site;broken.actions.data[broken.count++]=(concept_frame_action){.kind={.tag=3},.source={.tag=0},.destination={.tag=0},.slot=0,.offset=frame.slots.data[0].offset,.width=4,.value=-1};REQUIRE(F(replay_call_actions)(&broken,view,count,&a,&frame).tag);
 f.virtualRegs.data[4].width=3;a.physical.data[4]=(concept_register){.tag=10};REQUIRE(F(qualify_function_frame)(&f,&a,&bad).tag);
 f.virtualRegs.data[4].width=4;
 clock_t start=clock();for(int i=0;i<1000;++i)REQUIRE(!F(realize_frame)(&f,&a,&live,&bad).tag);double frame_ms=1000.0*(clock()-start)/CLOCKS_PER_SEC;
 start=clock();for(int i=0;i<1000;++i)REQUIRE(!F(plan_allocated_call)(&f,&a,&live,position,&plan,view,&count).tag);double spill_ms=1000.0*(clock()-start)/CLOCKS_PER_SEC;
 start=clock();for(int i=0;i<1000;++i)REQUIRE(!F(lower_call_actions)(&plan,view,count,&a,&bad,&site).tag);double action_ms=1000.0*(clock()-start)/CLOCKS_PER_SEC;
 printf("1000 frame plans=%.3f ms spill requirements=%.3f ms action lowering=%.3f ms\n",frame_ms,spill_ms,action_ms);
 printf("qualified maximum frame=%d metadata frame=%zu slot=%zu action=%zu site=%zu\n",maximum,sizeof(concept_finalized_frame),sizeof(concept_frame_slot),sizeof(concept_frame_action),sizeof(concept_call_site_actions));
 return 0;
}
`
