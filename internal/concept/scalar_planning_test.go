package concept

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestEVT2f3ScalarAllocationABIAndFrames(t *testing.T) {
	var declarations strings.Builder
	for i, path := range []string{"../../language/evt2f/valid/scalars.concept", "../../language/evt2f/valid/planning.concept"} {
		source, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		m, e := Parse(path, string(source))
		if e != nil {
			t.Fatal(e)
		}
		mir, e := GenerateMachineIR(m)
		if e != nil {
			t.Fatal(e)
		}
		artifact, e := EncodeMachineBridge(mir)
		if e != nil {
			t.Fatal(e)
		}
		fmt.Fprintf(&declarations, "static const unsigned char artifact%d[]={", i)
		for _, b := range artifact {
			fmt.Fprintf(&declarations, "%d,", b)
		}
		fmt.Fprintf(&declarations, "};\n#define COUNT%d %d\n", i, len(mir.Scalar.Functions))
	}
	harness := strings.Replace(scalarPlanningHarness, "/* ARTIFACTS */", declarations.String(), 1)
	var normal string
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			output := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "scalar_plans.c", harness, "-pedantic-errors", "-O2")
			var observations []string
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "scalar-plan ") || strings.HasPrefix(line, "scalar-call ") {
					observations = append(observations, strings.TrimSpace(line))
				}
			}
			functions := 0
			for _, line := range observations {
				if strings.HasPrefix(line, "scalar-plan ") {
					functions++
				}
			}
			if functions != 61 {
				t.Fatalf("planning fixture count=%d", functions)
			}
			canonical := strings.Join(observations, "\n")
			if verify && canonical != normal {
				t.Fatal("Normal/Verify physical planning differs")
			}
			if !verify {
				normal = canonical
			}
		})
	}
}

