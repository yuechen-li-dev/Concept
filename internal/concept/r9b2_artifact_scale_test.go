package concept

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestR9b2LibraryArtifactScale(t *testing.T) {
	root := os.Getenv("CONCEPT_R9B2_SCALE_ROOT")
	if root == "" {
		root = filepath.Join("..", "..", "libraries")
	}
	var fixtures []innateScaleFixture
	var imports []string
	for _, group := range []string{"Standard", "DragonGod", "Golden", "Vulkan"} {
		err := filepath.WalkDir(filepath.Join(root, group), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".concept" || entry.Name() == "manifest.concept" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			module, err := parseSyntaxModule(filepath.ToSlash(path), string(source))
			if err != nil {
				return err
			}
			imports = append(imports, module.Imports...)
			fixtures = append(fixtures, innateScaleFixture{filepath.ToSlash(path), string(source), group})
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
	for _, group := range []string{"Standard", "DragonGod", "Golden", "Vulkan"} {
		start := time.Now()
		total, peak, count := 0, 0, 0
		for _, fixture := range fixtures {
			if fixture.group != group {
				continue
			}
			artifact, err := CompileSemanticModule(fixture.path, fixture.source, artifacts)
			if err != nil {
				t.Fatalf("%s: %v", fixture.path, err)
			}
			total += len(artifact)
			peak = max(peak, len(artifact))
			count++
		}
		t.Logf("%s modules=%d artifact_bytes=%d largest_artifact_bytes=%d compile_wall=%s", group, count, total, peak, time.Since(start))
	}
}
