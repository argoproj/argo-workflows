package parameterization

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Freeze references into native templates, never into a second scheduling graph.
// The caller's client resolves each resource once; the execution digest covers
// the complete closure. Argo still interprets depends/when/loops/outputs/hooks.
func freezeTemplateClosure(ctx context.Context, namespace string, wf *wfv1.Workflow, versions map[string]string) error {
	if wf.Spec.WorkflowTemplateRef != nil {
		return fmt.Errorf("a bound WorkflowTemplate must contain its Workflow spec, not another workflowTemplateRef")
	}
	f := &templateClosure{namespace: namespace, versions: versions,
		sources: map[string]wfv1.TemplateHolder{"root": wf}, names: map[string]string{}, used: map[string]bool{}}
	for _, t := range wf.Spec.Templates {
		if f.used[t.Name] {
			return fmt.Errorf("duplicate native template %s", t.Name)
		}
		f.used[t.Name] = true
	}
	for i := range wf.Spec.Templates {
		if err := f.rewrite(ctx, "root", &wf.Spec.Templates[i], 0); err != nil {
			return err
		}
	}
	if err := f.hooks(ctx, "root", wf.Spec.Hooks, 0); err != nil {
		return err
	}
	for _, t := range f.frozen {
		wf.Spec.Templates = append(wf.Spec.Templates, *t)
	}
	return nil
}

type templateClosure struct {
	namespace string
	versions  map[string]string
	sources   map[string]wfv1.TemplateHolder
	names     map[string]string
	used      map[string]bool
	frozen    []*wfv1.Template
}

func staticTemplateName(name string) bool { return name != "" && !strings.ContainsAny(name, "{}") }

