package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

const vulkanBindUsage = `Usage:
  concept vulkan-bind <kernel.spv> [-o <Module.concept>] [--name <Name>] [--module <Module>]
  concept vulkan-bind <kernel.spv> -o <Module.concept> --check
  concept vulkan-bind <kernel.spv> --describe

Reflects a SPIR-V compute kernel (from GLSL, HLSL, or SDSL-V) and writes a
profile Vulkan module: a loader, a push-constant layout at the offsets the
shader declares, and Record<Name>, whose buffer access comes from the shader
so the Recording can derive barriers. --check exits 1 when the module was
generated from a different interface.
`

func runVulkanBindCommand(args []string) {
	var kernel, output, name, module string
	check, describe := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--name", "--module":
			if i+1 >= len(args) {
				fmt.Fprint(os.Stderr, vulkanBindUsage)
				os.Exit(2)
			}
			switch args[i] {
			case "-o":
				output = args[i+1]
			case "--name":
				name = args[i+1]
			case "--module":
				module = args[i+1]
			}
			i++
		case "--check":
			check = true
		case "--describe":
			describe = true
		case "--help", "-h":
			fmt.Print(vulkanBindUsage)
			return
		default:
			if kernel != "" || strings.HasPrefix(args[i], "-") {
				fmt.Fprint(os.Stderr, vulkanBindUsage)
				os.Exit(2)
			}
			kernel = args[i]
		}
	}
	if kernel == "" || check && output == "" {
		fmt.Fprint(os.Stderr, vulkanBindUsage)
		os.Exit(2)
	}
	module_, err := os.ReadFile(kernel)
	if err != nil {
		fail(err)
	}
	reflected, err := concept.ReflectSPIRV(module_)
	if err != nil {
		fail(fmt.Errorf("%s: %w", kernel, err))
	}
	if describe {
		fmt.Printf("%s\nfingerprint %s\n", strings.TrimRight(reflected.Describe(), "\n"), reflected.Fingerprint())
		return
	}
	if check {
		existing, err := os.ReadFile(output)
		if err != nil {
			fail(err)
		}
		want := concept.VulkanBindFingerprintLine + reflected.Fingerprint()
		if !strings.Contains(strings.ReplaceAll(string(existing), "\r\n", "\n"), want+"\n") {
			fmt.Fprintf(os.Stderr, "%s is stale: %s changed its interface; rerun concept vulkan-bind %s -o %s\n", output, kernel, kernel, output)
			os.Exit(1)
		}
		fmt.Printf("%s: up to date with %s\n", output, kernel)
		return
	}
	if name == "" {
		name = concept.VulkanBindName(kernel)
	}
	if module == "" {
		module = name + "Kernel"
	}
	kernelPath := filepath.ToSlash(kernel)
	if output != "" {
		if relative, err := filepath.Rel(filepath.Dir(output), kernel); err == nil {
			kernelPath = filepath.ToSlash(relative)
		}
	}
	text, err := concept.GenerateVulkanBinding(reflected, concept.VulkanBindOptions{Name: name, Module: module, KernelPath: kernelPath, Source: kernelPath})
	if err != nil {
		fail(fmt.Errorf("%s: %w", kernel, err))
	}
	if output == "" {
		fmt.Print(text)
		return
	}
	if err := os.WriteFile(output, []byte(text), 0644); err != nil {
		fail(err)
	}
	fmt.Println(filepath.ToSlash(output))
}
