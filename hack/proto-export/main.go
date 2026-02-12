package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	outDir := flag.String("out", "dist", "Output directory for exported protos")
	flag.Parse()

	// Read config
	data, err := os.ReadFile("argo-proto.yaml")
	if err != nil {
		fmt.Printf("Error reading argo-proto.yaml: %v\n", err)
		os.Exit(1)
	}

	var config ArgoProto
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing argo-proto.yaml: %v\n", err)
		os.Exit(1)
	}

	// Pass 1: Bundling
	fmt.Println("Starting Bundling Pass...")
	if err := bundleDependencies(config); err != nil {
		fmt.Printf("Error during bundling: %v\n", err)
		os.Exit(1)
	}

	// Pass 2: Exporting
	fmt.Println("Starting Exporting Pass...")
	if err := exportDependencies(config, *outDir); err != nil {
		fmt.Printf("Error during exporting: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done.")
}

func bundleDependencies(config ArgoProto) error {
	for key, dep := range config.Dependencies {
		if len(dep.Bundle) == 0 {
			continue
		}

		fmt.Printf("Processing bundle for %s...\n", key)

		// Create directories: .{{key}}-vendor/{{key}}
		vendorDir := fmt.Sprintf(".%s-vendor", key)
		moduleDir := filepath.Join(vendorDir, key)
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", moduleDir, err)
		}

		// Generate buf.yaml in the vendorDir (one level up from moduleDir)
		bufConfig := "version: v1\n"
		bufConfigFile := filepath.Join(vendorDir, "buf.yaml")
		if err := os.WriteFile(bufConfigFile, []byte(bufConfig), 0644); err != nil {
			return fmt.Errorf("failed to write buf.yaml: %w", err)
		}

		// Clone repositories
		for name, bundleCfg := range dep.Bundle {
			targetDir := filepath.Join(moduleDir, name)

			// Check if already exists and has content
			if entries, err := os.ReadDir(targetDir); err == nil && len(entries) > 0 {
				fmt.Printf("  %s already exists and is not empty, skipping clone.\n", targetDir)
				continue
			}

			repoURL := fmt.Sprintf("https://github.com/%s/%s", bundleCfg.Owner, bundleCfg.Name)
			fmt.Printf("  Fetching %s (%s) to %s...\n", repoURL, bundleCfg.Ref, targetDir)

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
			}
			// Sequence for robust shallow fetch of a specific ref (branch, tag, or SHA)
			steps := [][]string{
				{"git", "init"},
				{"git", "remote", "add", "origin", repoURL},
				{"git", "fetch", "--depth", "1", "origin", bundleCfg.Ref},
				{"git", "checkout", "FETCH_HEAD"},
			}

			for _, args := range steps {
				cmd := exec.Command(args[0], args[1:]...)
				cmd.Dir = targetDir
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("git command %v failed in %s: %s: %w", args, targetDir, string(output), err)
				}
			}
		}
	}
	return nil
}

func exportDependencies(config ArgoProto, outDir string) error {
	// Ensure output directory exists

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	for key, dep := range config.Dependencies {
		var source string
		target := outDir

		// If the key contains a slash, treat it as a sub-path within the output directory
		if filepath.Separator == '/' && (filepath.Base(key) != key) {
			target = filepath.Join(outDir, key)
		} else if filepath.Separator != '/' && (filepath.ToSlash(key) != key) {
			// Handle windows if necessary, though key is usually forward-slash
			target = filepath.Join(outDir, key)
		}

		if dep.Buf != nil {
			source = fmt.Sprintf("buf.build/%s/%s:%s", dep.Owner, dep.Name, dep.Buf.Ref)
			fmt.Printf("Exporting %s (Buf source: %s to %s)...\n", key, source, target)
		} else if len(dep.Bundle) > 0 {
			source = fmt.Sprintf(".%s-vendor", key)
			target = outDir // Bundles already have the correct structure in their vendorDir
			fmt.Printf("Exporting %s (Local path: %s to %s)...\n", key, source, target)
		} else {
			continue
		}

		// Ensure target sub-directory exists
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", target, err)
		}

		// buf export <source> --output <target> --exclude-imports
		cmd := exec.Command("buf", "export", source, "--output", target, "--exclude-imports")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to export %s: %w", key, err)
		}
	}
	return nil
}
