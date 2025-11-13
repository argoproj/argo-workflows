package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	fileutil "github.com/argoproj/argo-workflows/v3/util/file"
	jsonpkg "github.com/argoproj/argo-workflows/v3/util/json"
	"github.com/argoproj/argo-workflows/v3/workflow/convert"
)

func NewConvertCommand() *cobra.Command {
	var (
		output = common.EnumFlagValue{
			AllowedValues: []string{"yaml", "json"},
			Value:         "yaml",
		}
	)

	command := &cobra.Command{
		Use:   "convert FILE...",
		Short: "convert workflow manifests from legacy format to current format",
		Long:  "Converts singular schedule, mutex and semaphore to the new plural version",
		Example: `
# Convert manifests in a specified directory:

  argo convert ./manifests

# Convert a single file:

  argo convert workflow.yaml

# Convert from stdin:

  cat workflow.yaml | argo convert -`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(cmd.Context(), args, output.String())
		},
	}

	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	command.Flags().BoolVar(&common.NoColor, "no-color", false, "Disable colorized output")

	return command
}

var yamlSeparator = regexp.MustCompile(`\n---`)

func runConvert(ctx context.Context, args []string, output string) error {
	for _, file := range args {
		err := fileutil.WalkManifests(ctx, file, func(path string, data []byte) error {
			if jsonpkg.IsJSON(data) {
				// Parse single JSON document
				if err := convertDocument(data, output, true); err != nil {
					return fmt.Errorf("error converting %s: %w", path, err)
				}
			} else {
				// Split YAML documents
				for _, doc := range yamlSeparator.Split(string(data), -1) {
					doc = strings.TrimSpace(doc)
					if doc == "" {
						continue
					}
					if err := convertDocument([]byte(doc), output, false); err != nil {
						return fmt.Errorf("error converting %s: %w", path, err)
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func convertDocument(data []byte, outputFormat string, isJSON bool) error {
	// First, determine the kind
	var typeMeta metav1.TypeMeta
	if err := yaml.Unmarshal(data, &typeMeta); err != nil {
		return fmt.Errorf("failed to parse TypeMeta: %w", err)
	}

	var converted interface{}

	// Parse into legacy type and convert to current type
	switch typeMeta.Kind {
	case wf.CronWorkflowKind:
		var legacy convert.LegacyCronWorkflow
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return fmt.Errorf("failed to parse CronWorkflow: %w", err)
		}
		converted = legacy.ToCurrent()

	case wf.WorkflowKind:
		var legacy convert.LegacyWorkflow
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return fmt.Errorf("failed to parse Workflow: %w", err)
		}
		converted = legacy.ToCurrent()

	case wf.WorkflowTemplateKind:
		var legacy convert.LegacyWorkflowTemplate
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return fmt.Errorf("failed to parse WorkflowTemplate: %w", err)
		}
		converted = legacy.ToCurrent()

	case wf.ClusterWorkflowTemplateKind:
		var legacy convert.LegacyClusterWorkflowTemplate
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return fmt.Errorf("failed to parse ClusterWorkflowTemplate: %w", err)
		}
		converted = legacy.ToCurrent()

	default:
		// Unknown type - pass through unchanged
		// Re-parse as generic map to preserve structure
		var generic map[string]interface{}
		if err := yaml.Unmarshal(data, &generic); err != nil {
			return fmt.Errorf("failed to parse unknown kind %s: %w", typeMeta.Kind, err)
		}
		converted = generic
	}

	return outputObject(converted, outputFormat, isJSON)
}

func outputObject(obj interface{}, format string, preferJSON bool) error {
	var outBytes []byte
	var err error

	// Output JSON if format is "json", or if input was JSON and format is not explicitly "yaml"
	// This preserves input format by default while allowing explicit format override
	outputJSON := format == "json" || (preferJSON && format != "yaml")

	if outputJSON {
		outBytes, err = json.Marshal(obj)
		if err != nil {
			return err
		}
		fmt.Println(string(outBytes))
	} else {
		outBytes, err = yaml.Marshal(obj)
		if err != nil {
			return err
		}
		// Print separator between objects for YAML
		if _, err := os.Stdout.Write([]byte("---\n")); err != nil {
			return err
		}
		if _, err := os.Stdout.Write(outBytes); err != nil {
			return err
		}
	}

	return nil
}
