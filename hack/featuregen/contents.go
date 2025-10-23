package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// timeNow is a variable that can be replaced in tests to mock time.Now()
var timeNow = time.Now

type feature struct {
	Component   string
	Description string
	Author      string
	Issues      []string
	Details     string
}

// Metadata field definitions
var (
	metadataFields = []string{"Component", "Issues", "Description", "Authors", "Author"}
)

func getMetadataPattern(field string) *regexp.Regexp {
	return regexp.MustCompile(field + `:\s*(.*?)(?:\n|$)`)
}

func validateHeaders(content string) (bool, string) {
	re := regexp.MustCompile(`(?m)^(#+)\s+(.+)$`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) != 3 {
			continue
		}
		level := len(match[1])

		// Require level 3 or higher for all headers
		if level < 3 {
			return false, fmt.Sprintf("Header '%s' must be at least level 3 (###)", match[0])
		}
	}
	return true, ""
}

func validateMetadataOrder(content string) (bool, string) {
	lines := strings.Split(content, "\n")

	// Find the first line that's not a metadata field
	metadataEnd := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		isMetadata := false
		for _, field := range metadataFields {
			if strings.HasPrefix(line, field+":") {
				isMetadata = true
				break
			}
		}
		if !isMetadata && line != "" {
			metadataEnd = i
			break
		}
	}

	// Check if any metadata fields appear after this point
	for i := metadataEnd; i < len(lines); i++ {
		line := lines[i]
		for _, field := range metadataFields {
			if strings.HasPrefix(line, field+":") {
				return false, fmt.Sprintf("Metadata field '%s' must appear before any other content", line)
			}
		}
	}

	return true, ""
}

func parseContent(source string, content string) (bool, feature, error) {
	// Check required sections
	isValid := true
	for _, field := range metadataFields {
		switch field {
		case "Authors":
			if !strings.Contains(content, "Authors:") && !strings.Contains(content, "Author:") {
				fmt.Printf("Error: Missing required section 'Authors:' (or 'Author:') in %s\n", source)
				isValid = false
			}
		case "Author":
			// Skip - handled by "Authors" check above
		default:
			if !strings.Contains(content, field+":") {
				fmt.Printf("Error: Missing required section '%s:' in %s\n", field, source)
				isValid = false
			}
		}
	}

	if headerValid, errMsg := validateHeaders(content); !headerValid {
		fmt.Printf("Error: %s in %s\n", errMsg, source)
		isValid = false
	}

	if orderValid, errMsg := validateMetadataOrder(content); !orderValid {
		fmt.Printf("Error: %s in %s\n", errMsg, source)
		isValid = false
	}

	// Extract metadata fields
	component := ""
	if matches := getMetadataPattern("Component").FindStringSubmatch(content); len(matches) > 1 {
		component = strings.TrimSpace(matches[1])
		if !isValidComponent(component) {
			fmt.Printf("Error: Invalid component '%s' in %s. Valid components are: %s. Add more in hack/featuregen/components.go\n", component, source, listValidComponents())
			isValid = false
		}
	}

	issuesSection := ""
	if matches := getMetadataPattern("Issues").FindStringSubmatch(content); len(matches) > 1 {
		issuesSection = matches[1]
	}
	issues := regexp.MustCompile(`(\d+)`).FindAllStringSubmatch(issuesSection, -1)
	issueNumbers := make([]string, len(issues))
	for i, issue := range issues {
		issueNumbers[i] = issue[1]
	}
	if len(issueNumbers) == 0 {
		fmt.Printf("Error: At least one issue number must be present in %s\n", source)
		isValid = false
	}

	description := ""
	if matches := getMetadataPattern("Description").FindStringSubmatch(content); len(matches) > 1 {
		description = strings.TrimSpace(matches[1])
	}

	author := ""
	if matches := getMetadataPattern("Authors").FindStringSubmatch(content); len(matches) > 1 {
		author = strings.TrimSpace(matches[1])
	} else if matches := getMetadataPattern("Author").FindStringSubmatch(content); len(matches) > 1 {
		author = strings.TrimSpace(matches[1])
	}

	// Extract details (everything after metadata)
	details := ""
	pattern := `(?s)(?:` + strings.Join(metadataFields, ":|") + `:).*?\n\n(.*)`
	if detailsMatch := regexp.MustCompile(pattern).FindStringSubmatch(content); len(detailsMatch) > 1 {
		details = strings.TrimSpace(detailsMatch[1])
	}

	return isValid, feature{
		Component:   component,
		Description: description,
		Author:      author,
		Issues:      issueNumbers,
		Details:     details,
	}, nil
}

func format(version string, features []feature) string {
	var output strings.Builder

	// Format new content
	versionHeader := "Unreleased"
	if version != "" {
		versionHeader = version
	}

	currentDate := timeNow().Format("2006-01-02")
	output.WriteString(fmt.Sprintf("# New features in %s (%s)\n\nThis is a concise list of new features.\n\n", versionHeader, currentDate))

	// Group features by component
	featuresByComponent := make(map[string][]feature)
	for _, f := range features {
		featuresByComponent[f.Component] = append(featuresByComponent[f.Component], f)
	}

	// Output features in order of validComponents
	for _, component := range validComponents {
		componentFeatures := featuresByComponent[component]
		if len(componentFeatures) == 0 {
			continue
		}

		output.WriteString(fmt.Sprintf("## %s\n\n", component))

		for _, feature := range componentFeatures {
			issuesStr := ""
			if len(feature.Issues) > 0 {
				issues := make([]string, len(feature.Issues))
				for i, issue := range feature.Issues {
					issues[i] = fmt.Sprintf("[#%s](https://github.com/argoproj/argo-workflows/issues/%s)", issue, issue)
				}
				issuesStr = fmt.Sprintf("(%s)", strings.Join(issues, ", "))
			}

			output.WriteString(fmt.Sprintf("- %s by %s %s\n", feature.Description, feature.Author, issuesStr))

			if feature.Details != "" {
				for line := range strings.SplitSeq(feature.Details, "\n") {
					if line != "" {
						output.WriteString(fmt.Sprintf("  %s\n", line))
					}
				}
			}

			output.WriteString("\n")
		}
	}

	return output.String()
}
