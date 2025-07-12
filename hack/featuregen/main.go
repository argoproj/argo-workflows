package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	featuresDir  = ".features"
	docsOutput   = "docs/new-features.md"
	templateFile = ".features/TEMPLATE.md"
	pendingDir   = ".features/pending"
)

var (
	rootCmd = &cobra.Command{
		Use:   "featuregen",
		Short: "Feature documentation management tool for Argo Workflows",
		Long:  "A tool for managing feature documentation in Argo Workflows.\nProvides functionality to create, validate, preview, and update feature documentation.",
	}

	newCmd = &cobra.Command{
		Use:   "new",
		Short: "Create a new feature document",
		RunE: func(cmd *cobra.Command, args []string) error {
			filename, _ := cmd.Flags().GetString("filename")
			return newFeature(filename)
		},
	}

	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate all feature documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateFeatures()
		},
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update the feature documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			dry, _ := cmd.Flags().GetBool("dry")
			version, _ := cmd.Flags().GetString("version")
			final, _ := cmd.Flags().GetBool("final")
			return updateFeatures(dry, version, final)
		},
	}
)

func init() {
	rootCmd.AddCommand(newCmd, validateCmd, updateCmd)
	newCmd.Flags().String("filename", "", "Specify the filename for the new feature")
	updateCmd.Flags().Bool("dry", false, "Preview changes without applying them")
	updateCmd.Flags().String("version", "", "Specify the version for the update")
	updateCmd.Flags().Bool("final", false, "Move features from pending to released")
}

func ensureDirs() error {
	for _, dir := range []string{featuresDir, pendingDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

func getGitBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	if output, err := cmd.Output(); err == nil {
		if branch := strings.TrimSpace(string(output)); branch != "" {
			return branch
		}
	}
	return "new-feature"
}

func newFeature(filename string) error {
	if err := ensureDirs(); err != nil {
		return err
	}

	if filename == "" {
		filename = getGitBranch()
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	targetPath := filepath.Join(pendingDir, filename)
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("file %s already exists", targetPath)
	}

	if err := copyFile(templateFile, targetPath); err != nil {
		return fmt.Errorf("failed to create feature document: %v", err)
	}

	fmt.Printf("Created new feature document at %s\n", targetPath)
	fmt.Println("Please edit this file to describe your feature")
	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func loadFeatureFile(filePath string) (bool, feature, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, feature{}, err
	}
	return parseContent(filePath, string(content))
}

func getFeatureFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var featureFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			featureFiles = append(featureFiles, file.Name())
		}
	}
	return featureFiles, nil
}

func loadPendingFeatures() (bool, []feature, error) {
	featureFiles, err := getFeatureFiles(pendingDir)
	if err != nil {
		return false, nil, err
	}

	if len(featureFiles) == 0 {
		fmt.Println("No pending features to load")
		return true, nil, nil
	}

	allValid := true
	var featuresData []feature

	for _, file := range featureFiles {
		filePath := filepath.Join(pendingDir, file)
		isValid, featureData, err := loadFeatureFile(filePath)
		if err != nil {
			return false, nil, err
		}

		if !isValid {
			allValid = false
			fmt.Printf("Invalid feature document: %s\n", filePath)
		} else {
			featuresData = append(featuresData, featureData)
		}
	}

	if allValid {
		fmt.Printf("All %d feature documents are valid\n", len(featureFiles))
	}

	return allValid, featuresData, nil
}

func validateFeatures() error {
	allValid, _, err := loadPendingFeatures()
	if err != nil {
		return err
	}
	if !allValid {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func moveFeaturesToReleasedDir(version string, featureFiles []string) error {
	releasedDir := filepath.Join(featuresDir, "released", version)
	if err := os.MkdirAll(releasedDir, 0755); err != nil {
		return err
	}

	for _, file := range featureFiles {
		source := filepath.Join(pendingDir, file)
		target := filepath.Join(releasedDir, file)
		if err := copyFile(source, target); err != nil {
			return err
		}
		if err := os.Remove(source); err != nil {
			return err
		}
	}

	fmt.Printf("Updated features documentation with version %s\n", version)
	fmt.Printf("Moved %d feature files to %s\n", len(featureFiles), releasedDir)
	return nil
}

func updateFeatures(dryRun bool, version string, final bool) error {
	allValid, features, err := loadPendingFeatures()
	if err != nil {
		return err
	}

	if !allValid {
		return fmt.Errorf("validation failed, not updating features")
	}

	if len(features) == 0 {
		return nil
	}

	outputContent := format(version, features)

	fmt.Printf("Preview of changes with %d features:\n", len(features))
	fmt.Println("===================")
	fmt.Println(outputContent)

	if !dryRun {
		if err := os.MkdirAll(filepath.Dir(docsOutput), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(docsOutput, []byte(outputContent), 0644); err != nil {
			return err
		}

		if final && version != "" {
			featureFiles, err := getFeatureFiles(pendingDir)
			if err != nil {
				return err
			}

			if err := moveFeaturesToReleasedDir(version, featureFiles); err != nil {
				return err
			}
		} else {
			versionStr := ""
			if version != "" {
				versionStr = fmt.Sprintf(" with version %s", version)
			}
			fmt.Printf("Updated features documentation%s\n", versionStr)
			if !final {
				fmt.Println("Features remain in pending directory (--final not specified)")
			}
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
