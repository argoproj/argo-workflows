package validate

import (
	"encoding/json"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	apiv1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// Poolable checks that tmpl, a template with pool set, can run on a hot worker
// pool under wfSpec, the workflow spec it runs with. It returns nil for a
// template without pool. wfSpec may be nil to check the template alone.
//
// Validation calls it on each template as written, and again with the
// workflow's templateDefaults merged in when pool.Pooled says the merged
// template is pooled. The controller does the same for the merged template it
// dispatches, with the execution workflow's spec, to catch what only merging
// shows. It finds {{pod.name}} only in a template whose variables aren't
// substituted yet, and checks a pool name holding a variable only once it is
// substituted.
func Poolable(tmpl *wfv1.Template, wfSpec *wfv1.WorkflowSpec) error {
	if tmpl.Pool == nil {
		return nil
	}
	prefix := "templates." + tmpl.Name
	if err := validatePool(prefix+".pool", tmpl.Pool); err != nil {
		return err
	}
	if !pool.Pooled(tmpl) {
		return errors.Errorf(errors.CodeBadRequest, "%s.pool: only container and script templates can be pooled, not %s", prefix, tmpl.GetType())
	}
	if err := pool.CheckTemplate(prefix, tmpl); err != nil {
		return err
	}
	if err := rejectPodName(prefix, tmpl); err != nil {
		return err
	}
	for _, art := range tmpl.Inputs.Artifacts {
		if art.Path == "" {
			continue
		}
		// As init-less pods reject it (see addInputArtifactsVolumes), for the
		// same reason.
		if mnt := common.FindVolumeMountNestedUnderPath(tmpl, art.Path); mnt != nil {
			return errors.Errorf(errors.CodeBadRequest,
				"%s.inputs.artifacts.%s.path '%s' is an ancestor of volume mount '%s' (%s), which a pooled template can't have: staging the artifact would clear the mounted volume",
				prefix, art.Name, art.Path, mnt.Name, mnt.MountPath)
		}
	}
	if wfSpec == nil {
		return nil
	}
	if err := pool.CheckWorkflowSpec(wfSpec); err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s: %s", prefix, err)
	}
	return rejectVolumeClaimMounts(prefix, tmpl, wfSpec)
}

// validatePool checks the pool block. Only a name the user sets needs
// checking: the default is always a valid pool name.
func validatePool(prefix string, p *wfv1.Pool) error {
	if p.Name != "" && !strings.Contains(p.Name, "{{") {
		errs := apivalidation.IsDNS1123Label(p.Name)
		if len(p.Name) > pool.NameMaxLength {
			errs = []string{apivalidation.MaxLenError(pool.NameMaxLength)}
		}
		if len(errs) > 0 {
			return errors.Errorf(errors.CodeBadRequest, "%s.name '%s' is not a valid pool name: %s", prefix, p.Name, strings.Join(errs, "; "))
		}
	}
	// A value holding a template variable is checked once substituted.
	for _, setting := range []struct {
		number pool.Number
		value  *intstr.IntOrString
	}{{pool.InitialReplicas, p.InitialReplicas}, {pool.RetentionSeconds, p.RetentionSeconds}} {
		if pool.HasVariable(setting.value) {
			continue
		}
		if _, _, err := setting.number.Value(setting.value); err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s.%s", prefix, err)
		}
	}
	return nil
}

// rejectPodName fails if tmpl refers to {{pod.name}}, in a simple tag or an
// expression: a pooled task has no pod of its own.
func rejectPodName(prefix string, tmpl *wfv1.Template) error {
	data, err := json.Marshal(tmpl)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	vars, err := template.Variables(string(data))
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s: %s", prefix, err)
	}
	if slices.Contains(vars, varkeys.PodName.Template()) {
		return errors.Errorf(errors.CodeBadRequest, "%s: {{%s}} can't be used in a pooled template, which has no pod of its own; use {{%s}}",
			prefix, varkeys.PodName.Template(), varkeys.NodeTaskName.Template())
	}
	return nil
}

