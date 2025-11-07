//go:build !windows

// Not tested on windows as this is not os specific and path matching
// differs on that platform
package v1alpha1

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsinternal "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	fileutil "github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

const (
	// repoRoot is the relative path from this test file to the repository root
	repoRoot = "../../../.."
)

// TestCRDExamples validates all YAML files in the examples directory against the CRD schemas
func TestCRDExamples(t *testing.T) {
	ctx := logging.TestContext(context.Background())

	// Load CRDs from manifests
	crds, err := loadCRDs(ctx, filepath.Join(repoRoot, "manifests", "base", "crds", "full"))
	require.NoError(t, err, "Failed to load CRDs")

	t.Logf("Loaded %d CRDs", len(crds))

	// Validate resources from both examples and test/e2e directories
	testDirs := []string{
		filepath.Join(repoRoot, "examples"),
		filepath.Join(repoRoot, "test", "e2e"),
	}

	for _, testDir := range testDirs {
		err = fileutil.WalkManifests(ctx, testDir, func(path string, data []byte) error {
			// Skip .json files
			if filepath.Ext(path) == ".json" {
				return nil
			}

			// Skip directories with malformed or expectedfailures in the path
			pathLower := strings.ToLower(path)
			if strings.Contains(pathLower, "/malformed/") ||
				strings.Contains(pathLower, "/expectedfailures/") ||
				strings.Contains(pathLower, "/lintfail/") {
				return nil
			}

			// Parse the YAML file into resources
			resources, err := parseYAMLResources(data)
			if err != nil {
				// Log but don't fail on parsing errors
				// (may be non-K8s files, but log so someone might notice if they're supposed to be checked)
				t.Logf("Skipping %s: failed to parse as Kubernetes resource: %v", path, err)
				return nil
			}

			// Validate each resource
			for _, resource := range resources {
				gvk := resource.GroupVersionKind()

				// Find matching CRD
				crd := findMatchingCRD(crds, gvk)
				if crd == nil {
					// Not an Argo Workflows CRD, skip
					continue
				}

				// Get resource name for subtest
				name := resource.GetName()
				if name == "" {
					name = resource.GetGenerateName()
				}

				// Skip resources with "invalid" or "malformed" in their name
				nameLower := strings.ToLower(name)
				if strings.Contains(nameLower, "invalid") || strings.Contains(nameLower, "malformed") {
					continue
				}

				// Strip repoRoot prefix from path for cleaner test names
				cleanPath := strings.TrimPrefix(path, repoRoot+string(filepath.Separator))

				// Format: <path>:<CRDType>/<Name>
				testName := fmt.Sprintf("%s:%s/%s", cleanPath, gvk.Kind, name)

				t.Run(testName, func(t *testing.T) {
					t.Parallel()
					// Validate the resource against the CRD schema
					err := validateResourceAgainstCRD(ctx, resource, crd)
					require.NoError(t, err, "validation failed")
				})
			}

			return nil
		})
	}
}

// loadCRDs loads all CRD definitions from the specified directory
func loadCRDs(ctx context.Context, crdPath string) (map[string]*apiextensionsv1.CustomResourceDefinition, error) {
	crds := make(map[string]*apiextensionsv1.CustomResourceDefinition)

	err := fileutil.WalkManifests(ctx, crdPath, func(path string, data []byte) error {
		// Skip kustomization.yaml
		if strings.Contains(path, "kustomization.yaml") {
			return nil
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(data, crd); err != nil {
			return fmt.Errorf("failed to parse CRD from %s: %w", path, err)
		}

		if crd.Kind == "CustomResourceDefinition" {
			crds[crd.Name] = crd
		}

		return nil
	})

	return crds, err
}

// parseYAMLResources parses a YAML file that may contain multiple resources
func parseYAMLResources(data []byte) ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured

	// Split by YAML document separator
	separator := []byte("\n---\n")
	docs := strings.Split(string(data), string(separator))

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		un := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), un); err != nil {
			return nil, err
		}

		// Skip empty documents or non-Kubernetes resources
		if un.GetKind() == "" {
			continue
		}

		resources = append(resources, un)
	}

	return resources, nil
}

// findMatchingCRD finds the CRD that matches the given GroupVersionKind
func findMatchingCRD(crds map[string]*apiextensionsv1.CustomResourceDefinition, gvk apischema.GroupVersionKind) *apiextensionsv1.CustomResourceDefinition {
	for _, crd := range crds {
		if crd.Spec.Group == gvk.Group && crd.Spec.Names.Kind == gvk.Kind {
			return crd
		}
	}
	return nil
}

