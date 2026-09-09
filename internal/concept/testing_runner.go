package concept

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const TestManifestSchema = "concept-test-manifest.v1"
const TestResultsSchema = "concept-test-results.v1"

type TestKind string

const (
	TestFact      TestKind = "fact"
	TestTheory    TestKind = "theory"
	TestBenchmark TestKind = "benchmark"
	TestProphecy  TestKind = "prophecy"
)

type TestArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type TestParameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type TestDeclaration struct {
	TestID      string          `json:"test_id"`
	Kind        TestKind        `json:"kind"`
	Function    string          `json:"function"`
	Source      string          `json:"source"`
	SourceLine  int             `json:"source_line"`
	Artifacts   []TestArtifact  `json:"artifacts,omitempty"`
	Async       bool            `json:"async"`
	Foretold    bool            `json:"foretold"`
	Parameters  []TestParameter `json:"theory_parameters,omitempty"`
	module      Module
	function    FunctionDecl
	sourceBytes []byte
	sourcePath  string
}

type TestManifest struct {
	Schema   string            `json:"schema"`
	Compiler string            `json:"compiler"`
	Root     string            `json:"root"`
	Tests    []TestDeclaration `json:"tests"`
}

type TestFailure struct {
	Kind         string `json:"kind"`
	Message      string `json:"message"`
	SourceFile   string `json:"source_file"`
	SourceLine   int    `json:"source_line"`
	Expected     string `json:"expected,omitempty"`
	Actual       string `json:"actual,omitempty"`
	ExpectedType string `json:"expected_type,omitempty"`
	ActualType   string `json:"actual_type,omitempty"`
	ErrorPayload string `json:"error_payload,omitempty"`
}

type BenchmarkEvidence struct {
	Iterations  int   `json:"iterations"`
	TotalNanos  int64 `json:"total_nanos"`
	MeanNanos   int64 `json:"mean_nanos"`
	MinNanos    int64 `json:"min_nanos"`
	MaxNanos    int64 `json:"max_nanos"`
	MedianNanos int64 `json:"median_nanos"`
}

type TestResult struct {
	TestID           string             `json:"test_id"`
	Kind             TestKind           `json:"kind"`
	SourceFile       string             `json:"source_file"`
	SourceLine       int                `json:"source_line"`
	Status           string             `json:"status"`
	CaseIndex        *int               `json:"case_index,omitempty"`
	BoundValues      map[string]any     `json:"bound_values,omitempty"`
	StartTime        string             `json:"start_time"`
	DurationNanos    int64              `json:"duration_nanos"`
	ProcessExitCode  *int               `json:"process_exit_code,omitempty"`
	TerminationKind  string             `json:"termination_kind,omitempty"`
	TerminationCode  string             `json:"termination_code,omitempty"`
	PanicReason      string             `json:"panic_reason,omitempty"`
	Stdout           string             `json:"stdout,omitempty"`
	Stderr           string             `json:"stderr,omitempty"`
	Artifacts        []TestArtifact     `json:"artifacts,omitempty"`
	LastCheckpoints  []string           `json:"last_checkpoints,omitempty"`
	BuildIdentity    string             `json:"build_identity"`
	CompilerIdentity string             `json:"compiler_identity"`
	TargetIdentity   string             `json:"target_identity"`
	Failure          *TestFailure       `json:"failure,omitempty"`
	Benchmark        *BenchmarkEvidence `json:"benchmark,omitempty"`
}