// rejectVolumeClaimMounts fails if tmpl mounts one of wfSpec's
// volumeClaimTemplates: the claim is deleted with its workflow while the pool
// still uses it. A mount resolves as a pod's does
// (pool.MountedWorkflowVolumes): the template's volumes, then the workflow's,
// then its claims.
func rejectVolumeClaimMounts(prefix string, tmpl *wfv1.Template, wfSpec *wfv1.WorkflowSpec) error {
	_, unresolved := pool.MountedWorkflowVolumes(tmpl, wfSpec.Volumes)
	for _, name := range unresolved {
		if slices.ContainsFunc(wfSpec.VolumeClaimTemplates, func(c apiv1.PersistentVolumeClaim) bool { return c.Name == name }) {
			return errors.Errorf(errors.CodeBadRequest, "%s: volume '%s' is one of spec.volumeClaimTemplates, which a pooled template can't mount: the claim is deleted with the workflow while the pool still uses it", prefix, name)
		}
	}
	return nil
}

// validatePooled checks a template with pool set, as written, and again once
// the workflow's templateDefaults are merged into it, as the controller
// merges them, if that makes it pooled (pool.Pooled), under the
// workflow-level fields it runs with (see Poolable). A pool inherited from
// templateDefaults pools only container and script templates.
func (tctx *templateValidationCtx) validatePooled(tmpl *wfv1.Template) error {
	wfSpec, err := tctx.pooledWorkflowSpec(tmpl)
	if err != nil || wfSpec == nil {
		return err
	}
	if err := Poolable(tmpl, wfSpec); err != nil {
		return err
	}
	if wfSpec.TemplateDefaults == nil {
		return nil
	}
	merged := tmpl.DeepCopy()
	if err := common.MergeTemplateDefaults(merged, wfSpec.TemplateDefaults); err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s: spec.templateDefaults: %s", tmpl.Name, err)
	}
	if !pool.Pooled(merged) {
		return nil
	}
	if err := Poolable(merged, wfSpec); err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s (with spec.templateDefaults merged in)", err)
	}
	return nil
}

// pooledWorkflowSpec returns the workflow spec tmpl runs with if it is
// pooled, with or without the workflow's templateDefaults, and nil if it
// isn't. It merges tctx.poolSpecs the first time it is needed.
func (tctx *templateValidationCtx) pooledWorkflowSpec(tmpl *wfv1.Template) (*wfv1.WorkflowSpec, error) {
	if tmpl.Pool == nil && !slices.ContainsFunc(tctx.poolSpecs, func(s *wfv1.WorkflowSpec) bool {
		return s.TemplateDefaults != nil && s.TemplateDefaults.Pool != nil
	}) {
		return nil, nil
	}
	if tctx.poolSpec == nil {
		spec, err := mergeWorkflowSpecs(tctx.poolSpecs)
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "merging the workflow spec for pooled templates: %s", err)
		}
		tctx.poolSpec = spec
	}
	return tctx.poolSpec, nil
}

// mergeWorkflowSpecs merges specs, highest priority first, with the merge
// the controller uses for a workflow, its WorkflowTemplate and the workflow
// defaults (JoinWorkflowSpec in workflow/util, which imports this package,
// adds fix-ups for fields pooled templates' checks don't read).
func mergeWorkflowSpecs(specs []*wfv1.WorkflowSpec) (*wfv1.WorkflowSpec, error) {
	if len(specs) == 0 {
		return &wfv1.WorkflowSpec{}, nil
	}
	merged := &wfv1.Workflow{Spec: *specs[0].DeepCopy()}
	for _, spec := range specs[1:] {
		if err := common.MergeTo(&wfv1.Workflow{Spec: *spec.DeepCopy()}, merged); err != nil {
			return nil, err
		}
	}
	return &merged.Spec, nil
}