const scalarPlanningHarness = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"scalar planning failed line %d: %s\n",__LINE__,#c);return 1;}}while(0)
/* ARTIFACTS */
static int indexof(concept_scalar_location location){return location.registerClass.tag==1?16+location.xmm.tag:location.gpr.tag;}
static uint64_t bits(int id,concept_scalar_representation type){
 if(type.tag==10)return UINT64_C(0x7fc01000)+(unsigned)id;
 if(type.tag==11)return UINT64_C(0x7ff8000000001000)+(unsigned)id;
 if(type.tag==1)return (unsigned)id&1u;
 return UINT64_C(0x1000)+(unsigned)id;
}
// Independent byte-stack oracle. The expected volatile sets and physical ABI
// examples are explicit architectural tables, not planner-derived predictions.
static int byte_replay(const concept_call_site_actions *site,const concept_allocation *a,const concept_finalized_frame *frame,const concept_preservation *records,int count){
 uint64_t registers[32]={0};unsigned char stack[65536]={0};
 for(int i=0;i<site->abi.argumentCount;i++){int id=site->abi.arguments.data[i].source.value;concept_scalar_location loc=a->scalarLocations.data[id];uint64_t token=bits(id,site->abi.scalar.arguments.data[i]);
  if(loc.kind.tag==0)registers[indexof(loc)]=token;else memcpy(stack+frame->slots.data[frame->spillSlots.data[id]].offset,&token,(size_t)site->abi.arguments.data[i].source.width);}
 for(int i=0;i<count;i++)registers[indexof(records[i].location)]=bits(records[i].value,records[i].representation);
 for(int i=0;i<site->count;i++){
  if(i==site->callIndex){
   for(int arg=0;arg<site->abi.argumentCount;arg++){uint64_t actual=0;int width=site->abi.arguments.data[arg].source.width;concept_scalar_location destination=site->abi.scalar.destinations.data[arg];
    if(arg<4)actual=registers[indexof(destination)];else memcpy(&actual,stack+32+(arg-4)*8,(size_t)width);
    uint64_t mask=width==8?UINT64_MAX:((UINT64_C(1)<<(width*8))-1);
    if((actual&mask)!=(bits(site->abi.arguments.data[arg].source.value,site->abi.scalar.arguments.data[arg])&mask))return 0;
   }
   const int volatile_gpr[]={0,2,3,8,9,10,11};for(int v=0;v<7;v++)registers[volatile_gpr[v]]=UINT64_C(0xdeadbeef);
   for(int x=0;x<6;x++)registers[16+x]=UINT64_C(0xbadc0ffee);
   if(site->abi.hasResult)registers[indexof(site->abi.scalar.resultSource)]=bits(site->abi.result.value,site->abi.scalar.result);
  }else{
   concept_scalar_move transfer=site->scalarActions.data[i];int width=site->actions.data[i].width;uint64_t token=0;
   if(transfer.source.kind.tag==0)token=registers[indexof(transfer.source)];else memcpy(&token,stack+site->sourceOffsets.data[i],(size_t)width);
   if(transfer.destination.kind.tag==0)registers[indexof(transfer.destination)]=token;else memcpy(stack+site->destinationOffsets.data[i],&token,(size_t)width);
  }
 }
 for(int i=0;i<count;i++){int width=records[i].width;uint64_t mask=width==8?UINT64_MAX:((UINT64_C(1)<<(width*8))-1);if((registers[indexof(records[i].location)]&mask)!=(bits(records[i].value,records[i].representation)&mask))return 0;}
 if(site->abi.hasResult){int id=site->abi.result.value;concept_scalar_location loc=a->scalarLocations.data[id];uint64_t token=0;
  if(loc.kind.tag==0)token=registers[indexof(loc)];else memcpy(&token,stack+frame->slots.data[frame->spillSlots.data[id]].offset,(size_t)site->abi.result.width);
  if(token!=bits(id,site->abi.scalar.result))return 0;}
 return 1;
}
static void describe(char *out,const concept_backend_function *f,const concept_allocation *a,const concept_finalized_frame *frame){
 out+=sprintf(out,"frame=%d:%d:%d",frame->byteCount,frame->unalignedSize,frame->padding);
 for(int i=0;i<f->virtualRegCount;i++){concept_scalar_location l=a->scalarLocations.data[i];out+=sprintf(out," v%d:%d=%d/%d/%d:%d..%d",i,f->scalarTypes.data[i].tag,l.kind.tag,l.registerClass.tag,indexof(l),a->start.data[i],a->end.data[i]);}
 for(int i=0;i<frame->slotCount;i++){concept_frame_slot s=frame->slots.data[i];out+=sprintf(out," slot%d=%d/%d/%d/%d/%d:%d",i,s.category.tag,s.identity,s.offset,s.size,s.alignment,frame->representations.data[i].tag);}
}
static int named(concept_readonly_span_byte input,const concept_backend_function *f,const char *name){return f->name.length==(int)strlen(name)&&!memcmp(input.data+f->name.offset,name,strlen(name));}
static int plans(concept_readonly_span_byte input,int functions){
 concept_backend_function f=F(empty_function)();concept_allocation a=F(empty_allocation)();concept_finalized_frame frame=F(empty_finalized_frame)();
 concept_liveness live=F(empty_liveness)();concept_preservation records[512];concept_span_preservation span={records,512};
 double allocation_ms=0,frame_ms=0,abi_ms=0;
 for(int ordinal=0;ordinal<functions;ordinal++){
  REQUIRE(!F(decode_scalar_function)(input,ordinal,&f).tag);char baseline[65536],next[65536];
  for(int run=0;run<100;run++){
   a=F(empty_allocation)();clock_t begin=clock();REQUIRE(!F(allocate_registers)(&f,&a).tag);allocation_ms+=1000.0*(clock()-begin)/CLOCKS_PER_SEC;
   REQUIRE(!F(validate_scalar_allocation)(&f,&a).tag);REQUIRE(!F(plan_function_calls)(&f,&a).tag);live=F(empty_liveness)();REQUIRE(!F(compute_liveness)(&f,&live).tag);
   begin=clock();frame=F(empty_finalized_frame)();REQUIRE(!F(qualify_function_frame)(&f,&a,&frame).tag);frame_ms+=1000.0*(clock()-begin)/CLOCKS_PER_SEC;
   describe(next,&f,&a,&frame);if(!run)strcpy(baseline,next);else REQUIRE(!strcmp(baseline,next));
   int spills=0;for(int v=0;v<f.virtualRegCount;v++){concept_scalar_location loc=a.scalarLocations.data[v];int floating=f.scalarTypes.data[v].tag==10||f.scalarTypes.data[v].tag==11;
    REQUIRE(loc.registerClass.tag==floating);if(loc.kind.tag==0&&floating)REQUIRE(loc.xmm.tag<6);if(a.spilled.data[v])spills++;
    if(frame.spillSlots.data[v]>=0){concept_frame_slot slot=frame.slots.data[frame.spillSlots.data[v]];REQUIRE(slot.size==f.virtualRegs.data[v].width&&slot.alignment==slot.size&&frame.representations.data[frame.spillSlots.data[v]].tag==f.scalarTypes.data[v].tag);}}
   if(named(input,&f,"Pressure32")||named(input,&f,"Pressure64"))REQUIRE(spills>=2);
   if(named(input,&f,"MixedPressure"))REQUIRE(spills>=1);
   for(int p=0;p<f.instructionCount;p++)if(f.instructions.data[p].op.tag==16){
    concept_call_plan plan=F(empty_call_plan)();int count=0;begin=clock();REQUIRE(!F(plan_allocated_call)(&f,&a,&live,p,&plan,span,&count).tag);abi_ms+=1000.0*(clock()-begin)/CLOCKS_PER_SEC;
    REQUIRE(plan.scalar.xmmClobberMask==63&&plan.flagsClobbered&&plan.shadowSpaceBytes==32);
    if(named(input,&f,"CallA")){const int classes[]={0,1,0,1},ids[]={2,1,8,3};for(int i=0;i<4;i++){REQUIRE(plan.scalar.destinations.data[i].registerClass.tag==classes[i]);REQUIRE((classes[i]?plan.scalar.destinations.data[i].xmm.tag:plan.scalar.destinations.data[i].gpr.tag)==ids[i]);}}
    if(named(input,&f,"CallB")){const int classes[]={1,0,1,0},ids[]={0,3,2,9};for(int i=0;i<4;i++){REQUIRE(plan.scalar.destinations.data[i].registerClass.tag==classes[i]);REQUIRE((classes[i]?plan.scalar.destinations.data[i].xmm.tag:plan.scalar.destinations.data[i].gpr.tag)==ids[i]);}}
    if(named(input,&f,"Call8")){const int types[]={6,11,1,10,11,10,6,1};for(int i=0;i<8;i++)REQUIRE(plan.scalar.arguments.data[i].tag==types[i]);REQUIRE(plan.stackArgumentCount==4&&plan.stackArgumentBytes==32);for(int i=4;i<8;i++)REQUIRE(plan.scalar.destinations.data[i].kind.tag==1&&plan.scalar.destinations.data[i].slot==i-4);}
    if(named(input,&f,"CallFive")){for(int i=0;i<4;i++)REQUIRE(plan.scalar.destinations.data[i].xmm.tag==i);REQUIRE(plan.scalar.destinations.data[4].kind.tag==1&&plan.scalar.destinations.data[4].slot==0);}
    if(named(input,&f,"Cycle2")||named(input,&f,"Cycle3"))REQUIRE(plan.temporaryCount==1);
    if(named(input,&f,"CallA")||named(input,&f,"CallB")||named(input,&f,"CallZero32")||named(input,&f,"CallZero64"))REQUIRE(count==0);
    if(named(input,&f,"Live32")||named(input,&f,"Live64")){int floating=0;for(int i=0;i<count;i++)if(records[i].representation.tag==10||records[i].representation.tag==11)floating++;REQUIRE(floating==1);}
    if(named(input,&f,"MixedLive")){int integer=0,f32=0,f64=0;for(int i=0;i<count;i++){integer+=records[i].representation.tag==6;f32+=records[i].representation.tag==10;f64+=records[i].representation.tag==11;}REQUIRE(integer==1&&f32==1&&f64==1);}
    if(plan.hasResult&&(plan.scalar.result.tag==10||plan.scalar.result.tag==11))REQUIRE(plan.scalar.resultSource.registerClass.tag==1&&plan.scalar.resultSource.xmm.tag==0);
    REQUIRE(frame.byteCount%16==8);concept_call_site_actions site=F(empty_call_site_actions)();REQUIRE(!F(realize_allocated_call)(&f,&a,&live,p,&frame,&site).tag);REQUIRE(byte_replay(&site,&a,&frame,records,count));
    char call_next[8192];char *cursor=call_next;cursor+=sprintf(cursor,"p%d:%d:%d:%d",p,plan.moveCount,plan.temporaryCount,count);
    for(int i=0;i<site.count;i++){concept_scalar_move tr=site.scalarActions.data[i];cursor+=sprintf(cursor," a%d:%d/%d/%d/%d:%d/%d/%d/%d:%d:%d/%d",i,site.actions.data[i].kind.tag,tr.source.kind.tag,tr.source.registerClass.tag,indexof(tr.source),tr.destination.kind.tag,tr.destination.registerClass.tag,indexof(tr.destination),tr.representation.tag,site.actions.data[i].width,site.sourceOffsets.data[i],site.destinationOffsets.data[i]);}
    static char call_base[512][8192];if(!run){strcpy(call_base[p],call_next);printf("scalar-call %.*s %s\n",f.name.length,(const char*)input.data+f.name.offset,call_next);}else REQUIRE(!strcmp(call_base[p],call_next));
    if(!run&&count){concept_call_site_actions bad=site;int changed=-1;for(int i=site.callIndex+1;i<site.count;i++)if(site.actions.data[i].kind.tag==4){changed=i;break;}if(changed>=0){bad.actions.data[changed].width=16;REQUIRE(F(replay_call_actions)(&bad,span,count,&a,&frame).tag);bad=site;bad.sourceOffsets.data[changed]++;REQUIRE(F(replay_call_actions)(&bad,span,count,&a,&frame).tag);}}
    if(!run&&named(input,&f,"Live32")){concept_call_site_actions bad=site;int first=site.callIndex+1,last=site.count-1;
      concept_frame_action action=bad.actions.data[first];bad.actions.data[first]=bad.actions.data[last];bad.actions.data[last]=action;
      concept_scalar_move transfer=bad.scalarActions.data[first];bad.scalarActions.data[first]=bad.scalarActions.data[last];bad.scalarActions.data[last]=transfer;
      int from=bad.sourceOffsets.data[first],to=bad.destinationOffsets.data[first];bad.sourceOffsets.data[first]=bad.sourceOffsets.data[last];bad.destinationOffsets.data[first]=bad.destinationOffsets.data[last];bad.sourceOffsets.data[last]=from;bad.destinationOffsets.data[last]=to;
      REQUIRE(F(replay_call_actions)(&bad,span,count,&a,&frame).tag);REQUIRE(!byte_replay(&bad,&a,&frame,records,count));}
    if(!run&&p+1<f.instructionCount){concept_backend_function bad=f;bad.instructions.data[p+1].flagsUse=42;concept_call_plan reject=F(empty_call_plan)();int unused=0;
      concept_result_void_abierror error=F(plan_allocated_call)(&bad,&a,&live,p,&reject,span,&unused);REQUIRE(error.tag&&error.payload.error.error.tag==6);}
   }
  }
  printf("scalar-plan %.*s %s\n",f.name.length,(const char*)input.data+f.name.offset,baseline);
 }
 printf("planning 100 runs/function allocation=%.3f ms ABI=%.3f ms frame=%.3f ms\n",allocation_ms,abi_ms,frame_ms);return 0;
}
static concept_call_request request(int count,concept_scalar_representation type){concept_call_request r=F(empty_call_request)();r.scalar.enabled=1;r.argumentCount=count;for(int i=0;i<count;i++){r.scalar.arguments.data[i]=type;r.arguments.data[i]=(concept_abivalue){i,{0},{3},type.tag==10?4:8};}return r;}
static int negatives(void){
 REQUIRE(F(scalar_backend_error)((concept_backend_error){2}).tag==0);REQUIRE(F(scalar_backend_error)((concept_backend_error){4}).tag==13);
 REQUIRE(!strcmp(F(describe_native_error)(F(scalar_native_abierror)((concept_abierror){8})),"AMD64_MOVE_INCOMPATIBLE_REPRESENTATION"));
 REQUIRE(!strcmp(F(describe_native_error)(F(scalar_native_abierror)((concept_abierror){7})),"AMD64_RESERVED_XMM"));
 concept_backend_function legacy=F(empty_function)();concept_allocation legacy_allocation=F(empty_allocation)();legacy_allocation.assigned.data[511]=1;
 REQUIRE(F(allocated_gpr)(&legacy,&legacy_allocation,511));
 concept_call_request r=request(2,(concept_scalar_representation){10});concept_call_plan p=F(empty_call_plan)();REQUIRE(!F(plan_win64call)(&r,&p).tag);
 p.scalar.sources.data[0]=F(xmmlocation)((concept_xmmregister){1});p.scalar.sources.data[1]=F(xmmlocation)((concept_xmmregister){0});REQUIRE(!F(plan_scalar_argument_moves)(&p).tag&&p.temporaryCount==1&&p.scalar.temporaries.data[0].tag==10);
 concept_call_plan bad=p;bad.scalar.moves.data[0].representation.tag=11;REQUIRE(F(validate_scalar_argument_moves)(&bad).tag);bad=p;bad.moveCount=0;REQUIRE(F(validate_scalar_argument_moves)(&bad).tag);
 r.scalar.arguments.data[1].tag=11;r.arguments.data[1].width=8;REQUIRE(!F(plan_win64call)(&r,&p).tag);p.scalar.sources.data[0]=F(xmmlocation)((concept_xmmregister){1});p.scalar.sources.data[1]=F(xmmlocation)((concept_xmmregister){0});REQUIRE(!F(plan_scalar_argument_moves)(&p).tag&&p.temporaryCount==1&&p.scalar.temporaries.data[0].tag==11);REQUIRE(!F(validate_scalar_argument_moves)(&p).tag);
 for(int x=6;x<16;x++){concept_result_void_abierror error=F(validate_scalar_location)(F(xmmlocation)((concept_xmmregister){x}),(concept_scalar_representation){10});REQUIRE(error.tag&&error.payload.error.error.tag==7);}
 r=request(1,(concept_scalar_representation){10});r.form.tag=2;REQUIRE(F(plan_win64call)(&r,&p).tag);r.form.tag=0;r.arguments.data[0].category.tag=4;REQUIRE(F(plan_win64call)(&r,&p).tag);
 r.arguments.data[0].category.tag=3;r.scalar.arguments.data[0].tag=0;REQUIRE(F(plan_win64call)(&r,&p).tag);
 concept_allocation a=F(empty_allocation)();REQUIRE(F(add_scalar_spill)(&a,0,0,(concept_scalar_representation){10}).tag);REQUIRE(!F(add_scalar_spill)(&a,0,1,(concept_scalar_representation){10}).tag);REQUIRE(F(add_scalar_spill)(&a,1,1,(concept_scalar_representation){11}).tag);

 r=request(0,(concept_scalar_representation){10});r.hasResult=1;r.scalar.result.tag=1;r.result=(concept_abivalue){99,{1},{1},1};
 REQUIRE(!F(plan_win64call)(&r,&p).tag&&p.scalar.resultSource.registerClass.tag==0&&p.scalar.resultSource.gpr.tag==0);
 r=request(4,(concept_scalar_representation){11});for(int i=0;i<4;i+=2){r.scalar.arguments.data[i].tag=6;r.arguments.data[i].category.tag=0;r.arguments.data[i].width=4;}
 REQUIRE(!F(plan_win64call)(&r,&p).tag);
 p.scalar.sources.data[0]=F(gprlocation)((concept_register){8});p.scalar.sources.data[2]=F(gprlocation)((concept_register){2});
 p.scalar.sources.data[1]=F(xmmlocation)((concept_xmmregister){3});p.scalar.sources.data[3]=F(xmmlocation)((concept_xmmregister){1});
 REQUIRE(!F(plan_scalar_argument_moves)(&p).tag&&p.temporaryCount==2);REQUIRE(!F(validate_scalar_argument_moves)(&p).tag);
 p.scalar.sources.data[0]=F(xmmlocation)((concept_xmmregister){0});REQUIRE(F(plan_scalar_argument_moves)(&p).tag);
 for(int a0=0;a0<4;a0++)for(int a1=0;a1<4;a1++)for(int a2=0;a2<4;a2++)for(int a3=0;a3<4;a3++){
   const int ids[]={a0,a1,a2,a3};r=request(4,(concept_scalar_representation){10});for(int i=0;i<4;i++)r.arguments.data[i].value=ids[i];REQUIRE(!F(plan_win64call)(&r,&p).tag);
   for(int i=0;i<4;i++)p.scalar.sources.data[i]=F(xmmlocation)((concept_xmmregister){ids[i]});REQUIRE(!F(plan_scalar_argument_moves)(&p).tag);REQUIRE(!F(validate_scalar_argument_moves)(&p).tag);
 }
 printf("metadata CallPlan=%zu legacy=660 delta=%zu Allocation=%zu Frame=%zu CallSite=%zu Location=%zu Preservation=%zu\n",sizeof p,sizeof p-660,sizeof a,sizeof(concept_finalized_frame),sizeof(concept_call_site_actions),sizeof(concept_scalar_location),sizeof(concept_preservation));return 0;
}
int main(void){if(negatives())return 1;concept_readonly_span_byte a={artifact0,sizeof artifact0},b={artifact1,sizeof artifact1};if(plans(a,COUNT0)||plans(b,COUNT1))return 1;return 0;}
`

func TestEVT2f3UnsupportedScalarPlanningInput(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"float16", "EVT2_UNSUPPORTED_FLOAT_REPRESENTATION binary16"},
		// These source fixtures are records, including the SIMD-shaped name.
		// EVT2g retains their layouts but still rejects native scalar admission.
		{"packed_vector", "CMIRAMD5"},
		{"aggregate", "CMIRAMD5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := "../../language/evt2f/invalid/" + tc.name + ".concept"
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(path, string(source))
			if err != nil {
				t.Fatal(err)
			}
			machine, err := GenerateMachineIR(module)
			if tc.want == "CMIRAMD5" {
				if err != nil || machine.Object == nil {
					t.Fatalf("expected aggregate contract: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected early planning admission %q, got %v", tc.want, err)
			}
		})
	}
}
