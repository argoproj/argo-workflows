package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Set this to the root of the repo
const configDir = "config"

//go:embed workflow-controller-configmap.md
var header string

const outputFile = "docs/workflow-controller-configmap.md"

// visited tracks which types we've already documented
var visited = map[string]bool{}

// typeSpecs maps type name to its *ast.TypeSpec
var typeSpecs = map[string]*ast.TypeSpec{}

// typeComments maps type name to its documentation comment
var typeComments = map[string]*ast.CommentGroup{}

// documentedTypes tracks which types we will document in this file
var documentedTypes = map[string]bool{}

func generateConfigDocs() {
	fset := token.NewFileSet()

	// Parse all .go files in the config directory
	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip test files and non-go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("failed to parse %s: %v", path, err)
			return nil // Continue with other files
		}

		// Collect all type specs in this file
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok {
					typeSpecs[ts.Name.Name] = ts
					// Associate the GenDecl's comment with this type
					if gd.Doc != nil {
						typeComments[ts.Name.Name] = gd.Doc
					}
					documentedTypes[ts.Name.Name] = true
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("failed to walk config directory: %v", err)
	}

	var buf bytes.Buffer
	buf.WriteString(header)
	if ts, ok := typeSpecs["Config"]; ok {
		writeStructDoc(&buf, ts, "Config")
	} else {
		log.Fatalf("Config struct not found in %s directory", configDir)
	}

	err = os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		log.Fatalf("failed to write output: %v", err)
	}
	fmt.Printf("Wrote %s\n", outputFile)
}

func writeStructDoc(buf *bytes.Buffer, ts *ast.TypeSpec, name string) {
	if visited[name] {
		return
	}
	visited[name] = true

	// Check if this is actually a struct type first
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		// This is not a struct (e.g., type alias like TTL or MetricsTemporality)
		// Don't create a section for it
		return
	}

	buf.WriteString(fmt.Sprintf("\n## %s\n\n", name))
	// Check for comment from GenDecl first, then TypeSpec
	if comment, ok := typeComments[name]; ok && comment != nil {
		buf.WriteString(normalizeComment(comment.Text()) + "\n\n")
	} else if ts.Doc != nil {
		buf.WriteString(normalizeComment(ts.Doc.Text()) + "\n\n")
	}
	buf.WriteString("### Fields\n\n")
	buf.WriteString("| Field Name | Field Type | Description |\n")
	buf.WriteString("|:----------:|:----------:|:------------|\n")

	// Collect types to recurse into after processing all fields
	var typesToRecurse []string

	for _, field := range st.Fields.List {
		// Get field name(s) - handle both named and embedded fields
		var names []string
		if len(field.Names) == 0 {
			// Embedded field - use type name
			names = []string{exprString(field.Type)}
		} else {
			for _, n := range field.Names {
				names = append(names, n.Name)
			}
		}

		// Get type and create link
		typeStr := exprString(field.Type)
		linkedTypeStr := createTypeLink(typeStr)

		// Get documentation with fallback
		doc := "-"
		if field.Doc != nil {
			doc = normalizeComment(field.Doc.Text())
		} else if field.Comment != nil {
			doc = normalizeComment(field.Comment.Text())
		}
		if doc == "" {
			doc = "-"
		}

		// Write table rows for all field names
		for _, fname := range names {
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", fname, linkedTypeStr, doc))
		}

		// Collect types to recurse into later
		if baseType := baseTypeName(typeStr); typeSpecs[baseType] != nil && !visited[baseType] {
			typesToRecurse = append(typesToRecurse, baseType)
		}
	}

	// Now recurse into all the collected types
	for _, baseType := range typesToRecurse {
		if tts, ok := typeSpecs[baseType]; ok && !visited[baseType] {
			writeStructDoc(buf, tts, baseType)
		}
	}
}

// createTypeLink creates markdown links for type references
func createTypeLink(typeStr string) string {
	// Remove leading asterisks for display purposes
	displayType := strings.TrimPrefix(typeStr, "*")
	baseType := baseTypeName(typeStr)

	// Check if this is a type alias that we should document inline
	if inlineDoc := getInlineTypeDoc(baseType); inlineDoc != "" {
		return fmt.Sprintf("`%s` (%s)", displayType, inlineDoc)
	}

	// For complex types (maps, slices), we need to handle them specially
	if strings.Contains(displayType, "[") || strings.Contains(displayType, "map") {
		return createComplexTypeLink(displayType, baseType)
	}

	// Simple types - create appropriate links
	return createSimpleTypeLink(displayType, baseType)
}

// createTypeLinkWithSpacing creates a type link and returns both the link and whether it has a link
func createTypeLinkWithSpacing(baseType string) (string, bool) {
	cleanBaseType := strings.TrimPrefix(baseType, "*")

	if documentedTypes[baseType] {
		return fmt.Sprintf("[`%s`](#%s)", cleanBaseType, strings.ToLower(baseType)), true
	}

	if strings.HasPrefix(baseType, "wfv1.") {
		wfType := strings.TrimPrefix(baseType, "wfv1.")
		return fmt.Sprintf("[`%s`](fields.md#%s)", wfType, strings.ToLower(wfType)), true
	}

	if strings.HasPrefix(baseType, "apiv1.") {
		typeName := strings.TrimPrefix(baseType, "apiv1.")
		anchor := strings.ToLower(typeName)
		return fmt.Sprintf("[`%s`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#%s-v1-core)", typeName, anchor), true
	}

	if strings.HasPrefix(baseType, "metav1.") {
		typeName := strings.TrimPrefix(baseType, "metav1.")
		anchor := strings.ToLower(typeName)
		return fmt.Sprintf("[`%s`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#%s-v1-meta)", typeName, anchor), true
	}

	// For simple types like string, int, etc., just use the type name
	return cleanBaseType, false
}

