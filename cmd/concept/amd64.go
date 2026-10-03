package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	for index, function := range machine.Functions {
		cmd = exec.Command(backendExe, bridgePath, strconv.Itoa(index))
		code, err := cmd.Output()
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("EVT2D_BACKEND %s: %w: %s", function.Name, err, e.Stderr)
			}
			return fmt.Errorf("EVT2D_BACKEND %s: %w", function.Name, err)
		}
		fmt.Printf("fn %s [%s]\n  0000: %s\n", function.Name, function.Identity, hex.EncodeToString(code))
	}
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
  fprintf(stderr,"    %s %s -> %s slot=%d [rsp%+d] width=%d value=%d\n",
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
    fprintf(stderr,"%s\n",concept_standard__backend__amd64_describe_frame_error(qualified.payload.error.error));
    return 1;
  }
  fprintf(stderr,"frame size=%d unaligned=%d padding=%d base=post-prologue-RSP returns=%d\n",frame.byteCount,frame.unalignedSize,frame.padding,frame.returnCount);
  for(int r=0;r<7;++r) fprintf(stderr,"  region %s [rsp+%d .. +%d)\n",concept_standard__backend__amd64_frame_region_name(frame.regions.data[r].category),frame.regions.data[r].offset,frame.regions.data[r].offset+frame.regions.data[r].size);
  for(int i=0;i<frame.slotCount;++i) { concept_frame_slot s=frame.slots.data[i]; fprintf(stderr,"  %s(%d) -> [rsp+%d] size=%d align=%d\n",concept_standard__backend__amd64_frame_region_name(s.category),s.identity,s.offset,s.size,s.alignment); }
  fprintf(stderr,"  prologue\n");
  for(int i=0;i<frame.prologueCount;++i) print_frame_action(frame.prologue.data[i]);
  fprintf(stderr,"  epilogue (each return)\n");
  for(int i=0;i<frame.epilogueCount;++i) print_frame_action(frame.epilogue.data[i]);
  for (int position=0; position<function.instructionCount; ++position) {
    concept_machine_instruction instruction = function.instructions.data[position];
    if (!instruction.call.target.length) continue;
    concept_call_plan plan = concept_standard__backend__amd64_empty_call_plan();
    concept_preservation storage[512]; int count=0;
    concept_span_preservation records = {storage,512};
    if (concept_standard__backend__amd64_plan_allocated_call(&function,&allocation,&live,position,&plan,records,&count).tag) return 1;
    fprintf(stderr,"call @%.*s ; instruction %d source %d:%d lir b%d/i%d\n",
      plan.target.length,(const char*)source.data+plan.target.offset,position,instruction.sourceLine,instruction.sourceColumn,instruction.lirBlock,instruction.lirInstruction);
    for (int i=0;i<plan.argumentCount;++i) {
      concept_argument_placement a=plan.arguments.data[i];
      fprintf(stderr,"  arg%d v%d:%s width=%d -> ",i,a.source.value,concept_standard__backend__amd64_call_value_name(a.source.type),a.source.width);
      if(a.onStack) fprintf(stderr,"stack[%d]\n",a.stackSlot);
      else fprintf(stderr,"%s\n",concept_standard__backend__amd64_abiregister_name(a.physical));
    }
    if(plan.hasResult) fprintf(stderr,"  return RAX -> v%d width=%d\n",plan.result.value,plan.result.width);
    else fprintf(stderr,"  return void\n");
    fprintf(stderr,"  shadow=%d stack-bytes=%d flags-clobbered=%d temps=%d\n",plan.shadowSpaceBytes,plan.stackArgumentBytes,plan.flagsClobbered,plan.temporaryCount);
    concept_abiregister_tables tables=concept_standard__backend__amd64_win64registers();
    fprintf(stderr,"  clobbers=");
    for(int i=0;i<7;++i) fprintf(stderr,"%s%s",i?",":"",concept_standard__backend__amd64_abiregister_name(tables.callerSaved.data[i]));
    fprintf(stderr,"\n");
    concept_call_site_actions site = concept_standard__backend__amd64_empty_call_site_actions();
    concept_result_void_frame_error realized = concept_standard__backend__amd64_realize_allocated_call(&function,&allocation,&live,position,&frame,&site);
    if(realized.tag) { fprintf(stderr,"%s\n",concept_standard__backend__amd64_describe_frame_error(realized.payload.error.error)); return 1; }
    fprintf(stderr,"  pre-call\n");
    for(int i=0;i<site.callIndex;++i) print_frame_action(site.actions.data[i]);
    fprintf(stderr,"  abstract-call @%.*s\n",plan.target.length,(const char*)source.data+plan.target.offset);
    fprintf(stderr,"  post-call\n");
    for(int i=site.callIndex+1;i<site.count;++i) print_frame_action(site.actions.data[i]);
    for(int i=0;i<plan.moveCount;++i) {
      concept_argument_move m=plan.moves.data[i];
      fprintf(stderr,"  move ");
      if(m.sourceKind.tag==0) fprintf(stderr,"%s",concept_standard__backend__amd64_abiregister_name(m.source));
      else fprintf(stderr,"temp[%d]",m.sourceSlot);
      fprintf(stderr," -> ");
      if(m.destinationKind.tag==0) fprintf(stderr,"%s",concept_standard__backend__amd64_abiregister_name(m.destination));
      else fprintf(stderr,"%s[%d]",m.destinationKind.tag==1?"stack":"temp",m.destinationSlot);
      fprintf(stderr," width=%d\n",m.width);
    }
    for(int i=0;i<count;++i) fprintf(stderr,"  live v%d width=%d %s %s range=%d..%d\n",storage[i].value,storage[i].width,
      concept_standard__backend__amd64_abiregister_name(storage[i].physical),concept_standard__backend__amd64_preservation_name(storage[i].classification),storage[i].start,storage[i].end);
    fprintf(stderr,"  used-callee-saved=");
    for(int i=0;i<8;++i) {
      concept_register r=tables.calleeSaved.data[i];
      if(allocation.usedCalleeSaved.data[concept_standard__backend__amd64_register_index(r)]) fprintf(stderr,"%s ",concept_standard__backend__amd64_abiregister_name(r));
    }
    fprintf(stderr,"\n");
  }
  return 0;
}
int main(int argc, char** argv) {
  if (argc != 3) return 2;
  FILE* file = fopen(argv[1], "rb");
  if (!file) return 3;
  if (fseek(file, 0, SEEK_END) != 0) return 4;
  long size = ftell(file);
  if (size <= 0 || size > 10000000 || fseek(file, 0, SEEK_SET) != 0) return 5;
  unsigned char* input = malloc((size_t)size);
  if (!input || fread(input, 1, (size_t)size, file) != (size_t)size) return 6;
  fclose(file);
  unsigned char output[65536];
  concept_readonly_span_byte source = {input, (size_t)size};
  concept_span_byte target = {output, sizeof output};
  concept_result_int_backend_error result = concept_standard__backend__amd64_emit_function(source, atoi(argv[2]), target);
  if (result.tag != 0) {
    if (print_call_plans(source, atoi(argv[2]))) fprintf(stderr,"AMD64_CALL_PLAN_DIAGNOSTIC_FAILED\n");
    fprintf(stderr, "%s (tag=%u)\n", concept_standard__backend__amd64_describe_backend_error(result.payload.error.error), result.payload.error.error.tag);
    free(input);
    return 7;
  }
  free(input);
  if (fwrite(output, 1, (size_t)result.payload.ok.value, stdout) != (size_t)result.payload.ok.value) return 8;
  return 0;
}
`
