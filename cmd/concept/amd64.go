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
	if err := os.WriteFile(harnessPath, []byte(amd64BootstrapHost), 0600); err != nil {
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
int main(int argc, char** argv) {
  if(argc!=2)return 2;
  FILE *file=fopen(argv[1],"rb");if(!file)return 3;
  if(fseek(file,0,SEEK_END))return 4;long size=ftell(file);
  if(size<=0||size>10000000||fseek(file,0,SEEK_SET))return 5;
  unsigned char *input=malloc((size_t)size);if(!input||fread(input,1,(size_t)size,file)!=(size_t)size)return 6;fclose(file);
  unsigned char bytes[65536];concept_readonly_span_byte source={input,(size_t)size};concept_span_byte output={bytes,sizeof bytes};
  concept_native_image image=concept_standard__backend__amd64_empty_native_image();
  concept_result_int_native_error result=concept_standard__backend__amd64_emit_module(source,output,&image);
  if(result.tag){fprintf(stderr,"%s\n",concept_standard__backend__amd64_describe_native_error(result.payload.error.error));free(input);return 7;}
  printf("native image bytes=%d functions=%d calls=%d finalized=%d\n",image.byteCount,image.symbolCount,image.fixupCount,image.finalized);
  for(int i=0;i<image.symbolCount;++i){concept_native_symbol symbol=image.symbols.data[i];
    printf("fn %.*s [%.*s] offset=%d size=%d\n",symbol.name.length,(const char*)input+symbol.name.offset,symbol.identity.length,(const char*)input+symbol.identity.offset,symbol.offset,symbol.size);
    printf("  %04x: ",symbol.offset);for(int b=0;b<symbol.size;++b)printf("%02x",bytes[symbol.offset+b]);printf("\n");
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
