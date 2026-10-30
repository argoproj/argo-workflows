package controller

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

func (woc *wfOperationCtx) executePluginTemplate(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		if opts.boundaryID == "" {
			woc.log.Debug(ctx, "boundaryID was nil")
		}
		_, node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypePlugin, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag, true)
	}
	if !node.Fulfilled() {
		task := *tmpl
		if err := woc.addPluginArchiveLocation(ctx, nodeName, &task); err != nil {
			return node, err
		}
		woc.taskSet[node.ID] = task
	}
	return node, nil
}

// addPluginArchiveLocation gives a plugin template with artifacts the archive
// location the agent stores them in, as addArchiveLocation does for a pod's
// template. There is no pod per plugin node, so {{pod.name}} in the location
// is the name the node's pod would have.
func (woc *wfOperationCtx) addPluginArchiveLocation(ctx context.Context, nodeName string, tmpl *wfv1.Template) error {
	artifacts := slices.Concat(tmpl.Inputs.Artifacts, tmpl.Outputs.Artifacts)
	if len(artifacts) == 0 {
		return nil
	}
	if !tmpl.ArchiveLocation.HasLocation() && slices.ContainsFunc(artifacts, func(art wfv1.Artifact) bool { return !art.HasLocation() }) {
		tmpl.ArchiveLocation = woc.artifactRepository.ToArtifactLocation()
	}
	if tmpl.ArchiveLocation == nil {
		return nil
	}
	params := woc.globalParams()
	for _, param := range tmpl.Inputs.Parameters {
		params[varkeys.InputsParameterByName.Concretize(param.Name)] = param.Value.String()
	}
	params[varkeys.PodName.Template()] = woc.getPodName(nodeName, tmpl.Name)
	locationBytes, err := json.Marshal(tmpl.ArchiveLocation)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	resolved, err := template.Replace(ctx, string(locationBytes), template.ToAnyMap(params), true)
	if err != nil {
		return err
	}
	location := &wfv1.ArtifactLocation{}
	if err := json.Unmarshal([]byte(resolved), location); err != nil {
		return errors.InternalWrapError(err)
	}
	if err := validate.VerifyResolvedVariables(location); err != nil {
		return err
	}
	tmpl.ArchiveLocation = location
	return nil
}
