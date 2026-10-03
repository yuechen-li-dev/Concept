package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func describeNativeABIIdentity(identity NativeABIIdentity) string {
	return fmt.Sprintf("%s %s, %s/%s/%s, input %s, companion %s", identity.CompilerFamily,
		identity.CompilerVersion, identity.TargetTriple, identity.OperatingSystem, identity.Architecture,
		identity.BuildInputHash, identity.SemanticCompanionHash)
}

// EnsureNativeABIReport reuses a report only when the complete current native
// identity matches. Otherwise the ordinary native compiler probe is rerun.
func EnsureNativeABIReport(project NativeProject) (NativeABIReport, error) {
	identity, err := CurrentNativeABIIdentity(project)
	if err != nil {
		return NativeABIReport{}, err
	}
	path := filepath.Join(project.Root, ".native-build", "abi.json")
	if body, err := os.ReadFile(path); err == nil {
		var report NativeABIReport
		if json.Unmarshal(body, &report) == nil && report.Schema == "concept-native-abi.v1" && report.Identity == identity && nativeABIReportComplete(report) {
			return report, nil
		}
	}
	if err := CheckNativeABI(project); err != nil {
		return NativeABIReport{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return NativeABIReport{}, err
	}
	var report NativeABIReport
	if err := json.Unmarshal(body, &report); err != nil {
		return NativeABIReport{}, err
	}
	if report.Identity != identity || !nativeABIReportComplete(report) {
		return NativeABIReport{}, fmt.Errorf("NATIVE_ABI_EVIDENCE_STALE: reprobe produced an incomplete or different native identity")
	}
	return report, nil
}

func nativeABIReportComplete(report NativeABIReport) bool {
	if report.Identity.CompilerFamily == "" || report.Identity.CompilerVersion == "" || report.Identity.TargetTriple == "" ||
		report.Identity.OperatingSystem == "" || report.Identity.Architecture == "" || report.Identity.BuildInputHash == "" ||
		report.Identity.SemanticCompanionHash == "" {
		return false
	}
	compiler := "clang++"
	if report.Identity.CompilerFamily == "GCC" {
		compiler = "g++"
	}
	if (report.Identity.CompilerFamily != "Clang" && report.Identity.CompilerFamily != "GCC") ||
		report.Compiler != compiler || report.CompilerVersion != report.Identity.CompilerVersion ||
		report.Target != report.Identity.TargetTriple+"/"+report.Identity.OperatingSystem+"/"+report.Identity.Architecture ||
		report.BuildInputHash != report.Identity.BuildInputHash || report.SemanticCompanionHash != report.Identity.SemanticCompanionHash {
		return false
	}
	for _, evidence := range report.Evidence {
		if evidence.Identity != report.Identity || evidence.Origin != "NativeToolchainProbe" || evidence.Companion == "" || evidence.SourceSHA256 == "" ||
			evidence.TypeName == "" || evidence.Size <= 0 || evidence.Alignment <= 0 || len(evidence.Fields) != len(evidence.Offsets) {
			return false
		}
	}
	return true
}

func validateNativeABIEvidenceForModule(report NativeABIReport, module Module, sourceSHA string) error {
	if report.Schema != "concept-native-abi.v1" || !nativeABIReportComplete(report) {
		return fmt.Errorf("NATIVE_ABI_EVIDENCE_INVALID: incomplete native measurement")
	}
	env := newSemanticEnv(&coreProfileDefinition)
	for _, decl := range module.Structs {
		env.structs[decl.Name] = decl
	}
	for _, decl := range module.Handles {
		env.handles[decl.Name] = decl
	}
	matched := 0
	for _, decl := range module.Structs {
		if !evt1HasCRepr(decl.Attributes) {
			continue
		}
		var found *NativeABIEvidence
		for i := range report.Evidence {
			evidence := &report.Evidence[i]
			if evidence.TypeName == decl.Name && evidence.SourceSHA256 == sourceSHA {
				if found != nil {
					return fmt.Errorf("NATIVE_ABI_EVIDENCE_INVALID: duplicate measurement for %s", decl.Name)
				}
				found = evidence
			}
		}
		if found == nil {
			return fmt.Errorf("NATIVE_ABI_EVIDENCE_MISSING: %s has no measured evidence for source %s", decl.Name, sourceSHA)
		}
		offsets, size, align, err := evt1StructFieldOffsets(env, decl)
		if err != nil {
			return err
		}
		fields := make([]string, len(decl.Fields))
		for i, field := range decl.Fields {
			fields[i] = field.Name
		}
		if found.Size != size || found.Alignment != align || !reflect.DeepEqual(found.Fields, fields) || !reflect.DeepEqual(found.Offsets, offsets) {
			return fmt.Errorf("NATIVE_ABI_EVIDENCE_CONTRADICTION: %s native probe size %d align %d offsets %v; Concept declaration size %d align %d offsets %v", decl.Name, found.Size, found.Alignment, found.Offsets, size, align, offsets)
		}
		matched++
	}
	if matched == 0 {
		return fmt.Errorf("NATIVE_ABI_EVIDENCE_INVALID: report has no repr(C) declaration in %s", module.Name)
	}
	return nil
}

// CompileNativeSemanticModule requires the exact companion bytes measured by
// the native project. The resulting concept-module.v1 artifact hash covers the
// report, including its target, toolchain, inputs, origin, and measured facts.
func CompileNativeSemanticModule(project NativeProject, companion string, artifacts map[string][]byte) ([]byte, error) {
	declared := false
	for _, path := range project.Companions {
		if path == companion {
			declared = true
			break
		}
	}
	if !declared {
		return nil, fmt.Errorf("NATIVE_ABI_COMPANION_UNKNOWN: %s", companion)
	}
	source, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(companion)))
	if err != nil {
		return nil, err
	}
	report, err := EnsureNativeABIReport(project)
	if err != nil {
		return nil, err
	}
	return CompileSemanticModuleWithNativeABI(filepath.ToSlash(companion), string(source), artifacts, report.Identity, &report)
}

