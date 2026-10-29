package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

// The temporary C executable is the bootstrap host for the Concept module.
// Go only transports verified MachineIR and displays the bytes it returns.
func printConceptAMD64(machine concept.MachineModule, artifact []byte) error {
	backendPath, err := findAMD64BackendSource()
	if err != nil {
		return err
	}
	source, err := os.ReadFile(backendPath)
	if err != nil {
		return err
	}
	// The backend is an ordinary Concept library with imports. Build those
	// dependencies from the checkout's library root before generating C.
	libraryRoot := filepath.Dir(filepath.Dir(filepath.Dir(backendPath)))
	module, err := concept.ParseWithBuiltSemanticModuleRoots(backendPath, string(source), []string{libraryRoot})
	if err != nil {
		return err
	}
	outputs, err := concept.Generate(module, source)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "concept-amd64-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := concept.Write(dir, outputs); err != nil {
		return err
	}
	var generatedC string
	for name := range outputs {
		if strings.HasSuffix(name, ".generated.c") {
			generatedC = filepath.Join(dir, name)
		}
	}
	if generatedC == "" {
		return fmt.Errorf("EVT2D_BACKEND_C_MISSING")
	}
	bridgePath := filepath.Join(dir, "module.cmir")
	if err := os.WriteFile(bridgePath, artifact, 0600); err != nil {
		return err
	}
	harnessPath := filepath.Join(dir, "backend_host.c")
	scalarCount := 0
	if machine.Scalar != nil {
		scalarCount = len(machine.Scalar.Functions)
	}
	host := strings.ReplaceAll(amd64BootstrapHost, "/* SCALAR_COUNT */", fmt.Sprint(scalarCount))
	if err := os.WriteFile(harnessPath, []byte(host), 0600); err != nil {
		return err
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		return fmt.Errorf("EVT2D_C11_COMPILER_MISSING: %w", err)
	}
	backendExe := filepath.Join(dir, "concept-amd64-backend.exe")
	cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-O2", "-I", dir, generatedC, harnessPath, "-o", backendExe)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("EVT2D_BACKEND_BOOTSTRAP: %w\n%s", err, output)
	}
	_ = machine // Semantic identities and layout are consumed by Concept.
	cmd = exec.Command(backendExe, bridgePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("EVT2E_NATIVE_BACKEND: %w\n%s", err, output)
	}
	fmt.Print(string(output))
	return nil
}
func findAMD64BackendSource() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "libraries", "Standard", "Backend", "AMD64.concept")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("EVT2D_BACKEND_SOURCE_MISSING: run inside a Concept checkout")
		}
		dir = parent
	}
}

