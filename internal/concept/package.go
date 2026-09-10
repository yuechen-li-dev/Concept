package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const PackageGraphSchema = "concept-package-graph.v1"

type PackageVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}
type PackageManifestValue struct {
	Name           string         `json:"name"`
	Author         string         `json:"author"`
	Version        PackageVersion `json:"version"`
	Kind           string         `json:"kind"`
	Dependencies   []string       `json:"dependencies"`
	ManifestPath   string         `json:"manifest_path"`
	ManifestSHA256 string         `json:"manifest_sha256"`
}
type PackageBuild struct {
	Name           string                     `json:"name"`
	Author         string                     `json:"author"`
	Version        PackageVersion             `json:"version"`
	Kind           string                     `json:"kind"`
	Dependencies   []string                   `json:"dependencies"`
	ManifestSHA256 string                     `json:"manifest_sha256"`
	Modules        []SemanticModuleDependency `json:"modules"`
}
type PackageGraph struct {
	Schema        string         `json:"schema"`
	Compiler      string         `json:"compiler"`
	Packages      []PackageBuild `json:"packages"`
	ContentSHA256 string         `json:"content_sha256"`
}

func DiscoverPackageManifests(libraryRoot string) (map[string]PackageManifestValue, error) {
	paths, err := filepath.Glob(filepath.Join(libraryRoot, "*", "manifest.concept"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	manifests := map[string]PackageManifestValue{}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		manifest, err := extractPackageManifest(path, body)
		if err != nil {
			return nil, err
		}
		if _, exists := manifests[manifest.Name]; exists {
			return nil, fmt.Errorf("PACKAGE_IDENTITY_DUPLICATE: %s", manifest.Name)
		}
		manifests[manifest.Name] = manifest
	}
	return manifests, nil
}

func extractPackageManifest(path string, body []byte) (PackageManifestValue, error) {
	module, err := parseSyntaxModule(filepath.ToSlash(path), string(body))
	if err != nil {
		return PackageManifestValue{}, err
	}
	var value Expr
	for _, decl := range module.ComptimeDecls {
		if decl.Name == "Manifest" && decl.Type.Name == "PackageManifest" {
			value = decl.Value
		}
	}
	construct, ok := value.(*StructConstructExpr)
	if !ok || len(construct.Args) != 6 {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: %s must declare one immutable comptime PackageManifest Manifest value", filepath.ToSlash(path))
	}
	name, ok := construct.Args[0].(*StringLiteral)
	if !ok || strings.TrimSpace(name.Value) == "" {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: package name must be a nonblank string")
	}
	author, ok := construct.Args[1].(*StringLiteral)
	if !ok || strings.TrimSpace(author.Value) == "" {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: package author must be a nonblank string")
	}
	versionValue, ok := construct.Args[2].(*StructConstructExpr)
	if !ok || len(versionValue.Args) != 3 {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: version must contain major, minor, and patch")
	}
	major, ok1 := versionValue.Args[0].(*IntLiteral)
	minor, ok2 := versionValue.Args[1].(*IntLiteral)
	patch, ok3 := versionValue.Args[2].(*IntLiteral)
	if !ok1 || !ok2 || !ok3 || major.Value < 0 || minor.Value < 0 || patch.Value < 0 {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: version components must be nonnegative integers")
	}
	kindValue, ok := construct.Args[3].(*ConstructExpr)
	if !ok || kindValue.EnumName != "PackageKind" {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: kind must be a PackageKind value")
	}
	if kindValue.VariantName != "Library" && kindValue.VariantName != "Executable" && kindValue.VariantName != "Tool" {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: unsupported package kind %s", kindValue.VariantName)
	}
	dependenciesValue, ok := construct.Args[4].(*ArrayLiteralExpr)
	if !ok {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: dependencies must be an ordinary array")
	}
	count, ok := construct.Args[5].(*IntLiteral)
	if !ok || count.Value < 0 || count.Value > len(dependenciesValue.Elements) {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: dependency count is out of bounds")
	}
	manifest := PackageManifestValue{Name: name.Value, Author: author.Value, Version: PackageVersion{major.Value, minor.Value, patch.Value}, Kind: kindValue.VariantName, ManifestPath: filepath.ToSlash(path), ManifestSHA256: digest(body)}
	for i := 0; i < count.Value; i++ {
		dependency, ok := dependenciesValue.Elements[i].(*StructConstructExpr)
		if !ok || len(dependency.Args) != 1 {
			return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: dependency %d must be Dependency{name}", i)
		}
		dependencyName, ok := dependency.Args[0].(*StringLiteral)
		if !ok || strings.TrimSpace(dependencyName.Value) == "" {
			return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: dependency %d has no name", i)
		}
		manifest.Dependencies = append(manifest.Dependencies, dependencyName.Value)
	}
	if !sort.StringsAreSorted(manifest.Dependencies) {
		return PackageManifestValue{}, fmt.Errorf("PACKAGE_MANIFEST_INVALID: dependencies must be sorted by package identity")
	}
	return manifest, nil
}

func PackageBuildOrder(manifests map[string]PackageManifestValue, target string) ([]string, error) {
	states := map[string]int{}
	var order []string
	var visit func(string) error
	visit = func(name string) error {
		manifest, ok := manifests[name]
		if !ok {
			return fmt.Errorf("PACKAGE_DEPENDENCY_MISSING: %s", name)
		}
		if states[name] == 1 {
			return fmt.Errorf("PACKAGE_DEPENDENCY_CYCLE: cycle reaches %s", name)
		}
		if states[name] == 2 {
			return nil
		}
		states[name] = 1
		for _, dependency := range manifest.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		states[name] = 2
		order = append(order, name)
		return nil
	}
	if err := visit(target); err != nil {
		return nil, err
	}
	return order, nil
}

func BuildPackage(libraryRoot, outputRoot, target string) (PackageGraph, error) {
	manifests, err := DiscoverPackageManifests(libraryRoot)
	if err != nil {
		return PackageGraph{}, err
	}
	order, err := PackageBuildOrder(manifests, target)
	if err != nil {
		return PackageGraph{}, err
	}
	allArtifacts := map[string][]byte{}
	graph := PackageGraph{Schema: PackageGraphSchema, Compiler: CompilerID}
	for _, name := range order {
		manifest := manifests[name]
		pkg, built, err := buildOnePackage(filepath.Dir(filepath.FromSlash(manifest.ManifestPath)), outputRoot, manifest, allArtifacts)
		if err != nil {
			return PackageGraph{}, err
		}
		for identity, body := range built {
			allArtifacts[identity] = body
		}
		graph.Packages = append(graph.Packages, pkg)
	}
	copy := graph
	copy.ContentSHA256 = ""
	encoded, _ := json.Marshal(copy)
	sum := sha256.Sum256(encoded)
	graph.ContentSHA256 = hex.EncodeToString(sum[:])
	return graph, nil
}

func buildOnePackage(packageRoot, outputRoot string, manifest PackageManifestValue, available map[string][]byte) (PackageBuild, map[string][]byte, error) {
	sources := map[string]struct {
		path   string
		body   []byte
		module Module
	}{}
	err := filepath.WalkDir(packageRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		lower := strings.ToLower(entry.Name())
		if !strings.HasSuffix(lower, ".concept") || lower == "manifest.concept" || strings.HasSuffix(lower, ".concept_test") {
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
		if module.Name == "" {
			return fmt.Errorf("MODULE_DECLARATION_REQUIRED: package source %s", filepath.ToSlash(path))
		}
		if _, exists := sources[module.Name]; exists {
			return fmt.Errorf("MODULE_IDENTITY_DUPLICATE: %s", module.Name)
		}
		sources[module.Name] = struct {
			path   string
			body   []byte
			module Module
		}{path, body, module}
		return nil
	})
	if err != nil {
		return PackageBuild{}, nil, err
	}
	built := map[string][]byte{}
	states := map[string]int{}
	var build func(string) error
	build = func(identity string) error {
		if states[identity] == 1 {
			return fmt.Errorf("MODULE_IMPORT_CYCLE: source dependency cycle reaches %s", identity)
		}
		if states[identity] == 2 {
			return nil
		}
		source, local := sources[identity]
		if !local {
			if _, ok := available[identity]; ok {
				return nil
			}
			return fmt.Errorf("MODULE_IMPORT_MISSING: package %s imports %s", manifest.Name, identity)
		}
		states[identity] = 1
		for _, dependency := range source.module.Imports {
			if err := build(dependency); err != nil {
				return err
			}
		}
		artifacts := map[string][]byte{}
		for name, body := range available {
			artifacts[name] = body
		}
		for name, body := range built {
			artifacts[name] = body
		}
		body, err := CompileSemanticModule(filepath.ToSlash(source.path), string(source.body), artifacts)
		if err != nil {
			return err
		}
		built[identity] = body
		states[identity] = 2
		return nil
	}
	identities := make([]string, 0, len(sources))
	for identity := range sources {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	for _, identity := range identities {
		if err := build(identity); err != nil {
			return PackageBuild{}, nil, err
		}
	}
	pkg := PackageBuild{
		Name:           manifest.Name,
		Author:         manifest.Author,
		Version:        manifest.Version,
		Kind:           manifest.Kind,
		Dependencies:   append([]string{}, manifest.Dependencies...),
		ManifestSHA256: manifest.ManifestSHA256,
	}
	for _, identity := range identities {
		artifact, _, err := LoadSemanticModuleArtifact(built[identity])
		if err != nil {
			return PackageBuild{}, nil, err
		}
		pkg.Modules = append(pkg.Modules, SemanticModuleDependency{ModuleIdentity: identity, ContentSHA256: artifact.ContentSHA256})
		path := filepath.Join(outputRoot, manifest.Name, "modules", filepath.FromSlash(strings.ReplaceAll(identity, ".", "/")+".concept-module.json"))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return PackageBuild{}, nil, err
		}
		if err := os.WriteFile(path, append(built[identity], '\n'), 0o644); err != nil {
			return PackageBuild{}, nil, err
		}
	}
	manifestBody, err := os.ReadFile(filepath.Join(packageRoot, "manifest.concept"))
	if err != nil {
		return PackageBuild{}, nil, err
	}
	manifestArtifacts := map[string][]byte{}
	for name, body := range available {
		manifestArtifacts[name] = body
	}
	for name, body := range built {
		manifestArtifacts[name] = body
	}
	if _, err := ParseWithSemanticModules(filepath.ToSlash(filepath.Join(packageRoot, "manifest.concept")), string(manifestBody), manifestArtifacts); err != nil {
		return PackageBuild{}, nil, fmt.Errorf("PACKAGE_MANIFEST_SEMANTIC_INVALID: %s: %w", manifest.Name, err)
	}
	packageBody, _ := json.MarshalIndent(pkg, "", "  ")
	packagePath := filepath.Join(outputRoot, manifest.Name, "package.json")
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		return PackageBuild{}, nil, err
	}
	if err := os.WriteFile(packagePath, append(packageBody, '\n'), 0o644); err != nil {
		return PackageBuild{}, nil, err
	}
	return pkg, built, nil
}

func MarshalPackageGraph(graph PackageGraph) ([]byte, error) {
	return json.MarshalIndent(graph, "", "  ")
}
