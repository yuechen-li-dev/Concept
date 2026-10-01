package concept

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type innateScaleFixture struct{ path, source, group string }

func innateScaleAnalyze(fixture innateScaleFixture, artifacts map[string][]byte, metrics *evt1InnateMetrics) error {
	local, err := parseSyntaxModule(fixture.path, fixture.source)
	if err != nil {
		return err
	}
	module, _, err := composeSemanticModules(local, artifacts)
	if err != nil {
		return err
	}
	if err := evt1MaterializeGeneratedDeclarations(&module); err != nil {
		return err
	}
	if err := resolveNamespaceSymbols(&module); err != nil {
		return err
	}
	_, err = evt1AnalyzeModule(module, evt1AnalysisOptions{innateMetrics: metrics})
	return err
}

func logInnateScale(t *testing.T, label string, modules int, wall time.Duration, samples []evt1InnateSample) {
	t.Helper()
	var wait, execution time.Duration
	var peak evt1ComptimeUsage
	for _, sample := range samples {
		wait += sample.Wait
		execution += sample.Execution
		peak.Fuel = max(peak.Fuel, sample.Usage.Fuel)
		peak.Depth = max(peak.Depth, sample.Usage.Depth)
		peak.Loop = max(peak.Loop, sample.Usage.Loop)
		peak.Array = max(peak.Array, sample.Usage.Array)
	}
	t.Logf("%s modules=%d evaluations=%d validation_wall=%s lock_wait_sum=%s locked_execution_sum=%s max_fuel=%d max_depth=%d max_loop=%d max_array_literal=%d", label, modules, len(samples), wall, wait, execution, peak.Fuel, peak.Depth, peak.Loop, peak.Array)
	if len(samples) == 0 || peak.Fuel == 0 || peak.Depth == 0 || peak.Fuel > evt1ComptimeMaxFuel || peak.Depth > evt1ComptimeMaxCallDepth || peak.Loop > evt1ComptimeMaxLoopBound {
		t.Fatalf("unexpected innate usage: %+v, evaluations=%d", peak, len(samples))
	}
}

func TestR9aInnateLibraryScale(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	var fixtures []innateScaleFixture
	var imports []string
	for _, group := range []string{"Standard", "DragonGod", "Golden", "Vulkan"} {
		err := filepath.WalkDir(filepath.Join(root, group), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".concept") || entry.Name() == "manifest.concept" {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			module, err := parseSyntaxModule(filepath.ToSlash(path), string(body))
			if err != nil {
				return err
			}
			imports = append(imports, module.Imports...)
			fixtures = append(fixtures, innateScaleFixture{filepath.ToSlash(path), string(body), group})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources([]string{root, filepath.Join(root, "Standard"), filepath.Join(root, "Vulkan", "concept")}, imports)
	if err != nil {
		t.Fatal(err)
	}
	largest := append([]innateScaleFixture(nil), fixtures...)
	sort.Slice(largest, func(i, j int) bool { return len(largest[i].source) > len(largest[j].source) })
	for _, fixture := range largest[:min(5, len(largest))] {
		t.Logf("largest module %s bytes=%d", fixture.path, len(fixture.source))
	}
	for _, group := range []string{"Standard", "DragonGod", "Golden", "Vulkan"} {
		metrics := &evt1InnateMetrics{}
		start := time.Now()
		count := 0
		for _, fixture := range fixtures {
			if fixture.group != group {
				continue
			}
			if err := innateScaleAnalyze(fixture, artifacts, metrics); err != nil {
				t.Fatalf("%s: %v", fixture.path, err)
			}
			count++
		}
		logInnateScale(t, group, count, time.Since(start), metrics.snapshot())
	}
	// Equal workload: validate every module four times, first serially, then
	// with four workers. Each worker owns its parsed/composed environment.
	for _, workers := range []int{1, 4} {
		metrics := &evt1InnateMetrics{}
		jobs := make(chan innateScaleFixture)
		errs := make(chan error, len(fixtures)*4)
		var wg sync.WaitGroup
		start := time.Now()
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for fixture := range jobs {
					if err := innateScaleAnalyze(fixture, artifacts, metrics); err != nil {
						errs <- fmt.Errorf("%s: %w", fixture.path, err)
					}
				}
			}()
		}
		for repetition := 0; repetition < 4; repetition++ {
			for _, fixture := range fixtures {
				jobs <- fixture
			}
		}
		close(jobs)
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
		samples := metrics.snapshot()
		logInnateScale(t, fmt.Sprintf("equal-workload workers=%d", workers), len(fixtures)*4, time.Since(start), samples)
		keys := map[string]bool{}
		for _, sample := range samples {
			keys[sample.Module+"/"+sample.Predicate+"/"+sample.Subject] = true
		}
		t.Logf("workers=%d repeated predicate/subject candidates=%d/%d; keys are observation identities, not safe cache keys", workers, len(samples)-len(keys), len(samples))
	}
}

func TestR9aComptimeUsageAndLimitDiagnostics(t *testing.T) {
	source := "module Measure;\nprofile Core;\ncomptime int Count() { int<array>[3] items = [1,2,3]; int result = 0; for (item in items) { result = result + item; } return result; }\n"
	module, err := Parse("measure.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	var usage evt1ComptimeUsage
	value, err := evt1InvokeComptimeFunctionOnMeasured(env, env, "Count", nil, Span{Line: 3}, &usage)
	if err != nil || value.IntValue != 6 || usage.Array != 3 || usage.Loop != 3 || usage.Depth < 2 || usage.Fuel <= 0 {
		t.Fatalf("measurement: value=%v usage=%+v err=%v", value, usage, err)
	}
	state := newEVT1ComptimeState(env)
	if err := state.push("comptime fn Count"); err != nil {
		t.Fatal(err)
	}
	err = state.spend(Span{Line: 3}, evt1ComptimeMaxFuel+1)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4204" || !strings.Contains(diagnostic.Message, "used 4097, limit 4096") || !strings.Contains(diagnostic.Message, "Count") || diagnostic.Span.Line != 3 {
		t.Fatalf("fuel diagnostic: %v", err)
	}
	for i := 0; i < evt1ComptimeMaxCallDepth; i++ {
		err = state.push("comptime fn Count")
	}
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4211" || !strings.Contains(diagnostic.Message, "depth 33 exceeds limit 32") || !strings.Contains(diagnostic.Message, "Count") {
		t.Fatalf("depth diagnostic: %v", err)
	}
}
