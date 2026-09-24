package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
)

// Native projects use ordinary immutable Concept data. This adapter only reads
// that data; the selected C/C++ compiler remains authoritative for native code.
type NativeProject struct {
	Name        string           `json:"name"`
	Author      string           `json:"author"`
	Version     PackageVersion   `json:"version"`
	Toolchain   string           `json:"toolchain"`
	SourceRoots []string         `json:"source_roots"`
	Targets     []NativeTarget   `json:"targets"`
	Companions  []string         `json:"companions"`
	Tests       []string         `json:"tests"`
	ABI         []NativeABIClaim `json:"abi"`
	Root        string           `json:"-"`
}

type NativeABIClaim struct {
	Header    string   `json:"header"`
	TypeName  string   `json:"type_name"`
	Companion string   `json:"companion"`
	Size      int      `json:"size"`
	Alignment int      `json:"alignment"`
	Fields    []string `json:"fields"`
	Offsets   []int    `json:"offsets"`
}

type NativeABIEvidence struct {
	TypeName  string   `json:"type_name"`
	Origin    string   `json:"origin"`
	Size      int      `json:"size"`
	Alignment int      `json:"alignment"`
	Fields    []string `json:"fields"`
	Offsets   []int    `json:"offsets"`
}

type NativeABIReport struct {
	Schema                string              `json:"schema"`
	Compiler              string              `json:"compiler"`
	CompilerVersion       string              `json:"compiler_version"`
	Target                string              `json:"target"`
	BuildInputHash        string              `json:"build_input_hash"`
	SemanticCompanionHash string              `json:"semantic_companion_hash"`
	Evidence              []NativeABIEvidence `json:"evidence"`
}

type NativeTarget struct {
	Name       string         `json:"name"`
	Language   string         `json:"language"`
	Standard   string         `json:"standard"`
	Kind       string         `json:"kind"`
	Output     string         `json:"output"`
	Sources    []string       `json:"sources"`
	Includes   []string       `json:"includes"`
	Defines    []NativeDefine `json:"defines"`
	LinkInputs []string       `json:"link_inputs"`
}

type NativeDefine struct {
	Name     string `json:"name"`
	Value    string `json:"value,omitempty"`
	HasValue bool   `json:"has_value"`
}

type NativeCommand struct {
	Target  string   `json:"target"`
	Program string   `json:"program"`
	Args    []string `json:"args"`
}

type NativePlan struct {
	Schema                string               `json:"schema"`
	Project               string               `json:"project"`
	Toolchain             string               `json:"toolchain"`
	ToolchainVersion      string               `json:"toolchain_version"`
	BuildInputHash        string               `json:"build_input_hash"`
	SemanticCompanionHash string               `json:"semantic_companion_hash"`
	Commands              []NativeCommand      `json:"commands"`
	ABIProbes             []NativeABIProbePlan `json:"abi_probes,omitempty"`
}

type NativeABIProbePlan struct {
	Compiler string   `json:"compiler"`
	Header   string   `json:"header"`
	TypeName string   `json:"type_name"`
	Fields   []string `json:"fields"`
	Mode     string   `json:"mode"`
}

type NativeBuildResult struct {
	Plan               NativePlan        `json:"plan"`
	NativeOutputHashes map[string]string `json:"native_output_hashes"`
}

