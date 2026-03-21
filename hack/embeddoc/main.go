// embeddoc is a tool that extracts code snippets from source files and embeds them into documentation.
//
// Source files use markers like:
//
//	// <embed id="snippet-name">
//	code here...
//	// </embed>
//
// Documentation files use markers like:
//
//	<!-- <embed id="snippet-name" inject_from="code"> -->
//	(content will be replaced)
//	<!-- </embed> -->
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the embeddoc configuration file structure.
type Config struct {
	Includes []string `yaml:"includes"`
}

// Snippet represents an extracted code snippet.
type Snippet struct {
	Content string
	Source  string
}

var (
	// Source file markers (Go comments)
	sourceStartRe = regexp.MustCompile(`^\s*//\s*<embed\s+id="([^"]+)">\s*$`)
	sourceEndRe   = regexp.MustCompile(`^\s*//\s*</embed>\s*$`)

	// Documentation markers (HTML comments)
	docStartRe = regexp.MustCompile(`<!--\s*<embed\s+id="([^"]+)"\s+inject_from="code">\s*-->`)
	docEndRe   = regexp.MustCompile(`<!--\s*</embed>\s*-->`)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := loadConfig("embeddoc.yaml")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	snippets := make(map[string]*Snippet)
	var mdFiles []string

	for _, include := range config.Includes {
		err := filepath.WalkDir(include, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			switch {
			case strings.HasSuffix(path, ".go"):
				return extractSnippetsFromFile(path, snippets)
			case strings.HasSuffix(path, ".md"):
				mdFiles = append(mdFiles, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walking %s: %w", include, err)
		}
	}

	if len(snippets) == 0 {
		fmt.Println("No snippets found")
		return nil
	}

	fmt.Printf("Found %d snippet(s):\n", len(snippets))
	for id, s := range snippets {
		fmt.Printf("  - %s (from %s)\n", id, s.Source)
	}

	// Inject snippets into collected markdown files
	for _, path := range mdFiles {
		err := injectSnippetsInFile(path, snippets)
		if err != nil {
			return fmt.Errorf("injecting snippets in %s: %w", path, err)
		}
	}

	return nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func extractSnippetsFromFile(path string, snippets map[string]*Snippet) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	var currentContent strings.Builder
	inSnippet := false
	currentID := ""

	for _, line := range lines {
		if !inSnippet {
			if matches := sourceStartRe.FindStringSubmatch(line); matches != nil {
				currentID = matches[1]
				inSnippet = true
				currentContent.Reset()
			}
		} else if sourceEndRe.MatchString(line) {
			content := strings.TrimSuffix(currentContent.String(), "\n")

			if existing, ok := snippets[currentID]; ok {
				return fmt.Errorf("duplicate snippet ID %q: found in %s and %s", currentID, existing.Source, path)
			}

			snippets[currentID] = &Snippet{
				Content: content,
				Source:  path,
			}
			inSnippet = false
			currentID = ""
		} else {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	if inSnippet {
		return fmt.Errorf("unclosed snippet %q in %s", currentID, path)
	}

	return nil
}

func injectSnippetsInFile(path string, snippets map[string]*Snippet) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var result strings.Builder
	inBlock := false
	currentID := ""

	for _, line := range strings.Split(string(data), "\n") {
		if !inBlock {
			result.WriteString(line)
			result.WriteString("\n")

			if matches := docStartRe.FindStringSubmatch(line); matches != nil {
				currentID = matches[1]
				inBlock = true

				snippet, ok := snippets[currentID]
				if !ok {
					return fmt.Errorf("snippet %q not found (referenced in %s)", currentID, path)
				}

				result.WriteString(snippet.Content)
				result.WriteString("\n")
			}
		} else if docEndRe.MatchString(line) {
			// Skip existing content between markers when inBlock is true
			result.WriteString(line)
			result.WriteString("\n")
			inBlock = false
			currentID = ""
		}
	}

	if inBlock {
		return fmt.Errorf("unclosed embed block %q in %s", currentID, path)
	}

	output := strings.TrimSuffix(result.String(), "\n")

	if output != string(data) {
		if err := os.WriteFile(path, []byte(output), 0644); err != nil {
			return err
		}
		fmt.Printf("Updated %s\n", path)
	}

	return nil
}
