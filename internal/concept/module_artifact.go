package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const SemanticModuleSchema = "concept-module.v1"

type SemanticModuleDependency struct {
	ModuleIdentity string `json:"module_identity"`
	ContentSHA256  string `json:"content_sha256"`
}

type SemanticModuleEffectSummary struct {
	Operation string `json:"operation"`
	Signature string `json:"signature,omitempty"`
	Effect    string `json:"effect"`
	Origin    string `json:"origin"`
}

type SemanticModuleTypeSummary struct {
	Name     string `json:"name"`
	Copyable bool   `json:"copyable"`
	Movable  bool   `json:"movable"`
	HasDrop  bool   `json:"has_drop"`
	Ref      bool   `json:"ref,omitempty"`
}

type SemanticModuleExports struct {
	Types                []string                    `json:"types,omitempty"`
	Functions            []string                    `json:"functions,omitempty"`
	Concepts             []string                    `json:"concepts,omitempty"`
	Interfaces           []string                    `json:"interfaces,omitempty"`
	GenericTypes         []string                    `json:"generic_types,omitempty"`
	GenericFunctions     []string                    `json:"generic_functions,omitempty"`
	TypeAliases          []string                    `json:"type_aliases,omitempty"`
	NormalizedSignatures []string                    `json:"normalized_signatures,omitempty"`
	TypeSummaries        []SemanticModuleTypeSummary `json:"type_summaries,omitempty"`
	QualifiedSymbols     []string                    `json:"qualified_symbols,omitempty"`
}

// SemanticModuleArtifact is compiler-semantic data. SemanticPayload is a
// compiler-private encoding of the typed AST, not source text or backend code.
// The surrounding JSON stays inspectable and carries compatibility, integrity,
// dependency, export, effect, ownership, and diagnostic-origin summaries.
type SemanticModuleArtifact struct {
	SchemaVersion      string                        `json:"schema_version"`
	CompilerIdentity   string                        `json:"compiler_identity"`
	ModuleIdentity     string                        `json:"module_identity"`
	SourceIdentity     string                        `json:"source_identity"`
	SourceSHA256       string                        `json:"source_sha256"`
	ContentSHA256      string                        `json:"content_sha256"`
	Dependencies       []SemanticModuleDependency    `json:"dependencies,omitempty"`
	Exports            SemanticModuleExports         `json:"exports"`
	OperationEffects   []SemanticModuleEffectSummary `json:"operation_effect_summaries,omitempty"`
	ValueFactSummaries []SemanticFunctionFactSummary `json:"value_fact_summaries,omitempty"`
	SharedAccessFacts  []MIRSemanticFact             `json:"shared_access_facts,omitempty"`
	ForeignContracts   []ForeignContractDecl         `json:"foreign_contracts,omitempty"`
	SemanticPayload    []byte                        `json:"semantic_payload"`
}

var semanticGobOnce sync.Once

func registerSemanticGobTypes() {
	semanticGobOnce.Do(func() {
		values := []any{
			&OperationRequirement{}, &PrerequisiteRequirement{}, &FieldRequirement{}, &CompilerAnalysisRequirement{},
			&IfStmt{}, &Block{}, &VarDecl{}, &EffectsDecl{}, &ActuatorLocalDecl{}, &YieldStmt{}, &PushMachineStmt{},
			&MachineCompleteStmt{}, &TransitionStmt{}, &TransitionMatchStmt{}, &TransitionInferStmt{}, &TransitionDecideStmt{},
			&InstanceDecl{}, &ActuationDecl{}, &AssignStmt{}, &ReturnStmt{}, &AssertStmt{}, &TryStmt{}, &ExprStmt{},
			&StaticAssertStmt{}, &MatchStmt{}, &WhileStmt{}, &ForeachStmt{},
			&AwaitExpr{}, &InferExpr{}, &NameExpr{}, &IntLiteral{}, &FloatLiteral{}, &StringLiteral{}, &BoolLiteral{},
			&FieldExpr{}, &CallExpr{}, &DispatchExpr{}, &TemplateCallExpr{}, &BinaryExpr{}, &UnaryExpr{}, &MoveExpr{},
			&RefExpr{}, &BindExpr{}, &ConstructExpr{}, &StructConstructExpr{}, &CallableExpr{}, &WithExpr{},
			&ArrayLiteralExpr{}, &IndexExpr{}, &MatchExpr{}, &IfExpr{}, &FailureExpr{}, &ParenExpr{},
		}
		for _, value := range values {
			gob.Register(value)
		}
	})
}

