package main

import (
	"strings"
)

// Valid component names.
// This is the order they will appear in the new feature file.
// Try not to just add too many components, this is just for categorization in the feature docs.
var validComponents = []string{
	"General",
	"UI",
	"CLI",
	"CronWorkflows",
	"Telemetry",
	"Build and Development",
}

func isValidComponent(component string) bool {
	for _, c := range validComponents {
		if c == component {
			return true
		}
	}
	return false
}

func listValidComponents() string {
	return strings.Join(validComponents, ", ")
}