// validateResourceAgainstCRD validates a resource against its CRD schema including CEL rules
func validateResourceAgainstCRD(ctx context.Context, resource *unstructured.Unstructured, crd *apiextensionsv1.CustomResourceDefinition) error {
	// Find the version schema
	var schemaProps *apiextensionsv1.JSONSchemaProps
	resourceVersion := resource.GroupVersionKind().Version

	for _, version := range crd.Spec.Versions {
		if version.Name == resourceVersion {
			if version.Schema != nil && version.Schema.OpenAPIV3Schema != nil {
				schemaProps = version.Schema.OpenAPIV3Schema
				break
			}
		}
	}

	if schemaProps == nil {
		return fmt.Errorf("no schema found for version %s in CRD %s", resourceVersion, crd.Name)
	}

	// Convert v1 JSONSchemaProps to internal version
	internalSchema := &apiextensionsinternal.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(schemaProps, internalSchema, nil); err != nil {
		return fmt.Errorf("failed to convert schema: %w", err)
	}

	// Enable strict mode: disallow unknown fields by setting additionalProperties to false recursively
	enableStrictValidation(internalSchema)

	// Create structural schema
	structural, err := schema.NewStructural(internalSchema)
	if err != nil {
		return fmt.Errorf("failed to create structural schema: %w", err)
	}

	// Validate the structural schema is valid
	if errs := schema.ValidateStructural(nil, structural); len(errs) > 0 {
		return fmt.Errorf("invalid structural schema: %v", errs.ToAggregate())
	}

	// Create a validator with CEL support
	// NewSchemaValidator returns (validator, *Schema, error)
	validator, _, err := validation.NewSchemaValidator(internalSchema)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	// Validate the resource
	obj := resource.UnstructuredContent()
	if obj == nil {
		return fmt.Errorf("resource has no content")
	}

	// Perform validation including CEL rules
	// For CREATE operations, oldObj is nil
	errs := validation.ValidateCustomResource(field.NewPath(""), obj, validator)

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs.ToAggregate())
	}

	return nil
}

// enableStrictValidation recursively sets AdditionalProperties to false on all objects in the schema
// to enforce strict validation that disallows unknown fields
func enableStrictValidation(schema *apiextensionsinternal.JSONSchemaProps) {
	enableStrictValidationHelper(schema, true)
}

func enableStrictValidationHelper(schema *apiextensionsinternal.JSONSchemaProps, isRoot bool) {
	if schema == nil {
		return
	}

	// Skip if x-kubernetes-preserve-unknown-fields is true (allows arbitrary fields)
	if schema.XPreserveUnknownFields != nil && *schema.XPreserveUnknownFields {
		return
	}

	// Don't set additionalProperties on root level or if it's already set
	if !isRoot && schema.Type == "object" && schema.AdditionalProperties == nil {
		schema.AdditionalProperties = &apiextensionsinternal.JSONSchemaPropsOrBool{
			Allows: false,
		}
	}

	// Recursively process nested schemas
	if schema.Properties != nil {
		for key := range schema.Properties {
			prop := schema.Properties[key]
			// Skip metadata and preserve its special handling
			if isRoot && (key == "metadata" || key == "apiVersion" || key == "kind") {
				continue
			}
			enableStrictValidationHelper(&prop, false)
			schema.Properties[key] = prop
		}
	}

	if schema.Items != nil {
		if schema.Items.Schema != nil {
			enableStrictValidationHelper(schema.Items.Schema, false)
		}
		for i := range schema.Items.JSONSchemas {
			enableStrictValidationHelper(&schema.Items.JSONSchemas[i], false)
		}
	}

	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		enableStrictValidationHelper(schema.AdditionalProperties.Schema, false)
	}

	// Handle allOf, anyOf, oneOf
	for i := range schema.AllOf {
		enableStrictValidationHelper(&schema.AllOf[i], false)
	}
	for i := range schema.AnyOf {
		enableStrictValidationHelper(&schema.AnyOf[i], false)
	}
	for i := range schema.OneOf {
		enableStrictValidationHelper(&schema.OneOf[i], false)
	}

	if schema.Not != nil {
		enableStrictValidationHelper(schema.Not, false)
	}
}