func encodeSemanticModule(module Module) ([]byte, error) {
	registerSemanticGobTypes()
	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(module); err != nil {
		return nil, fmt.Errorf("MODULE_ARTIFACT_ENCODE: %w", err)
	}
	return buffer.Bytes(), nil
}

func decodeSemanticModule(payload []byte) (Module, error) {
	registerSemanticGobTypes()
	var module Module
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&module); err != nil {
		return Module{}, fmt.Errorf("MODULE_ARTIFACT_CORRUPT: semantic payload: %w", err)
	}
	return module, nil
}

func semanticArtifactHash(artifact SemanticModuleArtifact) (string, error) {
	artifact.ContentSHA256 = ""
	body, err := json.Marshal(artifact)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func LoadSemanticModuleArtifact(body []byte) (SemanticModuleArtifact, Module, error) {
	var artifact SemanticModuleArtifact
	if err := json.Unmarshal(body, &artifact); err != nil {
		return artifact, Module{}, fmt.Errorf("MODULE_ARTIFACT_CORRUPT: %w", err)
	}
	if artifact.SchemaVersion != SemanticModuleSchema {
		return artifact, Module{}, fmt.Errorf("MODULE_SCHEMA_UNSUPPORTED: expected %s, got %s", SemanticModuleSchema, artifact.SchemaVersion)
	}
	if artifact.CompilerIdentity != CompilerID {
		return artifact, Module{}, fmt.Errorf("MODULE_COMPILER_INCOMPATIBLE: expected %s, got %s", CompilerID, artifact.CompilerIdentity)
	}
	if err := validateSemanticFunctionFactSummaries(artifact.ValueFactSummaries); err != nil {
		return artifact, Module{}, err
	}
	hash, err := semanticArtifactHash(artifact)
	if err != nil {
		return artifact, Module{}, err
	}
	if hash != artifact.ContentSHA256 {
		return artifact, Module{}, fmt.Errorf("MODULE_ARTIFACT_CORRUPT: content hash mismatch")
	}
	module, err := decodeSemanticModule(artifact.SemanticPayload)
	if err != nil {
		return artifact, Module{}, err
	}
	if module.Name != artifact.ModuleIdentity {
		return artifact, Module{}, fmt.Errorf("MODULE_IDENTITY_MISMATCH: payload declares %s, artifact declares %s", module.Name, artifact.ModuleIdentity)
	}
	if !reflect.DeepEqual(module.ForeignContracts, artifact.ForeignContracts) {
		return artifact, Module{}, fmt.Errorf("MODULE_FOREIGN_CONTRACT_MISMATCH: inspectable foreign declarations differ from semantic payload")
	}
	return artifact, module, nil
}

func CompileSemanticModule(path, source string, artifacts map[string][]byte) ([]byte, error) {
	local, err := parseSyntaxModule(path, source)
	if err != nil {
		return nil, err
	}
	if local.Name == "" {
		return nil, evt1Diagnostic("MODULE_DECLARATION_REQUIRED", "semantic module compilation requires `module Name;`", Span{Line: 1, Column: 1})
	}
	composed, loaded, err := composeSemanticModules(local, artifacts)
	if err != nil {
		return nil, err
	}
	if err := resolveNamespaceSymbols(&composed); err != nil {
		return nil, err
	}
	env, err := analyzeModule(composed)
	if err != nil {
		return nil, err
	}
	portable := local
	portable.Path = local.Name
	// Preserve concrete generic field structure referenced by exported
	// signatures. Consumers must be able to validate Result/Option payloads and
	// field fact summaries even when their own source never spells the generic
	// application independently.
	evt1MaterializeGenericInstances(&portable, env)
	payload, err := encodeSemanticModule(portable)
	if err != nil {
		return nil, err
	}
	artifact := SemanticModuleArtifact{
		SchemaVersion:      SemanticModuleSchema,
		CompilerIdentity:   CompilerID,
		ModuleIdentity:     local.Name,
		SourceIdentity:     local.Name,
		SourceSHA256:       digest([]byte(source)),
		Exports:            semanticModuleExports(local, env),
		OperationEffects:   summarizeModuleEffects(local, env),
		ValueFactSummaries: semanticModuleFactSummaries(local, env),
		SharedAccessFacts:  evt1LocalSharedAccessFacts(local, env),
		ForeignContracts:   append([]ForeignContractDecl{}, local.ForeignContracts...),
		SemanticPayload:    payload,
	}
	for _, dependency := range loaded {
		artifact.Dependencies = append(artifact.Dependencies, SemanticModuleDependency{ModuleIdentity: dependency.ModuleIdentity, ContentSHA256: dependency.ContentSHA256})
	}
	sort.Slice(artifact.Dependencies, func(i, j int) bool {
		return artifact.Dependencies[i].ModuleIdentity < artifact.Dependencies[j].ModuleIdentity
	})
	artifact.ContentSHA256, err = semanticArtifactHash(artifact)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(artifact, "", "  ")
}

func ParseWithSemanticModules(path, source string, artifacts map[string][]byte) (Module, error) {
	local, err := parseSyntaxModule(path, source)
	if err != nil {
		return Module{}, err
	}
	module, _, err := composeSemanticModules(local, artifacts)
	if err != nil {
		return Module{}, err
	}
	if err := resolveNamespaceSymbols(&module); err != nil {
		return Module{}, err
	}
	env, err := analyzeModule(module)
	if err != nil {
		return Module{}, err
	}
	evt1MaterializeGenericInstances(&module, env)
	evt1ApplyExactCallableTypes(&module, env)
	return module, nil
}

// SemanticModuleImports reads only the importing unit's syntax. It is used by
// the deterministic filesystem resolver; imported source is never opened.
func SemanticModuleImports(path, source string) ([]string, error) {
	module, err := parseSyntaxModule(path, source)
	if err != nil {
		return nil, err
	}
	imports := append([]string{}, module.Imports...)
	sort.Strings(imports)
	return imports, nil
}

// ResolveSemanticModuleArtifacts resolves semantic names by exact path mapping
// under explicitly supplied roots: A.B -> <root>/A/B.concept-module.json.
// Multiple matches are rejected instead of selecting by root order.
func ResolveSemanticModuleArtifacts(roots []string, imports []string) (map[string][]byte, error) {
	artifacts := map[string][]byte{}
	uniqueRoots := make([]string, 0, len(roots))
	seenRoots := map[string]bool{}
	for _, root := range roots {
		root = filepath.Clean(root)
		if !seenRoots[root] {
			seenRoots[root] = true
			uniqueRoots = append(uniqueRoots, root)
		}
	}
	roots = uniqueRoots
	queued := append([]string{}, imports...)
	sort.Strings(queued)
	for len(queued) != 0 {
		name := queued[0]
		queued = queued[1:]
		if _, ok := artifacts[name]; ok {
			continue
		}
		relative := filepath.FromSlash(strings.ReplaceAll(name, ".", "/") + ".concept-module.json")
		var matches []string
		for _, root := range roots {
			candidate := filepath.Join(root, relative)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				matches = append(matches, candidate)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("MODULE_IMPORT_MISSING: no semantic artifact for %s under configured roots", name)
		}
		if len(matches) > 1 {
			sort.Strings(matches)
			return nil, fmt.Errorf("MODULE_IDENTITY_DUPLICATE: %s resolves to %s", name, strings.Join(matches, ", "))
		}
		body, err := os.ReadFile(matches[0])
		if err != nil {
			return nil, err
		}
		artifact, _, err := LoadSemanticModuleArtifact(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.ToSlash(matches[0]), err)
		}
		if artifact.ModuleIdentity != name {
			return nil, fmt.Errorf("MODULE_IDENTITY_MISMATCH: import %s resolved artifact %s", name, artifact.ModuleIdentity)
		}
		sourcePath := strings.TrimSuffix(matches[0], ".concept-module.json") + ".concept"
		if source, err := os.ReadFile(sourcePath); err == nil && digest(source) != artifact.SourceSHA256 {
			return nil, fmt.Errorf("MODULE_ARTIFACT_STALE: %s changed; rebuild %s", filepath.ToSlash(sourcePath), filepath.ToSlash(matches[0]))
		}
		artifacts[name] = body
		for _, dependency := range artifact.Dependencies {
			queued = append(queued, dependency.ModuleIdentity)
		}
		sort.Strings(queued)
	}
	return artifacts, nil
}