type TestRun struct {
	Schema     string       `json:"schema"`
	Compiler   string       `json:"compiler"`
	Results    []TestResult `json:"results"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	Fulfilled  int          `json:"prophecy_fulfilled"`
	Benchmarks int          `json:"benchmarks"`
}

type TestRunOptions struct {
	Root                string
	Filter              string
	ResultsDir          string
	Timeout             time.Duration
	BenchmarkWarmup     int
	BenchmarkIterations int
}

func DiscoverTests(root string) (TestManifest, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return TestManifest{}, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return TestManifest{}, err
	}
	projectRoot := absRoot
	var paths []string
	if !info.IsDir() {
		if !strings.HasSuffix(strings.ToLower(absRoot), ".concept_test") {
			return TestManifest{}, fmt.Errorf("test source must use .concept_test: %s", absRoot)
		}
		projectRoot = filepath.Dir(absRoot)
		paths = append(paths, absRoot)
	} else {
		err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && path != absRoot {
				switch strings.ToLower(entry.Name()) {
				case ".git", ".test-results", "testdata", ".zig-cache", "zig-cache":
					return filepath.SkipDir
				}
			}
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".concept_test") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return TestManifest{}, err
		}
	}
	sort.Slice(paths, func(i, j int) bool { return filepath.ToSlash(paths[i]) < filepath.ToSlash(paths[j]) })
	manifest := TestManifest{Schema: TestManifestSchema, Compiler: CompilerID, Root: filepath.ToSlash(projectRoot)}
	seen := map[string]bool{}
	for _, path := range paths {
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return TestManifest{}, readErr
		}
		module, parseErr := ParseWithBuiltSemanticModuleRoots(filepath.ToSlash(path), string(body), []string{filepath.Dir(path)})
		if parseErr != nil {
			return TestManifest{}, fmt.Errorf("%s: %w", filepath.ToSlash(path), parseErr)
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			return TestManifest{}, relErr
		}
		rel = filepath.ToSlash(rel)
		for _, fn := range module.Functions {
			kind, foretold, artifactNames, annotated := evt1TestMetadata(fn)
			if !annotated {
				continue
			}
			id := strings.TrimSuffix(rel, ".concept_test") + "::" + fn.Name
			if seen[id] {
				return TestManifest{}, evt1Diagnostic("TEST_DUPLICATE_IDENTITY", "duplicate test identity "+id, fn.Span)
			}
			seen[id] = true
			decl := TestDeclaration{TestID: id, Kind: kind, Function: fn.Name, Source: rel, SourceLine: fn.Span.Line, Async: fn.Async, Foretold: foretold, module: module, function: fn, sourceBytes: body, sourcePath: path}
			for _, p := range fn.Params {
				decl.Parameters = append(decl.Parameters, TestParameter{Name: p.Name, Type: p.Type.String()})
			}
			artifactSeen := map[string]bool{}
			for _, name := range artifactNames {
				resolved, resolveErr := resolveTestArtifact(projectRoot, filepath.Dir(path), name)
				if resolveErr != nil {
					return TestManifest{}, resolveErr
				}
				if artifactSeen[resolved] {
					return TestManifest{}, evt1Diagnostic("TEST_ARTIFACT_DUPLICATE", "duplicate artifact path "+name, fn.Span)
				}
				artifactSeen[resolved] = true
				contents, readErr := os.ReadFile(resolved)
				if readErr != nil {
					return TestManifest{}, evt1Diagnostic("TEST_ARTIFACT_MISSING", fmt.Sprintf("required artifact %s is unavailable", name), fn.Span)
				}
				sum := sha256.Sum256(contents)
				artifactRel, _ := filepath.Rel(projectRoot, resolved)
				decl.Artifacts = append(decl.Artifacts, TestArtifact{Path: filepath.ToSlash(artifactRel), SHA256: hex.EncodeToString(sum[:])})
			}
			manifest.Tests = append(manifest.Tests, decl)
		}
	}
	return manifest, nil
}

func evt1TestMetadata(fn FunctionDecl) (TestKind, bool, []string, bool) {
	var kind TestKind
	foretold, annotated := false, false
	var artifacts []string
	for _, attribute := range fn.Attributes {
		annotated = true
		switch attribute.Name {
		case "fact", "theory", "benchmark", "prophecy":
			kind = TestKind(attribute.Name)
		case "foretold":
			foretold = true
		case "artifact":
			artifacts = append(artifacts, attribute.Args[0].(*StringLiteral).Value)
		}
	}
	return kind, foretold, artifacts, annotated
}

func resolveTestArtifact(root, sourceDir, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", evt1Diagnostic("TEST_ARTIFACT_OUTSIDE_ROOT", "artifact path must be relative", Span{})
	}
	resolved, err := filepath.Abs(filepath.Join(sourceDir, filepath.FromSlash(name)))
	if err != nil {
		return "", err
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedReal, err := filepath.EvalSymlinks(resolved)
	if err == nil {
		resolved = resolvedReal
	} else if !os.IsNotExist(err) {
		return "", err
	}
	rel, err := filepath.Rel(rootResolved, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", evt1Diagnostic("TEST_ARTIFACT_OUTSIDE_ROOT", "artifact path escapes the test root", Span{})
	}
	return resolved, nil
}

func MarshalTestManifest(manifest TestManifest) ([]byte, error) {
	copy := manifest
	copy.Tests = append([]TestDeclaration{}, manifest.Tests...)
	for i := range copy.Tests {
		copy.Tests[i].module = Module{}
		copy.Tests[i].function = FunctionDecl{}
		copy.Tests[i].sourceBytes = nil
		copy.Tests[i].sourcePath = ""
	}
	return json.MarshalIndent(copy, "", "  ")
}

func RunTests(manifest TestManifest, options TestRunOptions) (TestRun, error) {
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	if options.BenchmarkWarmup == 0 {
		options.BenchmarkWarmup = 1
	}
	if options.BenchmarkIterations <= 0 {
		options.BenchmarkIterations = 5
	}
	if options.ResultsDir == "" {
		options.ResultsDir = filepath.Join(filepath.FromSlash(manifest.Root), ".test-results")
	}
	run := TestRun{Schema: TestResultsSchema, Compiler: CompilerID}
	for _, test := range manifest.Tests {
		if options.Filter != "" && !strings.Contains(strings.ToLower(test.TestID), strings.ToLower(options.Filter)) {
			continue
		}
		cases, err := evt1TheoryCases(test)
		if err != nil {
			return run, err
		}
		if test.Kind != TestTheory {
			cases = [][]any{nil}
		}
		for caseIndex, values := range cases {
			result := runOneTest(test, values, caseIndex, options)
			run.Results = append(run.Results, result)
			switch result.Status {
			case "PASS":
				run.Passed++
			case "PROPHECY_FULFILLED":
				run.Fulfilled++
			case "BENCHMARK":
				run.Benchmarks++
			default:
				run.Failed++
			}
			if test.Foretold {
				if err := writeForetoldEvidence(options.ResultsDir, result); err != nil {
					return run, err
				}
			}
		}
	}
	if err := os.MkdirAll(options.ResultsDir, 0o755); err != nil {
		return run, err
	}
	manifestBytes, err := MarshalTestManifest(manifest)
	if err != nil {
		return run, err
	}
	if err := os.WriteFile(filepath.Join(options.ResultsDir, "manifest.json"), append(manifestBytes, '\n'), 0o644); err != nil {
		return run, err
	}
	resultBytes, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return run, err
	}
	if err := os.WriteFile(filepath.Join(options.ResultsDir, "results.json"), append(resultBytes, '\n'), 0o644); err != nil {
		return run, err
	}
	return run, nil
}

func evt1TheoryCases(test TestDeclaration) ([][]any, error) {
	if test.Kind != TestTheory {
		return nil, nil
	}
	var dataPath string
	for _, artifact := range test.Artifacts {
		if strings.EqualFold(filepath.Ext(artifact.Path), ".json") {
			dataPath = filepath.Join(filepath.FromSlash(filepath.Dir(test.sourcePath)), filepath.Base(filepath.FromSlash(artifact.Path)))
			break
		}
	}
	// Artifact paths in the manifest are root-relative; recover the already-safe
	// source-relative path directly from metadata for execution.
	for _, attribute := range test.function.Attributes {
		if attribute.Name == "artifact" {
			name := attribute.Args[0].(*StringLiteral).Value
			if strings.EqualFold(filepath.Ext(name), ".json") {
				dataPath = filepath.Join(filepath.Dir(test.sourcePath), filepath.FromSlash(name))
				break
			}
		}
	}
	if dataPath == "" {
		return nil, evt1Diagnostic("TEST_THEORY_ARTIFACT_FORMAT", "theory requires a JSON artifact", test.function.Span)
	}
	body, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, err
	}
	var rows [][]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, evt1Diagnostic("TEST_THEORY_ARTIFACT_INVALID", "theory JSON must be an array of positional arrays", test.function.Span)
	}
	for i, row := range rows {
		if len(row) != len(test.function.Params) {
			return nil, evt1Diagnostic("TEST_THEORY_ROW_ARITY", fmt.Sprintf("theory row %d has %d value(s), expected %d", i, len(row), len(test.function.Params)), test.function.Span)
		}
		for j, value := range row {
			if _, err := evt1TestCLiteral(test.function.Params[j].Type, value); err != nil {
				return nil, evt1Diagnostic("TEST_THEORY_ROW_TYPE", fmt.Sprintf("theory row %d parameter %s: %v", i, test.function.Params[j].Name, err), test.function.Params[j].Span)
			}
		}
	}
	return rows, nil
}

func runOneTest(test TestDeclaration, values []any, caseIndex int, options TestRunOptions) TestResult {
	start := time.Now().UTC()
	result := TestResult{TestID: test.TestID, Kind: test.Kind, SourceFile: test.Source, SourceLine: test.SourceLine, StartTime: start.Format(time.RFC3339Nano), Artifacts: test.Artifacts, CompilerIdentity: CompilerID, TargetIdentity: runtime.GOOS + "/" + runtime.GOARCH}
	if test.Kind == TestTheory {
		result.CaseIndex = &caseIndex
		result.TestID += fmt.Sprintf("[%d]", caseIndex)
		result.BoundValues = map[string]any{}
		for i, p := range test.function.Params {
			result.BoundValues[p.Name] = values[i]
		}
	}
	outputs, err := Generate(test.module, test.sourceBytes)
	if err != nil {
		return failedTestResult(result, start, "compile", err.Error(), test)
	}
	buildHash := sha256.New()
	keys := make([]string, 0, len(outputs))
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		buildHash.Write(outputs[key])
	}
	result.BuildIdentity = hex.EncodeToString(buildHash.Sum(nil))
	temp, err := os.MkdirTemp("", "concept-test-")
	if err != nil {
		return failedTestResult(result, start, "runner", err.Error(), test)
	}
	defer os.RemoveAll(temp)
	if err := Write(temp, outputs); err != nil {
		return failedTestResult(result, start, "runner", err.Error(), test)
	}
	base := evt1OutputBase(test.sourcePath)
	harness := evt1TestHarness(test, values, base)
	harnessPath := filepath.Join(temp, "test_harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0o644); err != nil {
		return failedTestResult(result, start, "runner", err.Error(), test)
	}
	executable := filepath.Join(temp, "concept-test")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	compiler, args, err := evt1TestCompiler(temp, filepath.Join(temp, base+".generated.c"), harnessPath, executable)
	if err != nil {
		return failedTestResult(result, start, "compiler-unavailable", err.Error(), test)
	}
	build := exec.Command(compiler, args...)
	if output, err := build.CombinedOutput(); err != nil {
		return failedTestResult(result, start, "native-compile", string(output), test)
	}
	result.TargetIdentity += "/" + filepath.Base(compiler)
	if test.Kind == TestBenchmark {
		for i := 0; i < options.BenchmarkWarmup; i++ {
			stdout, stderr, code, _ := runTestProcess(executable, options.Timeout)
			if code != 0 {
				result.Stdout, result.Stderr = stdout, stderr
				result.ProcessExitCode = &code
				return failedTestResult(result, start, "benchmark-warmup", strings.TrimSpace(stderr), test)
			}
		}
		durations := make([]int64, 0, options.BenchmarkIterations)
		for i := 0; i < options.BenchmarkIterations; i++ {
			stdout, stderr, code, duration := runTestProcess(executable, options.Timeout)
			if code != 0 {
				result.Stdout, result.Stderr = stdout, stderr
				result.ProcessExitCode = &code
				return failedTestResult(result, start, "benchmark", strings.TrimSpace(stderr), test)
			}
			durations = append(durations, duration.Nanoseconds())
		}
		result.Status, result.Benchmark = "BENCHMARK", benchmarkEvidence(durations)
		result.DurationNanos = time.Since(start).Nanoseconds()
		return result
	}
	stdout, stderr, exitCode, duration := runTestProcess(executable, options.Timeout)
	result.Stdout, result.Stderr, result.DurationNanos = stdout, stderr, duration.Nanoseconds()
	result.ProcessExitCode = &exitCode
	result.LastCheckpoints = parseCheckpoints(stderr)
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, "Concept panic") {
			result.PanicReason = strings.TrimSpace(line)
		}
	}
	if exitCode != 0 {
		result.TerminationKind = "abnormal_exit"
		result.TerminationCode = strconv.Itoa(exitCode)
		if strings.Contains(stderr, "CONCEPT_TEST_TIMEOUT") {
			result.TerminationKind = "timeout"
		}
	}
	if test.Kind == TestProphecy {
		if exitCode != 0 && !strings.Contains(stderr, "CONCEPT_TEST_TIMEOUT") {
			result.Status = "PROPHECY_FULFILLED"
			return result
		}
		return failedTestResult(result, start, "prophecy-unfulfilled", "prophecy returned normally", test)
	}
	if exitCode == 0 {
		result.Status = "PASS"
		return result
	}
	return failedTestResult(result, start, classifyTestFailure(stderr), strings.TrimSpace(stderr), test)
}

func evt1TestCompiler(includeDir, generated, harness, executable string) (string, []string, error) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		return "", nil, errors.New("gcc or clang is required for concept test")
	}
	return compiler, []string{"-std=c11", "-Wall", "-Wextra", "-I", includeDir, generated, harness, "-lm", "-o", executable}, nil
}

func evt1TestHarness(test TestDeclaration, values []any, base string) string {
	args := make([]string, len(values))
	for i, value := range values {
		args[i], _ = evt1TestCLiteral(test.function.Params[i].Type, value)
	}
	call := evt1FunctionSymbol(base, test.function.Name) + "(" + strings.Join(args, ", ") + ")"
	if test.Async {
		call = "concept_async_operation operation = " + call + ";\n  int steps = 0;\n  while (!concept_async_complete(&operation) && steps < 100000) { concept_async_step(&operation); ++steps; }\n  if (!concept_async_complete(&operation)) return 124"
	}
	return fmt.Sprintf("#include \"%s.generated.h\"\n\nint main(void) {\n  %s;\n  return 0;\n}\n", base, call)
}

func evt1TestCLiteral(t Type, value any) (string, error) {
	switch t.Name {
	case "int":
		v, ok := value.(float64)
		if !ok || v < -2147483648 || v > 2147483647 || v != float64(int64(v)) {
			return "", fmt.Errorf("expected int")
		}
		return strconv.FormatInt(int64(v), 10), nil
	case "uint":
		v, ok := value.(float64)
		if !ok || v < 0 || v > 4294967295 || v != float64(uint64(v)) {
			return "", fmt.Errorf("expected non-negative integer")
		}
		return strconv.FormatUint(uint64(v), 10) + "u", nil
	case "byte":
		v, ok := value.(float64)
		if !ok || v < 0 || v > 255 || v != float64(uint64(v)) {
			return "", fmt.Errorf("expected byte")
		}
		return strconv.FormatUint(uint64(v), 10) + "u", nil
	case "float":
		v, ok := value.(float64)
		if !ok {
			return "", fmt.Errorf("expected number")
		}
		literal := strconv.FormatFloat(v, 'g', -1, 64)
		if !strings.ContainsAny(literal, ".eE") {
			literal += ".0"
		}
		return literal + "f", nil
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return "", fmt.Errorf("expected bool")
		}
		if v {
			return "true", nil
		}
		return "false", nil
	case "string":
		v, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("expected string")
		}
		return strconv.Quote(v), nil
	default:
		return "", fmt.Errorf("unsupported parameter type %s", t.String())
	}
}

func runTestProcess(executable string, timeout time.Duration) (string, string, int, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		stderr.WriteString("\nCONCEPT_TEST_TIMEOUT\n")
		return stdout.String(), stderr.String(), 124, duration
	}
	if err == nil {
		return stdout.String(), stderr.String(), 0, duration
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return stdout.String(), stderr.String(), exit.ExitCode(), duration
	}
	return stdout.String(), stderr.String(), 125, duration
}

func failedTestResult(result TestResult, start time.Time, kind, message string, test TestDeclaration) TestResult {
	result.Status = "FAIL"
	result.DurationNanos = time.Since(start).Nanoseconds()
	result.Failure = &TestFailure{Kind: kind, Message: message, SourceFile: test.Source, SourceLine: test.SourceLine}
	if failure := parseAssertionFailure(result.Stderr, test.Source); failure != nil {
		result.Failure = failure
	}
	return result
}

func parseAssertionFailure(stderr, source string) *TestFailure {
	for _, line := range strings.Split(stderr, "\n") {
		if !strings.HasPrefix(line, "CONCEPT_TEST_ASSERT|") {
			continue
		}
		parts := strings.Split(strings.TrimSpace(line), "|")
		if len(parts) < 6 {
			return nil
		}
		lineNumber, _ := strconv.Atoi(parts[2])
		failure := &TestFailure{Kind: "assertion:" + parts[1], Message: parts[4], SourceFile: source, SourceLine: lineNumber}
		for _, field := range parts[5:] {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case "actual", "result_tag":
				failure.Actual = value
			case "expected":
				failure.Expected = value
			case "error":
				failure.ErrorPayload = value
			case "success":
				failure.Actual = value
			}
		}
		return failure
	}
	return nil
}

func classifyTestFailure(stderr string) string {
	if strings.Contains(stderr, "CONCEPT_TEST_ASSERT|") {
		return "assertion"
	}
	if strings.Contains(stderr, "panic") {
		return "unexpected-panic"
	}
	if strings.Contains(stderr, "TIMEOUT") {
		return "timeout"
	}
	return "abnormal-exit"
}

func parseCheckpoints(stderr string) []string {
	var checkpoints []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "CONCEPT_TEST_CHECKPOINT|") {
			checkpoints = append(checkpoints, line)
			if len(checkpoints) > 16 {
				checkpoints = checkpoints[len(checkpoints)-16:]
			}
		}
	}
	return checkpoints
}

func benchmarkEvidence(durations []int64) *BenchmarkEvidence {
	sorted := append([]int64{}, durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var total int64
	for _, value := range durations {
		total += value
	}
	return &BenchmarkEvidence{Iterations: len(durations), TotalNanos: total, MeanNanos: total / int64(len(durations)), MinNanos: sorted[0], MaxNanos: sorted[len(sorted)-1], MedianNanos: sorted[len(sorted)/2]}
}

func writeForetoldEvidence(resultsDir string, result TestResult) error {
	dir := filepath.Join(resultsDir, "prophecy", sanitizeTestID(result.TestID), "latest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "result.json"), append(body, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "stdout.txt"), []byte(result.Stdout), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "stderr.txt"), []byte(result.Stderr), 0o644)
}

func sanitizeTestID(id string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, id)
}
