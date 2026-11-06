package pool

import (
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Pooled reports whether tmpl, with the workflow's templateDefaults merged
// in, runs on a hot worker pool: it is a container or script template with
// pool set. Other templates ignore a pool they inherit from templateDefaults,
// as they ignore an inherited timeout; validation rejects pool set on one
// directly.
func Pooled(tmpl *wfv1.Template) bool {
	if tmpl.Pool == nil {
		return false
	}
	t := tmpl.GetType()
	return t == wfv1.TemplateTypeContainer || t == wfv1.TemplateTypeScript
}