func LoadNativeProject(root string) (NativeProject, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return NativeProject{}, err
	}
	path := filepath.Join(root, "manifest.concept")
	body, err := os.ReadFile(path)
	if err != nil {
		return NativeProject{}, err
	}
	module, err := Parse(filepath.ToSlash(path), string(body))
	if err != nil {
		return NativeProject{}, fmt.Errorf("NATIVE_MANIFEST_SEMANTIC_INVALID: %w", err)
	}
	var declaration Expr
	for _, decl := range module.ComptimeDecls {
		if decl.Name == "Native" && decl.Type.Name == "NativeProjectManifest" {
			declaration = decl.Value
		}
	}
	fields, err := nativeFields(declaration, "NativeProjectManifest", 9)
	if err != nil {
		return NativeProject{}, err
	}
	name, err := nativeString(fields[0])
	if err != nil {
		return NativeProject{}, err
	}
	author, err := nativeString(fields[1])
	if err != nil {
		return NativeProject{}, err
	}
	versionFields, err := nativeFields(fields[2], "NativeVersion", 3)
	if err != nil {
		return NativeProject{}, err
	}
	major, err := nativeInt(versionFields[0])
	if err != nil {
		return NativeProject{}, err
	}
	minor, err := nativeInt(versionFields[1])
	if err != nil {
		return NativeProject{}, err
	}
	patch, err := nativeInt(versionFields[2])
	if err != nil {
		return NativeProject{}, err
	}
	toolchain, err := nativeEnum(fields[3], "NativeToolchain")
	if err != nil {
		return NativeProject{}, err
	}
	sourceRoots, err := nativeStrings(fields[4])
	if err != nil {
		return NativeProject{}, err
	}
	targets, err := nativeArray(fields[5])
	if err != nil {
		return NativeProject{}, err
	}
	companions, err := nativeStrings(fields[6])
	if err != nil {
		return NativeProject{}, err
	}
	tests, err := nativeStrings(fields[7])
	if err != nil {
		return NativeProject{}, err
	}
	project := NativeProject{Name: name, Author: author, Version: PackageVersion{major, minor, patch}, Toolchain: toolchain, SourceRoots: sourceRoots, Companions: companions, Tests: tests, Root: root}
	if name == "" || author == "" || major < 0 || minor < 0 || patch < 0 || len(targets) == 0 {
		return NativeProject{}, fmt.Errorf("NATIVE_MANIFEST_INVALID: project name and targets are required")
	}
	if toolchain != "Clang" && toolchain != "GCC" {
		return NativeProject{}, fmt.Errorf("NATIVE_TOOLCHAIN_UNSUPPORTED: %s", toolchain)
	}
	for _, sourceRoot := range sourceRoots {
		if err := nativeValidatePath(root, sourceRoot); err != nil {
			return NativeProject{}, err
		}
	}
	seen := map[string]bool{}
	for _, item := range targets {
		f, err := nativeFields(item, "NativeTarget", 9)
		if err != nil {
			return NativeProject{}, err
		}
		var t NativeTarget
		if t.Name, err = nativeString(f[0]); err != nil {
			return NativeProject{}, err
		}
		if t.Language, err = nativeEnum(f[1], "NativeLanguage"); err != nil {
			return NativeProject{}, err
		}
		if t.Standard, err = nativeEnum(f[2], "NativeStandard"); err != nil {
			return NativeProject{}, err
		}
		if t.Kind, err = nativeEnum(f[3], "NativeOutputKind"); err != nil {
			return NativeProject{}, err
		}
		if t.Output, err = nativeString(f[4]); err != nil {
			return NativeProject{}, err
		}
		if t.Sources, err = nativeStrings(f[5]); err != nil {
			return NativeProject{}, err
		}
		if t.Includes, err = nativeStrings(f[6]); err != nil {
			return NativeProject{}, err
		}
		defineItems, err := nativeArray(f[7])
		if err != nil {
			return NativeProject{}, err
		}
		for _, item := range defineItems {
			fields, err := nativeFields(item, "NativeDefine", 3)
			if err != nil {
				return NativeProject{}, err
			}
			name, err := nativeString(fields[0])
			if err != nil {
				return NativeProject{}, err
			}
			value, err := nativeString(fields[1])
			if err != nil {
				return NativeProject{}, err
			}
			hasValue, ok := fields[2].(*BoolLiteral)
			if !ok {
				return NativeProject{}, fmt.Errorf("NATIVE_DEFINE_INVALID: expected boolean hasValue")
			}
			if !nativeIdentifier(name) || strings.ContainsAny(value, "\r\n") {
				return NativeProject{}, fmt.Errorf("NATIVE_DEFINE_INVALID: %q", name)
			}
			t.Defines = append(t.Defines, NativeDefine{Name: name, Value: value, HasValue: hasValue.Value})
		}
		if t.LinkInputs, err = nativeStrings(f[8]); err != nil {
			return NativeProject{}, err
		}
		if len(t.LinkInputs) == 1 && t.LinkInputs[0] == "" {
			t.LinkInputs = nil
		}
		if t.Name == "" || seen[t.Name] || len(t.Sources) == 0 {
			return NativeProject{}, fmt.Errorf("NATIVE_TARGET_INVALID: target name must be unique and sources nonempty")
		}
		seen[t.Name] = true
		for _, source := range t.Sources {
			contained := false
			for _, sourceRoot := range sourceRoots {
				if source == sourceRoot || strings.HasPrefix(source, strings.TrimSuffix(sourceRoot, "/")+"/") {
					contained = true
					break
				}
			}
			if !contained {
				return NativeProject{}, fmt.Errorf("NATIVE_SOURCE_OUTSIDE_ROOTS: %s", source)
			}
		}
		if t.Language != "C" && t.Language != "Cpp" {
			return NativeProject{}, fmt.Errorf("NATIVE_LANGUAGE_UNSUPPORTED: %s", t.Language)
		}
		if (t.Language == "C" && t.Standard != "C11" && t.Standard != "C17") || (t.Language == "Cpp" && t.Standard != "Cpp17" && t.Standard != "Cpp20" && t.Standard != "Cpp23") {
			return NativeProject{}, fmt.Errorf("NATIVE_STANDARD_UNSUPPORTED: %s for %s", t.Standard, t.Language)
		}
		if t.Kind != "Executable" && t.Kind != "StaticLibrary" {
			return NativeProject{}, fmt.Errorf("NATIVE_OUTPUT_KIND_UNSUPPORTED: %s", t.Kind)
		}
		if t.Kind == "StaticLibrary" && len(t.LinkInputs) != 0 {
			return NativeProject{}, fmt.Errorf("NATIVE_TARGET_INVALID: static library %s cannot link external inputs", t.Name)
		}
		if err := nativeValidatePath(root, t.Output); err != nil {
			return NativeProject{}, err
		}
		for _, p := range append(append(append([]string{}, t.Sources...), t.Includes...), t.LinkInputs...) {
			if err := nativeValidatePath(root, p); err != nil {
				return NativeProject{}, err
			}
		}
		project.Targets = append(project.Targets, t)
	}
	for _, p := range append(append([]string{}, companions...), tests...) {
		if err := nativeValidatePath(root, p); err != nil {
			return NativeProject{}, err
		}
	}
	claims, err := nativeArray(fields[8])
	if err != nil {
		return NativeProject{}, err
	}
	for _, claim := range claims {
		f, err := nativeFields(claim, "NativeAbiClaim", 7)
		if err != nil {
			return NativeProject{}, err
		}
		var a NativeABIClaim
		if a.Header, err = nativeString(f[0]); err != nil {
			return NativeProject{}, err
		}
		if a.TypeName, err = nativeString(f[1]); err != nil {
			return NativeProject{}, err
		}
		if a.Companion, err = nativeString(f[2]); err != nil {
			return NativeProject{}, err
		}
		if a.Size, err = nativeInt(f[3]); err != nil {
			return NativeProject{}, err
		}
		if a.Alignment, err = nativeInt(f[4]); err != nil {
			return NativeProject{}, err
		}
		if a.Fields, err = nativeStrings(f[5]); err != nil {
			return NativeProject{}, err
		}
		if a.Offsets, err = nativeInts(f[6]); err != nil {
			return NativeProject{}, err
		}
		if err := nativeValidatePath(root, a.Header); err != nil {
			return NativeProject{}, err
		}
		if err := nativeValidatePath(root, a.Companion); err != nil {
			return NativeProject{}, err
		}
		if a.Size <= 0 || a.Alignment <= 0 || len(a.Fields) == 0 || len(a.Fields) != len(a.Offsets) {
			return NativeProject{}, fmt.Errorf("NATIVE_ABI_CLAIM_INVALID: %s", a.TypeName)
		}
		project.ABI = append(project.ABI, a)
	}
	return project, nil
}