// createComplexTypeLink handles complex types like maps, slices, and pointers
func createComplexTypeLink(displayType, baseType string) string {
	// Handle map types - convert to Map< key , value > format like fields.go
	if mapPattern := regexp.MustCompile(`^map\[([^\]]+)\](.+)$`); strings.HasPrefix(displayType, "map[") {
		if matches := mapPattern.FindStringSubmatch(displayType); len(matches) == 3 {
			keyType, valueType := matches[1], matches[2]
			valueBaseType := baseTypeName(valueType)

			valueLink, hasLink := createTypeLinkWithSpacing(valueBaseType)

			// Format with or without spaces based on whether we have links
			if hasLink {
				return fmt.Sprintf("`Map<%s,`%s`>`", keyType, valueLink)
			} else {
				return fmt.Sprintf("`Map<%s,%s>`", keyType, valueLink)
			}
		}
	}

	// Handle slice types - convert to Array<> format like fields.go
	for _, prefix := range []string{"*[]", "[]"} {
		if strings.HasPrefix(displayType, prefix) {
			elementType := displayType[len(prefix):]
			elementBaseType := baseTypeName(elementType)

			elementLink, hasLink := createTypeLinkWithSpacing(elementBaseType)

			// Format with or without spaces based on whether we have links
			if hasLink {
				return fmt.Sprintf("`Array<`%s`>`", elementLink)
			} else {
				return fmt.Sprintf("`Array<%s>`", elementLink)
			}
		}
	}

	return fmt.Sprintf("`%s`", displayType)
}

// createSimpleTypeLink creates links for simple (non-complex) types
func createSimpleTypeLink(displayType, baseType string) string {
	cleanBaseType := strings.TrimPrefix(baseType, "*")

	// Check if this is a type we document in this file
	if documentedTypes[baseType] {
		return fmt.Sprintf("[`%s`](#%s)", cleanBaseType, strings.ToLower(baseType))
	}

	// Define external type mappings
	externalTypes := map[string]string{
		"wfv1.":   "fields.md#%s",
		"apiv1.":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#%s-v1-core",
		"metav1.": "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#%s-v1-meta",
	}

	// Check for external type prefixes
	for prefix, urlTemplate := range externalTypes {
		if strings.HasPrefix(displayType, prefix) || strings.HasPrefix(baseType, prefix) {
			typeName := strings.TrimPrefix(baseType, prefix)
			anchor := strings.ToLower(typeName)
			return fmt.Sprintf("[`%s`]("+urlTemplate+")", cleanBaseType, anchor)
		}
	}

	// For other types, just add backticks
	return fmt.Sprintf("`%s`", displayType)
}

// exprString returns the string representation of an ast.Expr
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(e.Key), exprString(e.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct" // anonymous struct
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// baseTypeName strips pointer, slice, and map to get the base type name
func baseTypeName(typeStr string) string {
	t := typeStr
	for {
		switch {
		case strings.HasPrefix(t, "*"):
			t = t[1:]
		case strings.HasPrefix(t, "[]"):
			t = t[2:]
		case strings.HasPrefix(t, "map["):
			if closeIdx := strings.Index(t, "]"); closeIdx != -1 {
				t = t[closeIdx+1:]
			} else {
				return t
			}
		default:
			return t
		}
	}
}

// normalizeComment converts multi-line comments into single-line descriptions
func normalizeComment(comment string) string {
	if comment == "" {
		return ""
	}

	// Replace newlines with spaces
	result := strings.ReplaceAll(comment, "\n", " ")

	// Remove // comment markers, but be careful with URLs
	// Split on spaces, process each word, then rejoin
	words := strings.Fields(result)
	var cleanWords []string
	for _, word := range words {
		// Skip removing // if it's part of a URL
		if strings.Contains(word, "://") {
			cleanWords = append(cleanWords, word)
		} else {
			// Remove // from the beginning of words (comment markers)
			cleanWord := strings.TrimPrefix(word, "//")
			if cleanWord != "" {
				cleanWords = append(cleanWords, cleanWord)
			}
		}
	}

	return strings.Join(cleanWords, " ")
}

// getInlineTypeDoc returns inline documentation for type aliases from AST
func getInlineTypeDoc(typeName string) string {
	ts, exists := typeSpecs[typeName]
	if !exists {
		return ""
	}

	// Only handle type aliases, not structs
	if _, isStruct := ts.Type.(*ast.StructType); isStruct {
		return ""
	}

	// Get comment from GenDecl or TypeSpec
	var comment string
	if commentGroup, ok := typeComments[typeName]; ok && commentGroup != nil {
		comment = strings.TrimSpace(commentGroup.Text())
	} else if ts.Doc != nil {
		comment = strings.TrimSpace(ts.Doc.Text())
	}

	// Get underlying type
	underlyingType := exprString(ts.Type)

	// Format result based on available information
	if comment != "" && underlyingType != "" {
		return fmt.Sprintf("%s (underlying type: %s)", comment, underlyingType)
	}
	if underlyingType != "" {
		return fmt.Sprintf("(underlying type: %s)", underlyingType)
	}
	return comment
}