// BuildNativeCompanionArtifacts materializes only declared companion sources.
// It uses ordinary semantic imports and the one measured report for this
// native project; downstream consumers need only the returned artifacts.
func BuildNativeCompanionArtifacts(project NativeProject) (map[string][]byte, NativeABIIdentity, error) {
	report, err := EnsureNativeABIReport(project)
	if err != nil {
		return nil, NativeABIIdentity{}, err
	}
	type sourceUnit struct {
		path, source string
		imports      []string
	}
	units := map[string]sourceUnit{}
	for _, path := range project.Companions {
		body, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(path)))
		if err != nil {
			return nil, NativeABIIdentity{}, err
		}
		module, err := parseSyntaxModule(filepath.ToSlash(path), string(body))
		if err != nil {
			return nil, NativeABIIdentity{}, err
		}
		if module.Name == "" {
			return nil, NativeABIIdentity{}, fmt.Errorf("NATIVE_COMPANION_MODULE_REQUIRED: %s", path)
		}
		if _, exists := units[module.Name]; exists {
			return nil, NativeABIIdentity{}, fmt.Errorf("NATIVE_COMPANION_MODULE_DUPLICATE: %s", module.Name)
		}
		units[module.Name] = sourceUnit{path, string(body), module.Imports}
	}
	artifacts := map[string][]byte{}
	// Companions may import ordinary semantic libraries. Resolve those through
	// the same checked module builder used by source consumers, while keeping
	// native ABI evidence restricted to the explicitly measured companions.
	var libraryImports []string
	for _, unit := range units {
		for _, dependency := range unit.imports {
			if _, companion := units[dependency]; !companion {
				libraryImports = append(libraryImports, dependency)
			}
		}
	}
	if len(libraryImports) != 0 {
		sort.Strings(libraryImports)
		libraries, err := BuildSemanticModuleArtifactsFromSources(evt1TestModuleRoots(project.Root, project.Root), libraryImports)
		if err != nil {
			return nil, NativeABIIdentity{}, err
		}
		for name, artifact := range libraries {
			artifacts[name] = artifact
		}
	}
	active := map[string]bool{}
	var build func(string) error
	build = func(name string) error {
		if _, done := artifacts[name]; done {
			return nil
		}
		if active[name] {
			return fmt.Errorf("MODULE_IMPORT_CYCLE: native companion %s", name)
		}
		unit, exists := units[name]
		if !exists {
			return fmt.Errorf("MODULE_IMPORT_MISSING: native companion %s", name)
		}
		active[name] = true
		imports := append([]string{}, unit.imports...)
		sort.Strings(imports)
		for _, dependency := range imports {
			if err := build(dependency); err != nil {
				return err
			}
		}
		var localReport *NativeABIReport
		for _, evidence := range report.Evidence {
			if evidence.SourceSHA256 == digest([]byte(unit.source)) {
				localReport = &report
				break
			}
		}
		artifact, err := CompileSemanticModuleWithNativeABI(unit.path, unit.source, artifacts, report.Identity, localReport)
		if err != nil {
			return err
		}
		artifacts[name] = artifact
		delete(active, name)
		return nil
	}
	names := make([]string, 0, len(units))
	for name := range units {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := build(name); err != nil {
			return nil, NativeABIIdentity{}, err
		}
	}
	return artifacts, report.Identity, nil
}
