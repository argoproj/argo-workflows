package parameterization

import (
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func artifactNames(artifacts []wfv1.Artifact) []string {
	names := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		names = append(names, artifact.Name)
	}
	return names
}

func artifactByName(artifacts []wfv1.Artifact, name string) wfv1.Artifact {
	for _, artifact := range artifacts {
		if artifact.Name == name {
			return artifact
		}
	}
	return wfv1.Artifact{}
}

// mapActionArtifacts connects native references. The implementation owns file
// paths, storage and archive format; the adapter retains the caller's names,
// optionality and global exports. No file is read by the parameter resolver.
func mapActionArtifacts(baseline, implementation *wfv1.Template, call ActionCall, adapter *wfv1.Template) error {
	inputs, err := completeActionMappings(call.InputArtifactMappings, artifactNames(baseline.Inputs.Artifacts), artifactNames(implementation.Inputs.Artifacts))
	if err != nil {
		return fmt.Errorf("input artifact mapping: %w", err)
	}
	outputs, err := completeActionMappings(call.OutputArtifactMappings, artifactNames(implementation.Outputs.Artifacts), artifactNames(baseline.Outputs.Artifacts))
	if err != nil {
		return fmt.Errorf("output artifact mapping: %w", err)
	}
	for i := range adapter.Inputs.Artifacts {
		adapter.Inputs.Artifacts[i].Path = ""
		adapter.Inputs.Artifacts[i].Mode = nil
		adapter.Inputs.Artifacts[i].RecurseMode = false
	}
	for _, mapping := range inputs {
		source := artifactByName(baseline.Inputs.Artifacts, mapping.From)
		target := artifactByName(implementation.Inputs.Artifacts, mapping.To)
		if source.Optional && !target.Optional {
			return fmt.Errorf("optional input artifact cannot satisfy a required action artifact")
		}
		adapter.DAG.Tasks[0].Arguments.Artifacts = append(adapter.DAG.Tasks[0].Arguments.Artifacts, wfv1.Artifact{Name: mapping.To, From: "{{inputs.artifacts." + mapping.From + "}}", Optional: source.Optional})
	}
	for _, mapping := range outputs {
		source := artifactByName(implementation.Outputs.Artifacts, mapping.From)
		target := artifactByName(baseline.Outputs.Artifacts, mapping.To)
		if source.Optional && !target.Optional {
			return fmt.Errorf("optional action output cannot satisfy a required output artifact")
		}
		// An implementation must not introduce an additional global export that
		// bypasses the public mapping. The wrapper alone owns that export.
		if source.GlobalName != "" {
			return fmt.Errorf("mapped action artifacts cannot declare implementation global exports")
		}
		adapter.Outputs.Artifacts = append(adapter.Outputs.Artifacts, wfv1.Artifact{Name: mapping.To, From: "{{tasks.invoke.outputs.artifacts." + mapping.From + "}}", Optional: target.Optional, GlobalName: target.GlobalName})
	}
	return nil
}
