package concept

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7n2ImmovableStorageConstructionAndFailureCleanup(t *testing.T) {
	source := `module StationaryStorage;
profile Core;
struct SystemMemory {}
extern "C" void ObservePartDrop(int value);
extern "C" void ObserveCheckpoint(int stage);
enum BuildError { Failed }
struct Part { int value; }
void Drop(owned Part part) { ObservePartDrop(part.value); }
immovable struct Pinned { owned Part part; int key; }

Result<int, BuildError> Key(bool fail)
{
    if (fail) { return Result::Error(BuildError::Failed); }
    return Result::Ok(9);
}

Result<int, BuildError> Build(bool fail)
{
    int<array>[2] backing = [0 ...];
    Storage<Pinned> storage = bind<Pinned>(AddressOf<SystemMemory>(ref backing), SizeOf<Pinned>());
    Initialize(storage, Pinned{Part{7}, Key(fail)?});
    ref Pinned pinned = Value(storage);
    int key = pinned.key;
    Destroy(storage);
    return Result::Ok(key);
}

int Main()
{
    discard Build(true);
    ObserveCheckpoint(1);
    discard Build(false);
    ObserveCheckpoint(2);
    return 1;
}
`
	module, err := Parse("stationarystorage.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["stationarystorage.generated.c"])
	if strings.Contains(body, "*storage =") || strings.Contains(body, "malloc") || strings.Contains(body, "memcpy") {
		t.Fatal("stationary storage path created a complete temporary, allocation, or relocation")
	}
	if !strings.Contains(body, ").part =") || !strings.Contains(body, ").key =") {
		t.Fatal("stationary storage path did not initialize the final object field by field")
	}
	harness := `#include "stationarystorage.generated.h"
#include <stdio.h>
static int drops = 0;
static int value_sum = 0;
static int bad = 0;
void ObservePartDrop(int value) { drops += 1; value_sum += value; }
void ObserveCheckpoint(int stage) { if (drops != stage || value_sum != stage * 7) bad = 1; }
int main(void) {
    int result = concept_stationary_storage_main();
    if (!(result == 1 && drops == 2 && value_sum == 14 && bad == 0)) printf("result=%d drops=%d sum=%d bad=%d\n", result, drops, value_sum, bad);
    return result == 1 && drops == 2 && value_sum == 14 && bad == 0 ? 0 : 1;
}
`
	runFoundationNativeHarness(t, outputs, "stationarystorage_harness.c", harness)
	compiler, err := exec.LookPath("clang")
	if err != nil {
		compiler, err = exec.LookPath("gcc")
	}
	if err == nil {
		dir := t.TempDir()
		if err := Write(dir, outputs); err != nil {
			t.Fatal(err)
		}
		command := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-Wno-unused-function", "-fsyntax-only", filepath.Join(dir, "stationarystorage.generated.c"))
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("strict C11 compile failed: %v\n%s", err, output)
		}
	}
}