const amd64BootstrapHost = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
// Presentation only: every placement, move and preservation decision comes
// from the compiled Cathedral planner.
static void print_frame_action(concept_frame_action a) {
  fprintf(stdout,"    %s %s -> %s slot=%d [rsp%+d] width=%d value=%d\n",
    concept_standard__backend__amd64_frame_action_name(a.kind),
    concept_standard__backend__amd64_abiregister_name(a.source),
    concept_standard__backend__amd64_abiregister_name(a.destination),a.slot,a.offset,a.width,a.value);
}
static int print_call_plans(concept_readonly_span_byte source, int ordinal) {
  concept_backend_function function = concept_standard__backend__amd64_empty_function();
  if (concept_standard__backend__amd64_decode_function(source, ordinal, &function).tag) return 1;
  if (concept_standard__backend__amd64_layout_lowered_blocks(&function).tag) return 1;
  concept_allocation allocation = concept_standard__backend__amd64_empty_allocation();
  if (concept_standard__backend__amd64_allocate_registers(&function, &allocation).tag) return 1;
  if (concept_standard__backend__amd64_plan_function_calls(&function, &allocation).tag) return 1;
  concept_liveness live = concept_standard__backend__amd64_empty_liveness();
  if (concept_standard__backend__amd64_compute_liveness(&function, &live).tag) return 1;
  concept_finalized_frame frame = concept_standard__backend__amd64_empty_finalized_frame();
  concept_result_void_frame_error qualified = concept_standard__backend__amd64_qualify_function_frame(&function,&allocation,&frame);
  if(qualified.tag) {
    fprintf(stdout,"%s\n",concept_standard__backend__amd64_describe_frame_error(qualified.payload.error.error));
    return 1;
  }
  fprintf(stdout,"frame size=%d unaligned=%d padding=%d base=post-prologue-RSP returns=%d\n",frame.byteCount,frame.unalignedSize,frame.padding,frame.returnCount);
  for(int r=0;r<7;++r) fprintf(stdout,"  region %s [rsp+%d .. +%d)\n",concept_standard__backend__amd64_frame_region_name(frame.regions.data[r].category),frame.regions.data[r].offset,frame.regions.data[r].offset+frame.regions.data[r].size);
  for(int i=0;i<frame.slotCount;++i) { concept_frame_slot s=frame.slots.data[i]; fprintf(stdout,"  %s(%d) -> [rsp+%d] size=%d align=%d\n",concept_standard__backend__amd64_frame_region_name(s.category),s.identity,s.offset,s.size,s.alignment); }
  fprintf(stdout,"  prologue\n");
  for(int i=0;i<frame.prologueCount;++i) print_frame_action(frame.prologue.data[i]);
  fprintf(stdout,"  epilogue (each return)\n");
  for(int i=0;i<frame.epilogueCount;++i) print_frame_action(frame.epilogue.data[i]);
  for (int position=0; position<function.instructionCount; ++position) {
    concept_machine_instruction instruction = function.instructions.data[position];
    if (!instruction.call.target.length) continue;
    concept_call_plan plan = concept_standard__backend__amd64_empty_call_plan();
    concept_preservation storage[512]; int count=0;
    concept_span_preservation records = {storage,512};
    if (concept_standard__backend__amd64_plan_allocated_call(&function,&allocation,&live,position,&plan,records,&count).tag) return 1;
    fprintf(stdout,"call @%.*s ; instruction %d source %d:%d lir b%d/i%d\n",
      plan.target.length,(const char*)source.data+plan.target.offset,position,instruction.sourceLine,instruction.sourceColumn,instruction.lirBlock,instruction.lirInstruction);
    for (int i=0;i<plan.argumentCount;++i) {
      concept_argument_placement a=plan.arguments.data[i];
      fprintf(stdout,"  arg%d v%d:%s width=%d -> ",i,a.source.value,concept_standard__backend__amd64_call_value_name(a.source.type),a.source.width);
      if(a.onStack) fprintf(stdout,"stack[%d]\n",a.stackSlot);
      else fprintf(stdout,"%s\n",concept_standard__backend__amd64_abiregister_name(a.physical));
    }
    if(plan.hasResult) fprintf(stdout,"  return RAX -> v%d width=%d\n",plan.result.value,plan.result.width);
    else fprintf(stdout,"  return void\n");
    fprintf(stdout,"  shadow=%d stack-bytes=%d flags-clobbered=%d temps=%d\n",plan.shadowSpaceBytes,plan.stackArgumentBytes,plan.flagsClobbered,plan.temporaryCount);
    concept_abiregister_tables tables=concept_standard__backend__amd64_win64registers();
    fprintf(stdout,"  clobbers=");
    for(int i=0;i<7;++i) fprintf(stdout,"%s%s",i?",":"",concept_standard__backend__amd64_abiregister_name(tables.callerSaved.data[i]));
    fprintf(stdout,"\n");
    concept_call_site_actions site = concept_standard__backend__amd64_empty_call_site_actions();
    concept_result_void_frame_error realized = concept_standard__backend__amd64_realize_allocated_call(&function,&allocation,&live,position,&frame,&site);
    if(realized.tag) { fprintf(stdout,"%s\n",concept_standard__backend__amd64_describe_frame_error(realized.payload.error.error)); return 1; }
    fprintf(stdout,"  pre-call\n");
    for(int i=0;i<site.callIndex;++i) print_frame_action(site.actions.data[i]);
    fprintf(stdout,"  abstract-call @%.*s\n",plan.target.length,(const char*)source.data+plan.target.offset);
    fprintf(stdout,"  post-call\n");
    for(int i=site.callIndex+1;i<site.count;++i) print_frame_action(site.actions.data[i]);
    for(int i=0;i<plan.moveCount;++i) {
      concept_argument_move m=plan.moves.data[i];
      fprintf(stdout,"  move ");
      if(m.sourceKind.tag==0) fprintf(stdout,"%s",concept_standard__backend__amd64_abiregister_name(m.source));
      else fprintf(stdout,"temp[%d]",m.sourceSlot);
      fprintf(stdout," -> ");
      if(m.destinationKind.tag==0) fprintf(stdout,"%s",concept_standard__backend__amd64_abiregister_name(m.destination));
      else fprintf(stdout,"%s[%d]",m.destinationKind.tag==1?"stack":"temp",m.destinationSlot);
      fprintf(stdout," width=%d\n",m.width);
    }
    for(int i=0;i<count;++i) fprintf(stdout,"  live v%d width=%d %s %s range=%d..%d\n",storage[i].value,storage[i].width,
      concept_standard__backend__amd64_abiregister_name(storage[i].physical),concept_standard__backend__amd64_preservation_name(storage[i].classification),storage[i].start,storage[i].end);
    fprintf(stdout,"  used-callee-saved=");
    for(int i=0;i<8;++i) {
      concept_register r=tables.calleeSaved.data[i];
      if(allocation.usedCalleeSaved.data[concept_standard__backend__amd64_register_index(r)]) fprintf(stdout,"%s ",concept_standard__backend__amd64_abiregister_name(r));
    }
    fprintf(stdout,"\n");
  }
  return 0;
}
static void print_scalar_location(concept_scalar_location location) {
  if(location.kind.tag==0) printf("%s",concept_standard__backend__amd64_scalar_physical_name(location));
  else printf("%s[%d]",location.kind.tag==1?"stack":location.kind.tag==2?"temp":location.kind.tag==3?"spill":"incoming",location.slot);
}
static int print_scalar_plans(concept_readonly_span_byte source,int ordinal) {
  concept_backend_function f=concept_standard__backend__amd64_empty_function();
  if(concept_standard__backend__amd64_decode_scalar_function(source,ordinal,&f).tag) return 1;
  printf("fn %.*s scalar planning\n",f.name.length,(const char*)source.data+f.name.offset);
  concept_allocation a=concept_standard__backend__amd64_empty_allocation();
  concept_result_void_backend_error allocated=concept_standard__backend__amd64_allocate_registers(&f,&a);
  if(allocated.tag){puts(concept_standard__backend__amd64_describe_native_error(concept_standard__backend__amd64_scalar_backend_error(allocated.payload.error.error)));return 2;}
  concept_result_void_abierror calls=concept_standard__backend__amd64_plan_function_calls_checked(&f,&a);
  if(calls.tag){puts(concept_standard__backend__amd64_describe_abierror(calls.payload.error.error));return 2;}
  concept_liveness live=concept_standard__backend__amd64_empty_liveness();
  if(concept_standard__backend__amd64_compute_liveness(&f,&live).tag) return 3;
  concept_finalized_frame frame=concept_standard__backend__amd64_empty_finalized_frame();
  concept_result_void_frame_error qualified=concept_standard__backend__amd64_qualify_scalar_native_frame(&f,&a,&frame);
  if(qualified.tag){printf("%s\n",concept_standard__backend__amd64_describe_frame_error(qualified.payload.error.error));return 4;}
  for(int v=0;v<f.virtualRegCount;v++){printf("  v%d:%s -> ",v,concept_standard__backend__amd64_scalar_type_name(f.scalarTypes.data[v]));print_scalar_location(a.scalarLocations.data[v]);printf(" range=%d..%d\n",a.start.data[v],a.end.data[v]);}
  for(int i=0;i<f.argCount;i++){printf("  incoming arg%d -> ",i);print_scalar_location(concept_standard__backend__amd64_win64scalar_argument(i,f.scalarTypes.data[f.args.data[i].virtualReg]));puts("");}
  printf("  frame size=%d unaligned=%d padding=%d base=post-prologue-RSP\n",frame.byteCount,frame.unalignedSize,frame.padding);
  for(int r=0;r<7;r++)printf("  region %s [rsp+%d .. +%d)\n",concept_standard__backend__amd64_frame_region_name(frame.regions.data[r].category),frame.regions.data[r].offset,frame.regions.data[r].offset+frame.regions.data[r].size);
  for(int s=0;s<frame.slotCount;s++){concept_frame_slot slot=frame.slots.data[s];printf("  %s v%d %s -> [rsp+%d] size=%d align=%d\n",concept_standard__backend__amd64_frame_region_name(slot.category),slot.identity,concept_standard__backend__amd64_scalar_type_name(frame.representations.data[s]),slot.offset,slot.size,slot.alignment);}
  for(int p=0;p<f.instructionCount;p++){if(f.instructions.data[p].op.tag!=16)continue;
    concept_call_plan plan=concept_standard__backend__amd64_empty_call_plan();concept_preservation records[512];int count=0;concept_span_preservation span={records,512};
    concept_result_void_abierror planned=concept_standard__backend__amd64_plan_allocated_call(&f,&a,&live,p,&plan,span,&count);
    if(planned.tag){puts(concept_standard__backend__amd64_describe_abierror(planned.payload.error.error));return 5;}
    printf("  call @%.*s instruction=%d shadow=%d stack-bytes=%d flags-clobbered=%d xmm-clobbers=XMM0..XMM5\n",plan.target.length,(const char*)source.data+plan.target.offset,p,plan.shadowSpaceBytes,plan.stackArgumentBytes,plan.flagsClobbered);
    for(int i=0;i<plan.argumentCount;i++){printf("    arg%d v%d:%s -> ",i,plan.arguments.data[i].source.value,concept_standard__backend__amd64_scalar_type_name(plan.scalar.arguments.data[i]));print_scalar_location(plan.scalar.destinations.data[i]);puts("");}
    if(plan.hasResult){printf("    return ");print_scalar_location(plan.scalar.resultSource);printf(" -> v%d:%s\n",plan.result.value,concept_standard__backend__amd64_scalar_type_name(plan.scalar.result));}else puts("    return void");
    concept_call_site_actions site=concept_standard__backend__amd64_empty_call_site_actions();
    if(concept_standard__backend__amd64_realize_allocated_call(&f,&a,&live,p,&frame,&site).tag)return 6;
    for(int i=0;i<site.count;i++){if(i==site.callIndex){puts("    abstract-call (CALL rel32)");continue;}
      printf("    %s %s ",i<site.callIndex?"pre-call":"post-call",concept_standard__backend__amd64_frame_action_name(site.actions.data[i].kind));
      print_scalar_location(site.scalarActions.data[i].source);if(site.sourceOffsets.data[i]>=0)printf(" [rsp+%d]",site.sourceOffsets.data[i]);printf(" -> ");print_scalar_location(site.scalarActions.data[i].destination);if(site.destinationOffsets.data[i]>=0)printf(" [rsp+%d]",site.destinationOffsets.data[i]);printf(" %s\n",concept_standard__backend__amd64_scalar_type_name(site.scalarActions.data[i].representation));}
  }
  return 0;
}
static int print_object_plans(concept_readonly_span_byte source,int ordinal) {
  concept_backend_function f=concept_standard__backend__amd64_empty_function();
  concept_object_placement p=concept_standard__backend__amd64_empty_object_placement();
  concept_allocation a=concept_standard__backend__amd64_empty_allocation();
  concept_finalized_frame frame=concept_standard__backend__amd64_empty_finalized_frame();
  if(concept_standard__backend__amd64_plan_object_native_function(source,ordinal,&f,&p,&a,&frame).tag)return 1;
  printf("  object frame size=%d locals=%d base=post-prologue-RSP\n",frame.byteCount,frame.regions.data[2].size);
  for(int i=0;i<f.slotCount;i++){concept_machine_slot slot=f.slots.data[i];printf("    storage s%d %.*s -> [rsp+%d] size=%d align=%d\n",i,slot.source.length,(const char*)source.data+slot.source.offset,frame.slotOffsets.data[i],slot.size,slot.alignment);}
  for(int v=0;v<512;v++){if(p.values.data[v]>=0)printf("    aggregate v%d -> storage s%d\n",v,p.values.data[v]);}
  return 0;
}
static void print_aggregate_type(concept_readonly_span_byte source,concept_aggregate_abitype type) {
  printf("%.*s size=%d align=%d layout=%.*s %s",type.identity.length,(const char*)source.data+type.identity.offset,type.size,type.alignment,type.layout.length,(const char*)source.data+type.layout.offset,concept_standard__backend__amd64_aggregate_abiclass_name(type.category));
}
static int print_aggregate_plans(concept_readonly_span_byte source) {
  concept_result_wire_object_module_aggregate_abierror decoded=concept_standard__backend__amd64_admitted_aggregate_document(source);
  if(decoded.tag)return 0;
  concept_bridge_reader reader={source,0};
  if(!concept_standard__backend__amd64_object_module_has_aggregate_abi(&reader,decoded.payload.ok.value))return 0;
  static concept_backend_function f;
  static concept_object_placement p;
  static concept_allocation a;
  static concept_finalized_frame frame;
  static concept_aggregate_function_plan plan;
  for(int i=0;i<decoded.payload.ok.value.Functions.count;i++) {
    concept_result_void_aggregate_abierror r=concept_standard__backend__amd64_plan_aggregate_function(source,i,&f,&p,&a,&frame,&plan);
    if(r.tag){fprintf(stderr,"function %d: %s\n",i,concept_standard__backend__amd64_describe_aggregate_abierror(r.payload.error.error));return 1;}
    printf("fn %.*s Win64 default aggregate ABI\n  return ",f.name.length,(const char*)source.data+f.name.offset);
    print_aggregate_type(source,plan.result);
    if(plan.result.category.tag==6)printf(" caller-provided storage: RCX; same pointer -> RAX; carrier v%d",plan.hiddenPointer);
    if(plan.result.category.tag==5)printf(" payload -> RAX");
    puts("");
    for(int j=0;j<plan.parameterCount;j++) {
      concept_aggregate_argument_plan x=plan.parameters.data[j];
      printf("  parameter%d physical%d v%d ",j,x.physicalIndex,x.value);print_aggregate_type(source,x.type);printf(" -> ");
      print_scalar_location(concept_standard__backend__amd64_win64scalar_argument(x.physicalIndex,x.type.carrier));printf(" private-storage=s%d\n",x.storage);
    }
    printf("  frame size=%d locals=%d shadow=%d spills=%d saves=%d temps=%d base=post-prologue-RSP\n",frame.byteCount,frame.regions.data[2].size,frame.regions.data[0].size,frame.regions.data[4].size,frame.regions.data[3].size,frame.regions.data[5].size);
    for(int j=0;j<plan.callCount;j++) {
      concept_aggregate_call_plan c=plan.calls.data[j];
      printf("  call @%.*s b%d/i%d hidden-result=s%d pointer=v%d; return ",c.physical.target.length,(const char*)source.data+c.physical.target.offset,c.block,c.instruction,c.resultStorage,c.hiddenPointer);print_aggregate_type(source,c.result);puts("");
      for(int k=0;k<c.argumentCount;k++) {
        concept_aggregate_argument_plan x=c.arguments.data[k];
        printf("    arg%d physical%d v%d ",k,x.physicalIndex,x.value);print_aggregate_type(source,x.type);printf(" -> ");print_scalar_location(c.physical.scalar.destinations.data[x.physicalIndex]);
        if(x.storage>=0)printf(" storage=s%d [rsp+%d] lifetime=%d..%d",x.storage,frame.slotOffsets.data[x.storage],x.start,x.end);puts("");
      }
      printf("    shadow=%d outgoing=%d moves=%d cycle-temps=%d\n",c.physical.shadowSpaceBytes,c.physical.stackArgumentBytes,c.physical.moveCount,c.physical.temporaryCount);
      for(int k=0;k<c.physical.moveCount;k++){concept_scalar_move m=c.physical.scalar.moves.data[k];printf("    move ");print_scalar_location(m.source);printf(" -> ");print_scalar_location(m.destination);printf(" %s\n",concept_standard__backend__amd64_scalar_type_name(m.representation));}
    }
    for(int j=0;j<plan.transferCount;j++){concept_aggregate_transfer t=plan.transfers.data[j];printf("  transfer %s t%d v%d source=s%d storage=s%d carrier=v%d point=%d end=%d\n",concept_standard__backend__amd64_aggregate_transfer_name(t.kind),t.type,t.value,t.sourceSlot,t.storage,t.carrier,t.point,t.end);}
  }
  return 0;
}
int main(int argc, char** argv) {
  if(argc!=2)return 2;
  FILE *file=fopen(argv[1],"rb");if(!file)return 3;
  if(fseek(file,0,SEEK_END))return 4;long size=ftell(file);
  if(size<=0||size>10000000||fseek(file,0,SEEK_SET))return 5;
  unsigned char *input=malloc((size_t)size);if(!input||fread(input,1,(size_t)size,file)!=(size_t)size)return 6;fclose(file);
  unsigned char bytes[65536];concept_readonly_span_byte source={input,(size_t)size};concept_span_byte output={bytes,sizeof bytes};
  concept_native_image image=concept_standard__backend__amd64_empty_native_image();
  if(size>=8 && !memcmp(input,"CMIRAMD4",8)){for(int i=0;i</* SCALAR_COUNT */;i++){if(print_scalar_plans(source,i)){free(input);return 8;}}}
  if(size>=8 && !memcmp(input,"CMIRAMD6",8)){concept_result_void_borrow_error admitted=concept_standard__backend__amd64_validate_borrow_machine_bridge(source);if(admitted.tag){fprintf(stderr,"%s\n",concept_standard__backend__amd64_describe_borrow_error(admitted.payload.error.error));concept_result_wire_borrow_module_backend_error decoded=concept_standard__backend__amd64_native_borrow_document(source);if(!decoded.tag){concept_bridge_reader reader={source,0};for(int i=0;i<decoded.payload.ok.value.Objects.Functions.count;i++){concept_result_void_borrow_error checked=concept_standard__backend__amd64_validate_borrow_function(&reader,decoded.payload.ok.value,i);if(checked.tag)fprintf(stderr,"function %d: %s\n",i,concept_standard__backend__amd64_describe_borrow_error(checked.payload.error.error));}}free(input);return 7;}}
  if(size>=8 && (!memcmp(input,"CMIRAMD5",8)||!memcmp(input,"CMIRAMD6",8))){if(print_aggregate_plans(source)){free(input);return 8;}}
  concept_result_int_native_error result=concept_standard__backend__amd64_emit_module(source,output,&image);
  if(result.tag){fflush(stdout);fprintf(stderr,"%s\n",concept_standard__backend__amd64_describe_native_error(result.payload.error.error));free(input);return 7;}
  printf("native image bytes=%d functions=%d calls=%d finalized=%d\n",image.byteCount,image.symbolCount,image.fixupCount,image.finalized);
  for(int i=0;i<image.symbolCount;++i){concept_native_symbol symbol=image.symbols.data[i];
    printf("fn %.*s [%.*s] offset=%d size=%d\n",symbol.name.length,(const char*)input+symbol.name.offset,symbol.identity.length,(const char*)input+symbol.identity.offset,symbol.offset,symbol.size);
    printf("  %04x: ",symbol.offset);for(int b=0;b<symbol.size;++b)printf("%02x",bytes[symbol.offset+b]);printf("\n");
    if(size>=8 && (!memcmp(input,"CMIRAMD5",8)||!memcmp(input,"CMIRAMD6",8))){if(print_object_plans(source,i)){free(input);return 8;}}
    concept_backend_function function=concept_standard__backend__amd64_empty_function();
    if(!concept_standard__backend__amd64_decode_function(source,i,&function).tag){
      int calls=0;for(int p=0;p<function.instructionCount;++p)if(function.instructions.data[p].op.tag==16)calls++;
      if(calls && print_call_plans(source,i)){free(input);return 8;}
    }
  }
  for(int i=0;i<image.fixupCount;++i){concept_call_fixup fixup=image.fixups.data[i];printf("fixup source=%d call=%d target=%.*s rel32=%d resolved=%d\n",fixup.sourceOrdinal,fixup.callOffset,fixup.target.length,(const char*)input+fixup.target.offset,fixup.displacement,fixup.resolved);}
  free(input);return 0;
}
`