func ParseWithSemanticModuleRoots(path, source string, roots []string) (Module, error) {
	imports, err := SemanticModuleImports(path, source)
	if err != nil {
		return Module{}, err
	}
	artifacts, err := ResolveSemanticModuleArtifacts(roots, imports)
	if err != nil {
		return Module{}, err
	}
	return ParseWithSemanticModules(path, source, artifacts)
}

func CompileSemanticModuleWithRoots(path, source string, roots []string) ([]byte, error) {
	imports, err := SemanticModuleImports(path, source)
	if err != nil {
		return nil, err
	}
	artifacts, err := ResolveSemanticModuleArtifacts(roots, imports)
	if err != nil {
		return nil, err
	}
	return CompileSemanticModule(path, source, artifacts)
}

// BuildSemanticModuleArtifactsFromSources compiles an exact, acyclic source
// graph to in-memory artifacts. It is the R6a test-build path: dependencies are
// built first, and the consuming test still imports artifacts rather than
// textually including or reparsing their source during its own sema pass.
func BuildSemanticModuleArtifactsFromSources(roots []string, imports []string) (map[string][]byte, error) {
	artifacts := map[string][]byte{}
	states := map[string]int{}
	var build func(string) error
	build = func(name string) error {
		if states[name] == 1 {
			return fmt.Errorf("MODULE_IMPORT_CYCLE: source dependency cycle reaches %s", name)
		}
		if states[name] == 2 {
			return nil
		}
		states[name] = 1
		relative := filepath.FromSlash(strings.ReplaceAll(name, ".", "/") + ".concept")
		var matches []string
		seen := map[string]bool{}
		for _, root := range roots {
			candidate := filepath.Clean(filepath.Join(root, relative))
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				matches = append(matches, candidate)
			}
		}
		if len(matches) == 0 {
			return fmt.Errorf("MODULE_IMPORT_MISSING: no module source for %s under test roots", name)
		}
		if len(matches) > 1 {
			sort.Strings(matches)
			return fmt.Errorf("MODULE_IDENTITY_DUPLICATE: %s resolves to %s", name, strings.Join(matches, ", "))
		}
		source, err := os.ReadFile(matches[0])
		if err != nil {
			return err
		}
		module, err := parseSyntaxModule(filepath.ToSlash(matches[0]), string(source))
		if err != nil {
			return err
		}
		if module.Name != name {
			return fmt.Errorf("MODULE_IDENTITY_MISMATCH: path for %s declares %s", name, module.Name)
		}
		dependencies := append([]string{}, module.Imports...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := build(dependency); err != nil {
				return err
			}
		}
		body, err := CompileSemanticModule(filepath.ToSlash(matches[0]), string(source), artifacts)
		if err != nil {
			return err
		}
		artifacts[name] = body
		states[name] = 2
		return nil
	}
	ordered := append([]string{}, imports...)
	sort.Strings(ordered)
	for _, name := range ordered {
		if err := build(name); err != nil {
			return nil, err
		}
	}
	return artifacts, nil
}

