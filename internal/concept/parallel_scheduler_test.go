package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeSchedulerWorkers(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	sourceBytes, err := os.ReadFile(filepath.Join(root, "DragonGod", "parallel.concept_test"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(sourceBytes), "module DragonGod.Tests.Parallel;", "module Workers.NativeWorker;", 1)
	roots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7e/NativeWorker.concept", source, roots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	schedulerName := ""
	sharedSchedulerName := ""
	for _, decl := range module.Structs {
		if decl.Application == nil || decl.Application.Declaration.Name != "ParallelScheduler" {
			continue
		}
		args := decl.Application.Arguments
		if len(args) > 1 && args[1].Type.Name == "ParallelMachine" {
			schedulerName = evt1StructCName(decl)
		}
		if len(args) > 1 && args[1].Type.Name == "SharedAgentMachine" {
			sharedSchedulerName = evt1StructCName(decl)
		}
	}
	if schedulerName == "" || sharedSchedulerName == "" {
		t.Fatal("missing structured scheduler identities")
	}
	body := strings.ToLower(string(moduleOutput(t, outputs, ".generated.c")))
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "current_scheduler", "schedulerplan"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("unexpected scheduler runtime %q", forbidden)
		}
	}
	harness := nativeThreadShim + `#include "nativeworker.generated.h"
#include <stdio.h>

typedef concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int id; int decisions; } worker_args;
CPT_THREAD_FN(run_worker, raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_workers__native_worker_run_native_worker(args->scheduler, args->id);
  return 0;
}
int main(void) {
  for (int workers = 1; workers <= 4; workers *= 2) {
    for (int quantum = 1; quantum <= 4; quantum *= 2) {
    double begin = cpt_seconds_now();
    for (int trial = 0; trial < 20; ++trial) {
      scheduler_t scheduler = concept_workers__native_worker_make_native_workload(quantum);
      cpt_thread threads[4];
      worker_args args[4];
      for (int i = 0; i < workers; ++i) {
        args[i] = (worker_args){&scheduler, i + 1, 0};
        if (cpt_thread_start(&threads[i], run_worker, &args[i]) != 0) return 2;
      }
      { int join_failed = 0; for (int i = 0; i < workers; ++i) join_failed |= cpt_thread_join(threads[i], 30000); if (join_failed) return 3; }
      int active_workers = 0;
      int decisions = 0;
      for (int i = 0; i < workers; ++i) {
        decisions += args[i].decisions;
        if (args[i].decisions > 0) ++active_workers;
      }
      if (decisions != 4000 / quantum) return 5;
      if (workers > 1 && active_workers < 2) return 6;
      if (concept_workers__native_worker_native_final_steps(&scheduler) != 4000) return 4;
    }
    double seconds = cpt_seconds_now() - begin;
    printf("workers=%d quantum=%d steps=%d seconds=%.6f throughput=%.0f steps/s\n",
           workers, quantum, 20 * 4000, seconds, (20.0 * 4000.0) / seconds);
    }
  }
  for (int trial = 0; trial < 20; ++trial) {
    scheduler_t scheduler = concept_workers__native_worker_make_native_failure_workload();
    cpt_thread threads[4];
    worker_args args[4];
    for (int i = 0; i < 4; ++i) {
      args[i] = (worker_args){&scheduler, i + 1, 0};
      if (cpt_thread_start(&threads[i], run_worker, &args[i]) != 0) return 7;
    }
    { int join_failed = 0; for (int i = 0; i < 4; ++i) join_failed |= cpt_thread_join(threads[i], 30000); if (join_failed) return 8; }
    if (concept_workers__native_worker_native_failure_final_state(&scheduler) != 3000) return 9;
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_native_workers.c", strings.ReplaceAll(harness, "concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_", schedulerName))
	sharedHarness := nativeThreadShim + `#include "nativeworker.generated.h"
typedef concept_parallel_scheduler_parallel_configuration__shared_agent_machine__8__8__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int id; int decisions; } worker_args;
CPT_THREAD_FN(run_worker, raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_workers__native_worker_run_shared_agent_worker(args->scheduler, args->id);
  return 0;
}
int main(void) {
  for (int workers = 1; workers <= 4; workers *= 2) {
    for (int trial = 0; trial < 20; ++trial) {
      concept_shared_agent_fixture fixture = concept_workers__native_worker_make_shared_agent_fixture();
      scheduler_t scheduler = concept_workers__native_worker_make_shared_agent_workload(&fixture);
      cpt_thread threads[4];
      worker_args args[4];
      for (int i = 0; i < workers; ++i) {
        args[i] = (worker_args){&scheduler, i + 1, 0};
        if (cpt_thread_start(&threads[i], run_worker, &args[i]) != 0) return 2;
      }
      { int join_failed = 0; for (int i = 0; i < workers; ++i) join_failed |= cpt_thread_join(threads[i], 30000); if (join_failed) return 3; }
      if (concept_workers__native_worker_shared_agent_final_state(&scheduler, &fixture) != 0) return 4;
    }
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_shared_agents.c", strings.ReplaceAll(sharedHarness, "concept_parallel_scheduler_parallel_configuration__shared_agent_machine__8__8__64_", sharedSchedulerName))
	wakeHarness := nativeThreadShim + `#include "nativeworker.generated.h"
typedef concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int decisions; } worker_args;
CPT_THREAD_FN(run_worker, raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_workers__native_worker_run_native_worker(args->scheduler, 1);
  return 0;
}
int main(void) {
  for (int mode = 0; mode < 3; ++mode) {
    for (int trial = 0; trial < 20; ++trial) {
      scheduler_t scheduler = concept_workers__native_worker_make_native_wake_workload(mode == 1);
      worker_args args = {&scheduler, 0};
      cpt_thread thread;
      if (cpt_thread_start(&thread, run_worker, &args) != 0) return 2;
      int observed = 0;
      for (int spin = 0; spin < 1000000; ++spin) {
        if (concept_workers__native_worker_native_wake_entered(&scheduler) == 1) { observed = 1; break; }
        cpt_thread_yield();
      }
      if (!observed) return 3;
      if (mode == 1) concept_workers__native_worker_native_advance_time(&scheduler);
      else if (mode == 2) concept_workers__native_worker_native_notify_event(&scheduler);
      else concept_workers__native_worker_native_wake_now(&scheduler);
      concept_workers__native_worker_native_wake_proceed(&scheduler);
      if (cpt_thread_join(thread, 30000) != 0) return 4;
      if (args.decisions != 2) return 5;
      if (concept_workers__native_worker_native_wake_final_state(&scheduler) != 0) return 6;
    }
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_wake_races.c", strings.ReplaceAll(wakeHarness, "concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_", schedulerName))
}