func (f *templateClosure) source(ctx context.Context, ref *wfv1.TemplateRef) (string, error) {
	if !staticTemplateName(ref.Name) || !staticTemplateName(ref.Template) {
		return "", fmt.Errorf("templateRef names must be static to freeze their execution versions")
	}
	key := "WorkflowTemplate/" + f.namespace + "/" + ref.Name
	if ref.ClusterScope {
		key = "ClusterWorkflowTemplate/" + ref.Name
	}
	if _, ok := f.sources[key]; ok {
		return key, nil
	}
	if len(f.sources) >= 64 {
		return "", fmt.Errorf("template reference closure exceeds 64 resources")
	}
	var source wfv1.TemplateHolder
	var version string
	if ref.ClusterScope {
		x, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		source = x
		version = string(x.UID) + ":" + x.ResourceVersion + ":" + p.Digest(x.Spec)
	} else {
		x, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates(f.namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		source = x
		version = string(x.UID) + ":" + x.ResourceVersion + ":" + p.Digest(x.Spec)
	}
	if previous, exists := f.versions["templateRef/"+key]; exists && previous != version {
		return "", fmt.Errorf("template source changed while resolving execution; preview again")
	}
	f.sources[key] = source
	f.versions["templateRef/"+key] = version
	return key, nil
}

func (f *templateClosure) freeze(ctx context.Context, scope, name string, depth int) (string, error) {
	if !staticTemplateName(name) {
		return "", fmt.Errorf("template names must be static to freeze their execution versions")
	}
	source := f.sources[scope]
	original := source.GetTemplateByName(name)
	if original == nil {
		return "", fmt.Errorf("template %s is absent in %s", name, scope)
	}
	if scope == "root" {
		return name, nil
	}
	key := scope + "/" + name
	if frozen, ok := f.names[key]; ok {
		return frozen, nil
	}
	if len(f.frozen) >= 256 {
		return "", fmt.Errorf("template reference closure exceeds 256 templates")
	}
	frozen := "bound-" + p.Digest(key)[:32]
	if f.used[frozen] {
		return "", fmt.Errorf("frozen template name collides with an existing template")
	}
	f.used[frozen] = true
	// Reserve before descent so native recursive calls retain their identity.
	f.names[key] = frozen
	t := original.DeepCopy()
	t.Name = frozen
	if metadata := source.GetPodMetadata(); metadata != nil {
		if t.Metadata.Annotations == nil {
			t.Metadata.Annotations = map[string]string{}
		}
		if t.Metadata.Labels == nil {
			t.Metadata.Labels = map[string]string{}
		}
		// Same precedence as native templateresolution.Context.addPodMetadata.
		maps.Copy(t.Metadata.Annotations, metadata.Annotations)
		maps.Copy(t.Metadata.Labels, metadata.Labels)
	}
	f.frozen = append(f.frozen, t)
	if err := f.rewrite(ctx, scope, t, depth+1); err != nil {
		return "", err
	}
	return frozen, nil
}

func (f *templateClosure) call(ctx context.Context, scope string, name *string, ref **wfv1.TemplateRef, inline *wfv1.Template, depth int) error {
	count := 0
	if *name != "" {
		count++
	}
	if *ref != nil {
		count++
	}
	if inline != nil {
		count++
	}
	if count > 1 {
		return fmt.Errorf("native call has conflicting template, templateRef or inline targets")
	}
	if inline != nil {
		return f.rewrite(ctx, scope, inline, depth+1)
	}
	if *ref != nil {
		target, err := f.source(ctx, *ref)
		if err != nil {
			return err
		}
		frozen, err := f.freeze(ctx, target, (*ref).Template, depth)
		if err != nil {
			return err
		}
		*name, *ref = frozen, nil
	} else if *name != "" {
		frozen, err := f.freeze(ctx, scope, *name, depth)
		if err != nil {
			return err
		}
		*name = frozen
	}
	return nil
}

func (f *templateClosure) hooks(ctx context.Context, scope string, hooks wfv1.LifecycleHooks, depth int) error {
	keys := make([]string, 0, len(hooks))
	for key := range hooks {
		keys = append(keys, string(key))
	}
	sort.Strings(keys) // Stable closure order, independent of Go map traversal.
	for _, key := range keys {
		h := hooks[wfv1.LifecycleEvent(key)]
		if err := f.call(ctx, scope, &h.Template, &h.TemplateRef, nil, depth); err != nil {
			return err
		}
		hooks[wfv1.LifecycleEvent(key)] = h
	}
	return nil
}

func (f *templateClosure) exit(ctx context.Context, scope string, name *string, depth int) error {
	if *name == "" {
		return nil
	}
	frozen, err := f.freeze(ctx, scope, *name, depth)
	if err == nil {
		*name = frozen
	}
	return err
}

func (f *templateClosure) rewrite(ctx context.Context, scope string, t *wfv1.Template, depth int) error {
	if depth > 64 {
		return fmt.Errorf("template reference/inline nesting exceeds 64")
	}
	if t.DAG != nil {
		for i := range t.DAG.Tasks {
			x := &t.DAG.Tasks[i]
			if err := f.call(ctx, scope, &x.Template, &x.TemplateRef, x.Inline, depth); err != nil {
				return err
			}
			// Freeze legacy onExit too; dropping it would change existing workflows.
			if err := f.exit(ctx, scope, &x.OnExit, depth); err != nil {
				return err
			}
			if err := f.hooks(ctx, scope, x.Hooks, depth); err != nil {
				return err
			}
		}
	}
	for i := range t.Steps {
		for j := range t.Steps[i].Steps {
			x := &t.Steps[i].Steps[j]
			if err := f.call(ctx, scope, &x.Template, &x.TemplateRef, x.Inline, depth); err != nil {
				return err
			}
			// Freeze legacy onExit too; dropping it would change existing workflows.
			if err := f.exit(ctx, scope, &x.OnExit, depth); err != nil {
				return err
			}
			if err := f.hooks(ctx, scope, x.Hooks, depth); err != nil {
				return err
			}
		}
	}
	return nil
}
