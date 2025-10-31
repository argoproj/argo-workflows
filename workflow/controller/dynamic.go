package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// AddDynamicInputs expands a JSON-encoded Inputs struct from a parameter into the template
func AddDynamicInputs(ctx context.Context, tmpl *v1alpha1.Template) error {
	log := logging.RequireLoggerFromContext(ctx)

	for _, param := range tmpl.Inputs.Parameters {
		fmt.Println("AddDynamicInputs:", param)
		if param.Name != common.DynamicInputsParameterName {
			continue
		}

		if param.Value == nil {
			continue
		}

		valueStr := param.Value.String()
		if valueStr == "" {
			continue
		}

		var dynamicInputs v1alpha1.Inputs
		if err := json.Unmarshal([]byte(valueStr), &dynamicInputs); err != nil {
			return fmt.Errorf("failed to unmarshal dynamic inputs: %w", err)
		}

		// Append artifacts and parameters
		tmpl.Inputs.Artifacts = append(tmpl.Inputs.Artifacts, dynamicInputs.Artifacts...)
		tmpl.Inputs.Parameters = append(tmpl.Inputs.Parameters, dynamicInputs.Parameters...)

		log.WithField("addedParams", len(dynamicInputs.Parameters)).
			WithField("addedArtifacts", len(dynamicInputs.Artifacts)).
			Debug(ctx, "Expanded dynamic inputs")
	}

	return nil
}
