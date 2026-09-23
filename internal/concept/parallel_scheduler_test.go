package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7eNativeWorkers(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	sourceBytes, err := os.ReadFile(filepath.Join(root, "DragonGod", "parallel.concept_test"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(sourceBytes), "module DragonGod.Tests.Parallel;", "module R7e.NativeWorker;", 1)
	roots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7e/NativeWorker.concept", source, roots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ToLower(string(moduleOutput(t, outputs, ".generated.c")))
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "current_scheduler", "schedulerplan"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("unexpected scheduler runtime %q", forbidden)
		}
	}
	harness := `#include "nativeworker.generated.h"
#include <windows.h>
#include <stdio.h>

typedef concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int id; int decisions; } worker_args;
static DWORD WINAPI run_worker(LPVOID raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_r7e__native_worker_run_native_worker(args->scheduler, args->id);
  return 0;
}
int main(void) {
  LARGE_INTEGER frequency;
  QueryPerformanceFrequency(&frequency);
  for (int workers = 1; workers <= 4; workers *= 2) {
    for (int quantum = 1; quantum <= 4; quantum *= 2) {
    LARGE_INTEGER begin, end;
    QueryPerformanceCounter(&begin);
    for (int trial = 0; trial < 20; ++trial) {
      scheduler_t scheduler = concept_r7e__native_worker_make_native_workload(quantum);
      HANDLE threads[4];
      worker_args args[4];
      for (int i = 0; i < workers; ++i) {
        args[i] = (worker_args){&scheduler, i + 1, 0};
        threads[i] = CreateThread(NULL, 0, run_worker, &args[i], 0, NULL);
        if (threads[i] == NULL) return 2;
      }
      if (WaitForMultipleObjects((DWORD)workers, threads, TRUE, 30000) != WAIT_OBJECT_0) return 3;
      int active_workers = 0;
      int decisions = 0;
      for (int i = 0; i < workers; ++i) {
        CloseHandle(threads[i]);
        decisions += args[i].decisions;
        if (args[i].decisions > 0) ++active_workers;
      }
      if (decisions != 4000 / quantum) return 5;
      if (workers > 1 && active_workers < 2) return 6;
      if (concept_r7e__native_worker_native_final_steps(&scheduler) != 4000) return 4;
    }
    QueryPerformanceCounter(&end);
    double seconds = (double)(end.QuadPart - begin.QuadPart) / (double)frequency.QuadPart;
    printf("workers=%d quantum=%d steps=%d seconds=%.6f throughput=%.0f steps/s\n",
           workers, quantum, 20 * 4000, seconds, (20.0 * 4000.0) / seconds);
    }
  }
  for (int trial = 0; trial < 20; ++trial) {
    scheduler_t scheduler = concept_r7e__native_worker_make_native_failure_workload();
    HANDLE threads[4];
    worker_args args[4];
    for (int i = 0; i < 4; ++i) {
      args[i] = (worker_args){&scheduler, i + 1, 0};
      threads[i] = CreateThread(NULL, 0, run_worker, &args[i], 0, NULL);
      if (threads[i] == NULL) return 7;
    }
    if (WaitForMultipleObjects(4, threads, TRUE, 30000) != WAIT_OBJECT_0) return 8;
    for (int i = 0; i < 4; ++i) CloseHandle(threads[i]);
    if (concept_r7e__native_worker_native_failure_final_state(&scheduler) != 3000) return 9;
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_native_workers.c", harness)
	sharedHarness := `#include "nativeworker.generated.h"
#include <windows.h>
typedef concept_parallel_scheduler_parallel_configuration__shared_agent_machine__8__8__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int id; int decisions; } worker_args;
static DWORD WINAPI run_worker(LPVOID raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_r7e__native_worker_run_shared_agent_worker(args->scheduler, args->id);
  return 0;
}
int main(void) {
  for (int workers = 1; workers <= 4; workers *= 2) {
    for (int trial = 0; trial < 20; ++trial) {
      concept_shared_agent_fixture fixture = concept_r7e__native_worker_make_shared_agent_fixture();
      scheduler_t scheduler = concept_r7e__native_worker_make_shared_agent_workload(&fixture);
      HANDLE threads[4];
      worker_args args[4];
      for (int i = 0; i < workers; ++i) {
        args[i] = (worker_args){&scheduler, i + 1, 0};
        threads[i] = CreateThread(NULL, 0, run_worker, &args[i], 0, NULL);
        if (threads[i] == NULL) return 2;
      }
      if (WaitForMultipleObjects((DWORD)workers, threads, TRUE, 30000) != WAIT_OBJECT_0) return 3;
      for (int i = 0; i < workers; ++i) CloseHandle(threads[i]);
      if (concept_r7e__native_worker_shared_agent_final_state(&scheduler, &fixture) != 0) return 4;
    }
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_shared_agents.c", sharedHarness)
	wakeHarness := `#include "nativeworker.generated.h"
#include <windows.h>
typedef concept_parallel_scheduler_parallel_configuration__parallel_machine__4__4__64_ scheduler_t;
typedef struct { scheduler_t* scheduler; int decisions; } worker_args;
static DWORD WINAPI run_worker(LPVOID raw) {
  worker_args* args = (worker_args*)raw;
  args->decisions = concept_r7e__native_worker_run_native_worker(args->scheduler, 1);
  return 0;
}
int main(void) {
  for (int mode = 0; mode < 3; ++mode) {
    for (int trial = 0; trial < 20; ++trial) {
      scheduler_t scheduler = concept_r7e__native_worker_make_native_wake_workload(mode == 1);
      worker_args args = {&scheduler, 0};
      HANDLE thread = CreateThread(NULL, 0, run_worker, &args, 0, NULL);
      if (thread == NULL) return 2;
      int observed = 0;
      for (int spin = 0; spin < 1000000; ++spin) {
        if (concept_r7e__native_worker_native_wake_entered(&scheduler) == 1) { observed = 1; break; }
        Sleep(0);
      }
      if (!observed) return 3;
      if (mode == 1) concept_r7e__native_worker_native_advance_time(&scheduler);
      else if (mode == 2) concept_r7e__native_worker_native_notify_event(&scheduler);
      else concept_r7e__native_worker_native_wake_now(&scheduler);
      concept_r7e__native_worker_native_wake_proceed(&scheduler);
      if (WaitForSingleObject(thread, 30000) != WAIT_OBJECT_0) return 4;
      CloseHandle(thread);
      if (args.decisions != 2) return 5;
      if (concept_r7e__native_worker_native_wake_final_state(&scheduler) != 0) return 6;
    }
  }
  return 0;
}`
	runFoundationNativeHarness(t, outputs, "r7e_wake_races.c", wakeHarness)
}