func nativeFields(expr Expr, name string, count int) ([]Expr, error) {
	c, ok := expr.(*StructConstructExpr)
	if !ok || c.StructName != name || len(c.Args) != count {
		return nil, fmt.Errorf("NATIVE_MANIFEST_INVALID: expected %s with %d fields", name, count)
	}
	return c.Args, nil
}

func nativeString(expr Expr) (string, error) {
	s, ok := expr.(*StringLiteral)
	if !ok {
		return "", fmt.Errorf("NATIVE_MANIFEST_INVALID: expected string literal")
	}
	return s.Value, nil
}

func nativeEnum(expr Expr, name string) (string, error) {
	v, ok := expr.(*ConstructExpr)
	if !ok || v.EnumName != name {
		return "", fmt.Errorf("NATIVE_MANIFEST_INVALID: expected %s value", name)
	}
	return v.VariantName, nil
}

func nativeArray(expr Expr) ([]Expr, error) {
	a, ok := expr.(*ArrayLiteralExpr)
	if !ok {
		return nil, fmt.Errorf("NATIVE_MANIFEST_INVALID: expected array literal")
	}
	return a.Elements, nil
}

func nativeStrings(expr Expr) ([]string, error) {
	a, err := nativeArray(expr)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(a))
	for _, item := range a {
		s, err := nativeString(item)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func nativeInt(expr Expr) (int, error) {
	v, ok := expr.(*IntLiteral)
	if !ok {
		return 0, fmt.Errorf("NATIVE_MANIFEST_INVALID: expected integer literal")
	}
	n, ok := v.boundedInt()
	if !ok {
		return 0, fmt.Errorf("NATIVE_MANIFEST_INVALID: integer out of range")
	}
	return n, nil
}

func nativeInts(expr Expr) ([]int, error) {
	a, err := nativeArray(expr)
	if err != nil {
		return nil, err
	}
	result := make([]int, 0, len(a))
	for _, item := range a {
		n, err := nativeInt(item)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

func nativeValidatePath(root, relative string) error {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return fmt.Errorf("NATIVE_PATH_INVALID: %q", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("NATIVE_PATH_OUTSIDE_ROOT: %q", relative)
	}
	return nil
}

func nativeDigest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func nativeHashPart(sum hash.Hash, name string, body []byte) {
	fmt.Fprintf(sum, "%d:%s%d:", len(name), name, len(body))
	sum.Write(body)
}

func NativeBuildPlan(project NativeProject) (NativePlan, error) {
	compiler := "clang++"
	archiver := "llvm-ar"
	if project.Toolchain == "GCC" {
		compiler, archiver = "g++", "ar"
	}
	path, err := exec.LookPath(compiler)
	if err != nil {
		return NativePlan{}, fmt.Errorf("NATIVE_TOOLCHAIN_MISSING: %s: %w", compiler, err)
	}
	version, err := exec.Command(path, "--version").Output()
	if err != nil {
		return NativePlan{}, err
	}
	versionLine := strings.SplitN(strings.TrimSpace(string(version)), "\n", 2)[0]
	plan := NativePlan{Schema: "concept-native-plan.v1", Project: project.Name, Toolchain: project.Toolchain, ToolchainVersion: versionLine, Commands: []NativeCommand{}}
	manifest, err := os.ReadFile(filepath.Join(project.Root, "manifest.concept"))
	if err != nil {
		return NativePlan{}, err
	}
	input := sha256.New()
	nativeHashPart(input, "manifest.concept", manifest)
	nativeHashPart(input, "toolchain-version", []byte(versionLine))
	semantic := sha256.New()
	for _, p := range project.Companions {
		body, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(p)))
		if err != nil {
			return NativePlan{}, err
		}
		nativeHashPart(semantic, p, body)
	}
	plan.SemanticCompanionHash = hex.EncodeToString(semantic.Sum(nil))
	for _, claim := range project.ABI {
		plan.ABIProbes = append(plan.ABIProbes, NativeABIProbePlan{Compiler: compiler, Header: claim.Header, TypeName: claim.TypeName, Fields: append([]string{}, claim.Fields...), Mode: "compile-and-run"})
	}
	for _, target := range project.Targets {
		targetCompiler := compiler
		if target.Language == "C" {
			if project.Toolchain == "GCC" {
				targetCompiler = "gcc"
			} else {
				targetCompiler = "clang"
			}
		}
		if _, err := exec.LookPath(targetCompiler); err != nil {
			return NativePlan{}, fmt.Errorf("NATIVE_TOOLCHAIN_MISSING: %s: %w", targetCompiler, err)
		}
		std := map[string]string{"C11": "c11", "C17": "c17", "Cpp17": "c++17", "Cpp20": "c++20", "Cpp23": "c++23"}[target.Standard]
		objects := []string{}
		for i, source := range target.Sources {
			object := filepath.ToSlash(filepath.Join(".native-build", target.Name, fmt.Sprintf("%03d.o", i)))
			args := []string{"-std=" + std, "-c", source, "-o", object}
			for _, include := range target.Includes {
				args = append(args, "-I", include)
			}
			for _, define := range target.Defines {
				argument := "-D" + define.Name
				if define.HasValue {
					argument += "=" + define.Value
				}
				args = append(args, argument)
			}
			plan.Commands = append(plan.Commands, NativeCommand{Target: target.Name, Program: targetCompiler, Args: args})
			objects = append(objects, object)
			body, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(source)))
			if err != nil {
				return NativePlan{}, err
			}
			nativeHashPart(input, source, body)
		}
		// Conservative header dependency: every header in declared include roots is
		// hashed. Native compiler remains responsible for actual #include semantics.
		headers := []string{}
		headerRoots := append([]string{}, target.Includes...)
		for _, source := range target.Sources {
			headerRoots = append(headerRoots, filepath.ToSlash(filepath.Dir(filepath.FromSlash(source))))
		}
		sort.Strings(headerRoots)
		for i, include := range headerRoots {
			if i > 0 && include == headerRoots[i-1] {
				continue
			}
			err := filepath.WalkDir(filepath.Join(project.Root, filepath.FromSlash(include)), func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !entry.IsDir() && (strings.HasSuffix(path, ".h") || strings.HasSuffix(path, ".hpp")) {
					headers = append(headers, path)
				}
				return nil
			})
			if err != nil {
				return NativePlan{}, err
			}
		}
		sort.Strings(headers)
		headers = slices.Compact(headers)
		for _, header := range headers {
			body, err := os.ReadFile(header)
			if err != nil {
				return NativePlan{}, err
			}
			rel, _ := filepath.Rel(project.Root, header)
			nativeHashPart(input, filepath.ToSlash(rel), body)
		}
		link := append([]string{}, objects...)
		for _, dep := range target.LinkInputs {
			link = append(link, dep)
			body, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(dep)))
			if err != nil {
				return NativePlan{}, err
			}
			nativeHashPart(input, dep, body)
		}
		if target.Kind == "StaticLibrary" {
			if _, err := exec.LookPath(archiver); err != nil {
				return NativePlan{}, fmt.Errorf("NATIVE_TOOLCHAIN_MISSING: %s: %w", archiver, err)
			}
			plan.Commands = append(plan.Commands, NativeCommand{Target: target.Name, Program: archiver, Args: append([]string{"rcs", target.Output}, objects...)})
		} else {
			plan.Commands = append(plan.Commands, NativeCommand{Target: target.Name, Program: targetCompiler, Args: append(append([]string{}, link...), "-o", target.Output)})
		}
	}
	commandBody, err := json.Marshal(plan.Commands)
	if err != nil {
		return NativePlan{}, err
	}
	nativeHashPart(input, "command-plan", commandBody)
	claimBody, err := json.Marshal(project.ABI)
	if err != nil {
		return NativePlan{}, err
	}
	nativeHashPart(input, "abi-claims", claimBody)
	plan.BuildInputHash = hex.EncodeToString(input.Sum(nil))
	return plan, nil
}