func ParseWithBuiltSemanticModuleRoots(path, source string, roots []string) (Module, error) {
	imports, err := SemanticModuleImports(path, source)
	if err != nil {
		return Module{}, err
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources(roots, imports)
	if err != nil {
		return Module{}, err
	}
	return ParseWithSemanticModules(path, source, artifacts)
}

func composeSemanticModules(local Module, artifacts map[string][]byte) (Module, []SemanticModuleArtifact, error) {
	var order []SemanticModuleArtifact
	modules := map[string]Module{}
	states := map[string]int{}
	var load func(string) error
	load = func(name string) error {
		if name == local.Name && local.Name != "" {
			return fmt.Errorf("MODULE_IMPORT_CYCLE: %s imports itself", name)
		}
		switch states[name] {
		case 1:
			return fmt.Errorf("MODULE_IMPORT_CYCLE: dependency cycle reaches %s", name)
		case 2:
			return nil
		}
		body, ok := artifacts[name]
		if !ok {
			return fmt.Errorf("MODULE_IMPORT_MISSING: no semantic artifact for %s", name)
		}
		states[name] = 1
		artifact, module, err := LoadSemanticModuleArtifact(body)
		if err != nil {
			return err
		}
		if artifact.ModuleIdentity != name {
			return fmt.Errorf("MODULE_IDENTITY_MISMATCH: import %s resolved artifact %s", name, artifact.ModuleIdentity)
		}
		for _, dependency := range artifact.Dependencies {
			if err := load(dependency.ModuleIdentity); err != nil {
				return err
			}
			loadedDependency, _, err := LoadSemanticModuleArtifact(artifacts[dependency.ModuleIdentity])
			if err != nil {
				return err
			}
			if loadedDependency.ContentSHA256 != dependency.ContentSHA256 {
				return fmt.Errorf("MODULE_DEPENDENCY_STALE: %s expects %s at %s, got %s", artifact.ModuleIdentity, dependency.ModuleIdentity, dependency.ContentSHA256, loadedDependency.ContentSHA256)
			}
		}
		states[name] = 2
		modules[name] = module
		order = append(order, artifact)
		return nil
	}
	imports := append([]string{}, local.Imports...)
	sort.Strings(imports)
	for index := 1; index < len(imports); index++ {
		if imports[index] == imports[index-1] {
			return Module{}, nil, fmt.Errorf("MODULE_IMPORT_DUPLICATE: %s", imports[index])
		}
	}
	for _, name := range imports {
		if err := load(name); err != nil {
			return Module{}, nil, err
		}
	}
	composed := Module{Path: local.Path, Name: local.Name, Profile: local.Profile}
	for _, artifact := range order {
		dependency := modules[artifact.ModuleIdentity]
		appendSemanticDeclarations(&composed, dependency)
		for _, fn := range dependency.Functions {
			composed.ImportedFactAuthority = append(composed.ImportedFactAuthority, evt1FunctionProvenanceKey(fn))
		}
		for _, template := range dependency.Templates {
			composed.ImportedFactAuthority = append(composed.ImportedFactAuthority, "template:"+template.Name)
		}
		for _, summary := range artifact.OperationEffects {
			origin := string(FactOriginModuleSummaryEffect)
			if summary.Origin == string(FactOriginDeclaredForeign) {
				origin = summary.Origin
			}
			composed.OperationEffects = append(composed.OperationEffects, OperationEffectDecl{Effect: summary.Effect, Operation: summary.Operation, Signature: summary.Signature, Origin: origin, Module: artifact.ModuleIdentity})
		}
		for _, summary := range artifact.ValueFactSummaries {
			if summary.Origin != FactOriginDeclaredForeign {
				summary.Origin = FactOriginModuleFactSummary
			}
			composed.ImportedFactSummaries = append(composed.ImportedFactSummaries, summary)
		}
		for _, fact := range artifact.SharedAccessFacts {
			fact.Origin = FactOriginModuleFactSummary
			fact.Evidence.Authority = artifact.ModuleIdentity
			composed.SharedAccessFacts = append(composed.SharedAccessFacts, fact)
		}
	}
	appendSemanticDeclarations(&composed, local)
	composed.OperationEffects = append(composed.OperationEffects, local.OperationEffects...)
	composed.ImportedFactSummaries = append(composed.ImportedFactSummaries, local.ImportedFactSummaries...)
	composed.ImportedFactAuthority = append(composed.ImportedFactAuthority, local.ImportedFactAuthority...)
	composed.SharedAccessFacts = append(composed.SharedAccessFacts, local.SharedAccessFacts...)
	// Compile-time obligations belong only to the consuming unit. Imported
	// assertions are checked when their source module is built and are not
	// replayed, while local assertions must remain on the ordinary sema path.
	composed.Assertions = append(composed.Assertions, local.Assertions...)
	composed.StaticAsserts = append(composed.StaticAsserts, local.StaticAsserts...)
	return composed, order, nil
}

func appendSemanticDeclarations(target *Module, source Module) {
	target.NamespaceSymbols = append(target.NamespaceSymbols, source.NamespaceSymbols...)
	target.TypeAliases = append(target.TypeAliases, source.TypeAliases...)
	for _, incoming := range source.Structs {
		duplicateInstance := false
		if strings.Contains(incoming.Name, "<") {
			for _, existing := range target.Structs {
				if existing.Name == incoming.Name && reflect.DeepEqual(existing, incoming) {
					duplicateInstance = true
					break
				}
			}
		}
		if !duplicateInstance {
			target.Structs = append(target.Structs, incoming)
		}
	}
	target.Layouts = append(target.Layouts, source.Layouts...)
	target.Streams = append(target.Streams, source.Streams...)
	target.Enums = append(target.Enums, source.Enums...)
	target.Effects = append(target.Effects, source.Effects...)
	target.Actuators = append(target.Actuators, source.Actuators...)
	target.Automata = append(target.Automata, source.Automata...)
	target.Concepts = append(target.Concepts, source.Concepts...)
	target.ComptimeDecls = append(target.ComptimeDecls, source.ComptimeDecls...)
	target.Templates = append(target.Templates, source.Templates...)
	target.GenericTypes = append(target.GenericTypes, source.GenericTypes...)
	target.Functions = append(target.Functions, source.Functions...)
	target.ComptimeFns = append(target.ComptimeFns, source.ComptimeFns...)
	target.ForeignContracts = append(target.ForeignContracts, source.ForeignContracts...)
}

func semanticModuleExports(module Module, env *semanticEnv) SemanticModuleExports {
	var exports SemanticModuleExports
	for _, symbol := range module.NamespaceSymbols {
		exports.QualifiedSymbols = append(exports.QualifiedSymbols, symbol.Namespace+"."+symbol.Name)
	}
	sort.Strings(exports.QualifiedSymbols)
	for _, decl := range module.Structs {
		exports.Types = append(exports.Types, decl.Name)
		t := Type{Name: decl.Name, Kind: TypeStruct}
		exports.TypeSummaries = append(exports.TypeSummaries, SemanticModuleTypeSummary{Name: decl.Name, Copyable: evt1TypeCopyable(env, t), Movable: evt1TypeMovable(env, t), HasDrop: evt1TypeHasDrop(env, t), Ref: decl.Ref})
	}
	for _, decl := range module.Enums {
		exports.Types = append(exports.Types, decl.Name)
	}
	for _, decl := range module.Layouts {
		exports.Types = append(exports.Types, decl.Name)
	}
	for _, decl := range module.TypeAliases {
		exports.TypeAliases = append(exports.TypeAliases, decl.Name)
	}
	for _, decl := range module.GenericTypes {
		exports.GenericTypes = append(exports.GenericTypes, decl.Name)
	}
	for _, decl := range module.Templates {
		exports.GenericFunctions = append(exports.GenericFunctions, decl.Name)
	}
	for _, decl := range module.Functions {
		exports.Functions = append(exports.Functions, decl.Name)
		exports.NormalizedSignatures = append(exports.NormalizedSignatures, evt1FunctionSignature(decl))
	}
	for _, decl := range module.Concepts {
		if decl.Interface {
			exports.Interfaces = append(exports.Interfaces, decl.Name)
		} else {
			exports.Concepts = append(exports.Concepts, decl.Name)
		}
	}
	sort.Strings(exports.Types)
	sort.Strings(exports.Functions)
	sort.Strings(exports.Concepts)
	sort.Strings(exports.Interfaces)
	sort.Strings(exports.GenericTypes)
	sort.Strings(exports.GenericFunctions)
	sort.Strings(exports.TypeAliases)
	sort.Strings(exports.NormalizedSignatures)
	sort.Slice(exports.TypeSummaries, func(i, j int) bool { return exports.TypeSummaries[i].Name < exports.TypeSummaries[j].Name })
	return exports
}

func summarizeModuleEffects(module Module, env *semanticEnv) []SemanticModuleEffectSummary {
	state := map[string]int{}
	cache := map[string]SemanticModuleEffectSummary{}
	var summarize func(FunctionDecl) SemanticModuleEffectSummary
	summarize = func(fn FunctionDecl) SemanticModuleEffectSummary {
		key := evt1OperationEffectKey(fn.Name, evt1FunctionParamSignature(fn))
		if summary, ok := cache[key]; ok {
			return summary
		}
		if state[key] == 1 {
			return SemanticModuleEffectSummary{Operation: fn.Name, Effect: "Unknown", Origin: string(FactOriginCompilerAnalysis)}
		}
		state[key] = 1
		if effect, ok := evt1OperationEffectForFunction(env, fn); ok && effect.Effect == "Allocates" {
			origin := FactOriginDeclaredEffect
			if effect.Origin == string(FactOriginDeclaredForeign) {
				origin = FactOriginDeclaredForeign
			} else if fn.ExternABI != "" {
				origin = FactOriginExternalContractEffect
			}
			summary := SemanticModuleEffectSummary{Operation: fn.Name, Signature: evt1FunctionParamSignature(fn), Effect: "Allocates", Origin: string(origin)}
			cache[key], state[key] = summary, 2
			return summary
		}
		if fn.Body == nil {
			summary := SemanticModuleEffectSummary{Operation: fn.Name, Signature: evt1FunctionParamSignature(fn), Effect: "Unknown", Origin: string(FactOriginCompilerAnalysis)}
			cache[key], state[key] = summary, 2
			return summary
		}
		effect := "NoAllocation"
		origin := string(FactOriginCompilerAnalysis)
		for _, call := range evt1DirectCalls(*fn.Body) {
			candidates := env.functions[call]
			if len(candidates) != 1 {
				effect = "Unknown"
				continue
			}
			child := summarize(candidates[0])
			if child.Effect == "Allocates" {
				effect, origin = "Allocates", string(FactOriginDerivedCallEffect)
				break
			}
			if child.Effect == "Unknown" {
				effect = "Unknown"
			}
		}
		summary := SemanticModuleEffectSummary{Operation: fn.Name, Signature: evt1FunctionParamSignature(fn), Effect: effect, Origin: origin}
		cache[key], state[key] = summary, 2
		return summary
	}
	var summaries []SemanticModuleEffectSummary
	for _, fn := range module.Functions {
		summaries = append(summaries, summarize(fn))
	}
	for _, template := range module.Templates {
		if _, already := cache[evt1OperationEffectKey(template.Name, "template")]; already {
			continue
		}
		if effect, ok := env.operationEffects[evt1OperationEffectKey(template.Name, "template")]; ok {
			summaries = append(summaries, SemanticModuleEffectSummary{
				Operation: template.Name,
				Signature: "template",
				Effect:    effect.Effect,
				Origin:    string(FactOriginDeclaredEffect),
			})
		} else if effect, ok := env.operationEffects[template.Name]; ok {
			summaries = append(summaries, SemanticModuleEffectSummary{
				Operation: template.Name,
				Signature: "template",
				Effect:    effect.Effect,
				Origin:    string(FactOriginDeclaredEffect),
			})
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Operation != summaries[j].Operation {
			return summaries[i].Operation < summaries[j].Operation
		}
		return summaries[i].Signature < summaries[j].Signature
	})
	return summaries
}
