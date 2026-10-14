package parameterization

import (
	"fmt"
	"maps"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// resolveAction freezes one native leaf, not the library Workflow spec. The
// calling Workflow continues to own service accounts, defaults and scheduling.
func resolveAction(def ActionDefinition, lookup func(*wfv1.TemplateRef) (*wfv1.Template, error)) (ActionDefinition, error) {
	if (def.Template == nil) == (def.ImplementationRef == nil) {
		return def, fmt.Errorf("action requires exactly one of template or implementationRef")
	}
	if ref := def.ImplementationRef; ref != nil {
		if !staticTemplateName(ref.Name) || !staticTemplateName(ref.Template) {
			return def, fmt.Errorf("action implementationRef names must be static")
		}
		template, err := lookup(ref)
		if err != nil {
			return def, err
		}
		def.Template = template
		def.ImplementationRef = nil
	}
	return def, validateAction(def)
}

func actionTemplate(source wfv1.TemplateHolder, ref *wfv1.TemplateRef) (*wfv1.Template, error) {
	template := source.GetTemplateByName(ref.Template)
	if template == nil {
		return nil, fmt.Errorf("action implementation template %s is absent", ref.Template)
	}
	result := template.DeepCopy()
	// Match native templateresolution.GetTemplateFromRef: library pod metadata
	// takes precedence, while the original source object remains unchanged.
	if metadata := source.GetPodMetadata(); metadata != nil {
		if result.Metadata.Annotations == nil {
			result.Metadata.Annotations = map[string]string{}
		}
		if result.Metadata.Labels == nil {
			result.Metadata.Labels = map[string]string{}
		}
		maps.Copy(result.Metadata.Annotations, metadata.Annotations)
		maps.Copy(result.Metadata.Labels, metadata.Labels)
	}
	return result, nil
}