func RunNativeBuild(project NativeProject, plan NativePlan) (NativeBuildResult, error) {
	result := NativeBuildResult{Plan: plan, NativeOutputHashes: map[string]string{}}
	for _, command := range plan.Commands {
		for i, arg := range command.Args {
			if arg == "-o" && i+1 < len(command.Args) {
				if err := os.MkdirAll(filepath.Join(project.Root, filepath.Dir(filepath.FromSlash(command.Args[i+1]))), 0o755); err != nil {
					return result, err
				}
			}
		}
		if command.Program == "llvm-ar" || command.Program == "ar" {
			if err := os.MkdirAll(filepath.Join(project.Root, filepath.Dir(filepath.FromSlash(command.Args[1]))), 0o755); err != nil {
				return result, err
			}
		}
		cmd := exec.Command(command.Program, command.Args...)
		cmd.Dir = project.Root
		if output, err := cmd.CombinedOutput(); err != nil {
			kind := "NATIVE_LINK_FAILED"
			if command.Program == "llvm-ar" || command.Program == "ar" {
				kind = "NATIVE_ARCHIVE_FAILED"
			}
			if slices.Contains(command.Args, "-c") {
				kind = "NATIVE_COMPILE_FAILED"
			}
			return result, fmt.Errorf("%s: target %s: %s %s: %w\n%s", kind, command.Target, command.Program, strings.Join(command.Args, " "), err, output)
		}
	}
	for _, target := range project.Targets {
		body, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(target.Output)))
		if err != nil {
			return result, err
		}
		result.NativeOutputHashes[target.Name] = nativeDigest(body)
	}
	metadata, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(project.Root, ".native-build"), 0o755); err != nil {
		return result, err
	}
	if err := os.WriteFile(filepath.Join(project.Root, ".native-build", "build.json"), append(metadata, '\n'), 0o644); err != nil {
		return result, err
	}
	return result, nil
}

func MarshalNativePlan(plan NativePlan) ([]byte, error) { return json.MarshalIndent(plan, "", "  ") }

// CheckNativeABI checks exact declared repr(C) fields against a probe compiled
// and executed by the selected native compiler. No C++ source is parsed here.
func CheckNativeABI(project NativeProject) error {
	plan, err := NativeBuildPlan(project)
	if err != nil {
		return err
	}
	compiler := "clang++"
	if project.Toolchain == "GCC" {
		compiler = "g++"
	}
	target, err := exec.Command(compiler, "-dumpmachine").Output()
	if err != nil {
		return fmt.Errorf("NATIVE_ABI_TARGET_UNKNOWN: %w", err)
	}
	report := NativeABIReport{Schema: "concept-native-abi.v1", Compiler: compiler, CompilerVersion: plan.ToolchainVersion, Target: strings.TrimSpace(string(target)) + "/" + runtime.GOOS + "/" + runtime.GOARCH, BuildInputHash: plan.BuildInputHash, SemanticCompanionHash: plan.SemanticCompanionHash, Evidence: []NativeABIEvidence{}}
	claimed := map[string]bool{}
	for _, claim := range project.ABI {
		claimed[claim.Companion+"|"+claim.TypeName] = true
	}
	for _, companion := range project.Companions {
		path := filepath.Join(project.Root, filepath.FromSlash(companion))
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		module, err := Parse(filepath.ToSlash(path), string(body))
		if err != nil {
			return err
		}
		for _, decl := range module.Structs {
			if evt1HasCRepr(decl.Attributes) && !claimed[companion+"|"+decl.Name] {
				return fmt.Errorf("NATIVE_ABI_CLAIM_MISSING: %s in %s requires a native layout claim", decl.Name, companion)
			}
		}
	}
	for _, claim := range project.ABI {
		path := filepath.Join(project.Root, filepath.FromSlash(claim.Companion))
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		module, err := Parse(filepath.ToSlash(path), string(body))
		if err != nil {
			return err
		}
		found := false
		geometryEnv := newSemanticEnv(&coreProfileDefinition)
		for _, decl := range module.Structs {
			geometryEnv.structs[decl.Name] = decl
		}
		var conceptOffsets []int
		var conceptSize, conceptAlign int
		for _, decl := range module.Structs {
			if decl.Name != claim.TypeName {
				continue
			}
			found = true
			if !decl.Record || !evt1HasCRepr(decl.Attributes) || len(decl.Fields) != len(claim.Fields) {
				return fmt.Errorf("NATIVE_ABI_COMPANION_MISMATCH: %s must be a repr(C) record struct with %d fields", claim.TypeName, len(claim.Fields))
			}
			for i, field := range decl.Fields {
				if field.Name != claim.Fields[i] {
					return fmt.Errorf("NATIVE_ABI_COMPANION_MISMATCH: %s field %d must be %s", claim.TypeName, i, claim.Fields[i])
				}
			}
			conceptOffsets, conceptSize, conceptAlign, err = evt1StructFieldOffsets(geometryEnv, decl)
			if err != nil {
				return err
			}
		}
		if !found {
			return fmt.Errorf("NATIVE_ABI_COMPANION_MISMATCH: %s is absent from %s", claim.TypeName, claim.Companion)
		}
		if conceptSize != claim.Size || conceptAlign != claim.Alignment {
			return fmt.Errorf("NATIVE_ABI_MISMATCH: %s Concept size %d align %d; companion claims size %d align %d", claim.TypeName, conceptSize, conceptAlign, claim.Size, claim.Alignment)
		}
		for i, field := range claim.Fields {
			if conceptOffsets[i] != claim.Offsets[i] {
				return fmt.Errorf("NATIVE_ABI_MISMATCH: %s.%s Concept offset %d; companion claims offset %d", claim.TypeName, field, conceptOffsets[i], claim.Offsets[i])
			}
		}
		// The include and identifiers come from validated project data. Keep probe
		// input narrow: C identifiers and project-relative header paths only.
		if !nativeIdentifier(claim.TypeName) {
			return fmt.Errorf("NATIVE_ABI_CLAIM_INVALID: type %q", claim.TypeName)
		}
		for _, name := range claim.Fields {
			if !nativeIdentifier(name) {
				return fmt.Errorf("NATIVE_ABI_CLAIM_INVALID: field %q", name)
			}
		}
		if strings.ContainsAny(claim.Header, "\"\r\n") {
			return fmt.Errorf("NATIVE_ABI_CLAIM_INVALID: header %q", claim.Header)
		}
		var source strings.Builder
		fmt.Fprintf(&source, "#include \"%s\"\n#include <cstddef>\n#include <cstdio>\nint main() { std::printf(\"%%zu %%zu\", sizeof(%s), alignof(%s));\n", filepath.ToSlash(claim.Header), claim.TypeName, claim.TypeName)
		for _, field := range claim.Fields {
			fmt.Fprintf(&source, "std::printf(\" %%zu\", offsetof(%s, %s));\n", claim.TypeName, field)
		}
		source.WriteString("return 0; }\n")
		temp, err := os.MkdirTemp("", "concept-native-abi-")
		if err != nil {
			return err
		}
		probe := filepath.Join(temp, "probe.cpp")
		executable := filepath.Join(temp, "probe.exe")
		if err := os.WriteFile(probe, []byte(source.String()), 0o644); err != nil {
			os.RemoveAll(temp)
			return err
		}
		probeArgs, err := nativeABIProbeArgs(project, claim, probe, executable)
		if err != nil {
			os.RemoveAll(temp)
			return err
		}
		build := exec.Command(compiler, probeArgs...)
		output, err := build.CombinedOutput()
		if err != nil {
			os.RemoveAll(temp)
			return fmt.Errorf("NATIVE_ABI_PROBE_COMPILE: %s %s: %w\n%s", claim.TypeName, compiler, err, output)
		}
		output, err = exec.Command(executable).CombinedOutput()
		os.RemoveAll(temp)
		if err != nil {
			return fmt.Errorf("NATIVE_ABI_PROBE_RUN: %s: %w\n%s", claim.TypeName, err, output)
		}
		read := strings.NewReader(string(output))
		var size, alignment int
		if _, err := fmt.Fscan(read, &size, &alignment); err != nil {
			return fmt.Errorf("NATIVE_ABI_PROBE_OUTPUT: %s: %w", claim.TypeName, err)
		}
		if size != claim.Size || alignment != claim.Alignment {
			return fmt.Errorf("NATIVE_ABI_MISMATCH: %s Concept size %d align %d; companion claim size %d align %d; %s reports size %d align %d", claim.TypeName, conceptSize, conceptAlign, claim.Size, claim.Alignment, compiler, size, alignment)
		}
		measuredOffsets := make([]int, len(claim.Fields))
		for i, field := range claim.Fields {
			var offset int
			if _, err := fmt.Fscan(read, &offset); err != nil {
				return fmt.Errorf("NATIVE_ABI_PROBE_OUTPUT: %s.%s: %w", claim.TypeName, field, err)
			}
			if offset != claim.Offsets[i] {
				return fmt.Errorf("NATIVE_ABI_MISMATCH: %s.%s Concept offset %d; companion claim offset %d; %s reports offset %d", claim.TypeName, field, conceptOffsets[i], claim.Offsets[i], compiler, offset)
			}
			measuredOffsets[i] = offset
		}
		report.Evidence = append(report.Evidence, NativeABIEvidence{TypeName: claim.TypeName, Origin: "NativeToolchainProbe", Size: size, Alignment: alignment, Fields: append([]string{}, claim.Fields...), Offsets: measuredOffsets})
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(project.Root, ".native-build"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(project.Root, ".native-build", "abi.json"), append(body, '\n'), 0o644)
}

func nativeABIProbeArgs(project NativeProject, claim NativeABIClaim, probe, executable string) ([]string, error) {
	headerDir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(claim.Header)))
	for _, target := range project.Targets {
		if target.Language != "Cpp" {
			continue
		}
		matches := false
		for _, source := range target.Sources {
			if filepath.ToSlash(filepath.Dir(filepath.FromSlash(source))) == headerDir {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		standard := map[string]string{"Cpp17": "c++17", "Cpp20": "c++20", "Cpp23": "c++23"}[target.Standard]
		if standard == "" {
			return nil, fmt.Errorf("NATIVE_ABI_PROBE_FLAGS_UNKNOWN: %s target %s has unsupported C++ standard %s", claim.TypeName, target.Name, target.Standard)
		}
		args := []string{"-std=" + standard, "-I", project.Root}
		for _, include := range target.Includes {
			args = append(args, "-I", filepath.Join(project.Root, filepath.FromSlash(include)))
		}
		for _, define := range target.Defines {
			value := "-D" + define.Name
			if define.HasValue {
				value += "=" + define.Value
			}
			args = append(args, value)
		}
		return append(args, probe, "-o", executable), nil
	}
	return nil, fmt.Errorf("NATIVE_ABI_PROBE_TARGET_UNKNOWN: no C++ target owns header %s", claim.Header)
}

func nativeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
