// Package pool holds what the controller and validation share about pooled
// templates, which run on hot worker pools instead of a pod of their own: the
// classification of every template and workflow spec field by how it reaches
// a pooled task, and the pool name rules.
package pool

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// This file classifies every field of a pooled task's template and workflow
// spec by how it reaches the task, and copies a pool's constants from
// them by that classification, so the table and the controller's pool spec
// can't disagree. TestFieldTablesCoverTypes fails when a field of a covered
// type has no entry.

// Class is how a template or workflow spec field reaches a pooled task.
type Class int

const (
	// Hashed fields are the same for every task of a pool: they are in
	// spec.pool, so they are hashed. Each distinct value of a parameterised
	// one is a distinct pool.
	Hashed Class = iota + 1
	// PerTask fields may differ between the tasks of a pool: they are left
	// out of spec.pool and the hash, and reach the task through its TaskSet
	// (spec.tasks carries the whole template, spec.dispatch what the
	// controller resolves) or are acted on by the controller alone.
	PerTask
	// Rejected fields can't be set on a pooled template.
	Rejected
	// SplitEnv is container env, the one field split between the pool and
	// the task: valueFrom entries are pool constants, resolved by the
	// kubelet when the worker pod starts; plain value entries are per task.
	SplitEnv
)

// field classifies one struct field.
type field struct {
	class Class
	// into, for a hashed field, names the spec.pool field the controller
	// resolves it into instead of copying it as written.
	into string
	// fields classifies the fields of a struct, or pointer to struct, field
	// whose fields don't all share one class. It replaces class.
	fields fieldTable
	// why explains anything not obvious from the field and its class.
	why string
}

// fieldTable classifies a struct type's fields by Go field name.
type fieldTable map[string]field

var (
	hashed  = field{class: Hashed}
	perTask = field{class: PerTask}
)

func hashedBecause(why string) field { return field{class: Hashed, why: why} }

// resolvedInto is a hashed field the controller resolves into the spec.pool
// field into, instead of copying it as written.
func resolvedInto(into, why string) field {
	return field{class: Hashed, into: into, why: why}
}

func perTaskBecause(why string) field { return field{class: PerTask, why: why} }

func rejected(why string) field { return field{class: Rejected, why: why} }

func nested(fields fieldTable) field { return field{fields: fields} }

// controllerOnly is a per-task field only the controller acts on; no pod sees it.
var controllerOnly = perTaskBecause("controller only")

var containerFields = fieldTable{
	"Name":                     perTaskBecause("ignored: the container is always named main"),
	"Image":                    hashed,
	"Command":                  perTaskBecause("run by main per task; spec.dispatch has the image's entrypoint when it is empty"),
	"Args":                     perTask,
	"WorkingDir":               hashed,
	"Ports":                    hashed,
	"EnvFrom":                  hashedBecause("read when the worker pod starts"),
	"Env":                      {class: SplitEnv},
	"Resources":                hashed,
	"ResizePolicy":             hashed,
	"RestartPolicy":            rejected("a pool's main must restart with its pod, which a Deployment's pods always do"),
	"RestartPolicyRules":       rejected("as restartPolicy"),
	"VolumeMounts":             hashed,
	"VolumeDevices":            hashed,
	"LivenessProbe":            rejected("probes run between tasks too, when nothing they probe is running"),
	"ReadinessProbe":           rejected("as livenessProbe; it would also make idle pool pods unready"),
	"StartupProbe":             rejected("runs once per worker pod, before any task"),
	"Lifecycle":                rejected("hooks run once per worker pod, not per task, as initContainers would (v2)"),
	"TerminationMessagePath":   hashed,
	"TerminationMessagePolicy": hashed,
	"ImagePullPolicy":          hashed,
	"SecurityContext":          hashed,
	"Stdin":                    hashed,
	"StdinOnce":                hashed,
	"TTY":                      hashed,
}

var scriptFields = fieldTable{
	"Container": nested(containerFields),
	"Source":    perTaskBecause("staged by the worker per task"),
}

var retryStrategyFields = fieldTable{
	"Limit":       controllerOnly,
	"RetryPolicy": controllerOnly,
	"Backoff":     controllerOnly,
	"Affinity":    rejected("it changes the pod spec, and so the pool, every attempt"),
	"Expression":  controllerOnly,
}

// artifactsWhy is why inputs and outputs are per task although their secrets
// and plugins are pool constants.
const artifactsWhy = "the secrets and plugins its artifacts use are hashed as spec.pool artifactSecrets and artifactPlugins, so a task's pool has every secret and plugin it needs"

// notContainerOrScript is why a pooled template can't set another template
// type's field.
const notContainerOrScript = "only container and script templates can be pooled"

// podSpecPatchWhy is why a pooled task can't have a pod spec patch.
const podSpecPatchWhy = "it would patch the pool's pod template, shared by every task; that needs its own design (v2)"

// templateFields classifies Template, the merged template of a pooled task.
var templateFields = fieldTable{
	"Name":                         perTaskBecause("a pool can serve several templates; the default pool name comes from it"),
	"Inputs":                       perTaskBecause(artifactsWhy),
	"Outputs":                      perTaskBecause(artifactsWhy),
	"NodeSelector":                 hashedBecause("the boundary template's is folded in where the template has none"),
	"Affinity":                     hashedBecause("as nodeSelector"),
	"Metadata":                     hashed,
	"Daemon":                       rejected("a daemon runs on once its node is ready, and a worker runs one task at a time"),
	"Steps":                        rejected(notContainerOrScript),
	"Container":                    nested(containerFields),
	"ContainerSet":                 rejected(notContainerOrScript),
	"Script":                       nested(scriptFields),
	"Resource":                     rejected(notContainerOrScript),
	"DAG":                          rejected(notContainerOrScript),
	"Suspend":                      rejected(notContainerOrScript),
	"Data":                         rejected(notContainerOrScript),
	"HTTP":                         rejected(notContainerOrScript),
	"Plugin":                       rejected(notContainerOrScript),
	"Volumes":                      hashed,
	"InitContainers":               rejected("they would run once per worker pod, not per task; a pool-level meaning needs its own design (v2)"),
	"Sidecars":                     rejected("they would be shared by every task of the pool; that needs its own design (v2)"),
	"ArchiveLocation":              perTaskBecause("resolved with the task's name, and whether to archive logs, in the task's template; " + artifactsWhy),
	"ActiveDeadlineSeconds":        perTaskBecause("spec.dispatch, counted from the claim, as a pod's is from its start"),
	"RetryStrategy":                nested(retryStrategyFields),
	"Parallelism":                  controllerOnly,
	"FailFast":                     controllerOnly,
	"Tolerations":                  hashedBecause("as nodeSelector"),
	"SchedulerName":                hashed,
	"PriorityClassName":            hashed,
	"ServiceAccountName":           resolvedInto("ServiceAccountName", "with the workflow's"),
	"AutomountServiceAccountToken": resolvedInto("AutomountServiceAccountToken", "with the workflow's"),
	"Executor":                     resolvedInto("ExecutorServiceAccountTokenSecretName", "the caller looks up the executor service account's token Secret"),
	"HostAliases":                  hashed,
	"SecurityContext":              hashed,
	"PodSpecPatch":                 rejected(podSpecPatchWhy),
	"Metrics":                      controllerOnly,
	"Synchronization":              controllerOnly,
	"Memoize":                      controllerOnly,
	"Timeout":                      perTaskBecause("spec.dispatch deadline"),
	"Annotations":                  perTaskBecause("controller only: the node's display name"),
	"PendingTimeout":               perTaskBecause("controller only: it bounds the wait for a claim, from the node's start"),
	"PodResources":                 hashed,
	"ResourceClaims":               hashed,
	"Pool":                         resolvedInto("Name", "the name, defaulted when unset, is hashed; initialReplicas and retentionSeconds go into spec.pool unhashed"),
}

// workflowSpecFields classifies WorkflowSpec, the execution workflow spec of
// a pooled task, with global parameters already substituted.
var workflowSpecFields = fieldTable{
	"Templates":                    perTaskBecause("the task's own template, merged, is classified above"),
	"Entrypoint":                   controllerOnly,
	"Arguments":                    perTaskBecause("substituted into the template before dispatch"),
	"ServiceAccountName":           resolvedInto("ServiceAccountName", "with the template's"),
	"AutomountServiceAccountToken": resolvedInto("AutomountServiceAccountToken", "with the template's"),
	"Executor":                     resolvedInto("ExecutorServiceAccountTokenSecretName", "as the template's executor"),
	"Volumes":                      resolvedInto("WorkflowSpec.Volumes", "only those the template mounts, as substituted by the controller, as a pod gets them"),
	"VolumeClaimTemplates":         perTaskBecause("controller only; a pooled template can't mount one, as its claim belongs to one workflow and a pool serves many"),
	"Parallelism":                  controllerOnly,
	"ArtifactRepositoryRef":        perTaskBecause("the controller resolves it to the selected repository, which is hashed as spec.pool artifactRepository"),
	"Suspend":                      controllerOnly,
	"NodeSelector":                 hashed,
	"Affinity":                     hashed,
	"Tolerations":                  hashed,
	"ImagePullSecrets":             hashed,
	"HostNetwork":                  hashed,
	"DNSPolicy":                    hashed,
	"DNSConfig":                    hashed,
	"OnExit":                       controllerOnly,
	"TTLStrategy":                  controllerOnly,
	"ActiveDeadlineSeconds":        perTaskBecause("the workflow deadline, in spec.dispatch deadline"),
	"Priority":                     controllerOnly,
	"SchedulerName":                hashed,
	"PodGC":                        perTaskBecause("controller only: pool pods are deleted with their pool, not by pod GC"),
	"PodPriorityClassName":         hashed,
	"HostAliases":                  hashed,
	"SecurityContext":              hashed,
	"PodSpecPatch":                 rejected(podSpecPatchWhy),
	"PodDisruptionBudget":          perTaskBecause("controller only: it selects the workflow's pods by label, which pool pods don't carry"),
	"Metrics":                      controllerOnly,
	"Shutdown":                     controllerOnly,
	"WorkflowTemplateRef":          perTaskBecause("controller only; the spec it refers to is already merged"),
	"Synchronization":              controllerOnly,
	"VolumeClaimGC":                controllerOnly,
	"RetryStrategy":                nested(retryStrategyFields),
	"PodMetadata":                  hashed,
	"TemplateDefaults":             perTaskBecause("already merged into the template"),
	"ArchiveLogs":                  perTaskBecause("resolved into the archive location of the task's template"),
	"Hooks":                        controllerOnly,
	"WorkflowMetadata":             controllerOnly,
	"ArtifactGC":                   perTaskBecause("controller only: run by artifact GC pods"),
	"ExecutorPlugins":              perTaskBecause("controller only: sidecars of the agent pod"),
	"PodResources":                 hashed,
	"ResourceClaims":               hashed,
}

// HashedTemplate returns the hashed fields of tmpl, a pooled task's merged
// template, copied as written: everything else, including the fields the
// controller resolves into other spec.pool fields, is left zero. It fails if
// tmpl sets a rejected field, naming it by its JSON path. The result shares
// tmpl's pointers and slices, so callers pass a copy.
func HashedTemplate(tmpl *wfv1.Template) (*wfv1.Template, error) {
	out := &wfv1.Template{}
	if err := copyFields("", onTemplate, templateFields, reflect.ValueOf(tmpl).Elem(), reflect.ValueOf(out).Elem()); err != nil {
		return nil, err
	}
	return out, nil
}

// HashedWorkflowSpec is HashedTemplate for a pooled task's execution workflow
// spec, naming a rejected field under `spec`.
func HashedWorkflowSpec(spec *wfv1.WorkflowSpec) (*wfv1.WorkflowSpec, error) {
	out := &wfv1.WorkflowSpec{}
	if err := copyFields("spec", onWorkflow, workflowSpecFields, reflect.ValueOf(spec).Elem(), reflect.ValueOf(out).Elem()); err != nil {
		return nil, err
	}
	return out, nil
}

// CheckTemplate fails if tmpl, a pooled template, sets a field the tables
// mark rejected, naming the first it finds by its JSON path under path.
func CheckTemplate(path string, tmpl *wfv1.Template) error {
	return copyFields(path, onTemplate, templateFields, reflect.ValueOf(tmpl).Elem(), reflect.New(reflect.TypeFor[wfv1.Template]()).Elem())
}

// CheckWorkflowSpec fails if spec, the workflow spec a pooled template runs
// under, sets a field the tables mark rejected, naming the first it finds under
// `spec`.
func CheckWorkflowSpec(spec *wfv1.WorkflowSpec) error {
	return copyFields("spec", onWorkflow, workflowSpecFields, reflect.ValueOf(spec).Elem(), reflect.New(reflect.TypeFor[wfv1.WorkflowSpec]()).Elem())
}

// What a rejected field can't be set on, in its error.
const (
	onTemplate = "a pooled template"
	onWorkflow = "a workflow that runs a pooled template"
)

// copyFields copies the hashed fields of src, a struct the table classifies,
// into dst, a zero value of the same type, leaving per-task fields and fields
// resolved into other spec.pool fields zero. It fails if src sets a rejected
// field, naming it by its JSON path under path as a field that can't be set
// on what.
func copyFields(path, on string, table fieldTable, src, dst reflect.Value) error {
	for i := range src.NumField() {
		sf := src.Type().Field(i)
		f, ok := table[sf.Name]
		if !ok {
			return fmt.Errorf("%s.%s has no pool classification: say in workflow/pool/fields.go whether it is part of the pool, set per task, or rejected", src.Type(), sf.Name)
		}
		s, d := src.Field(i), dst.Field(i)
		fieldPath := joinFieldPath(path, sf)
		switch {
		case f.fields != nil:
			if err := copyNestedFields(fieldPath, on, f.fields, s, d); err != nil {
				return err
			}
		case f.class == Rejected:
			if !isUnset(s) {
				return errors.Errorf(errors.CodeBadRequest, "%s can't be set on %s", fieldPath, on)
			}
		case f.class == Hashed && f.into == "":
			d.Set(s)
		case f.class == SplitEnv:
			d.Set(reflect.ValueOf(poolEnv(s.Interface().([]apiv1.EnvVar))))
		}
	}
	return nil
}

// copyNestedFields applies table to a struct or pointer-to-struct field,
// leaving a pointer nil when nothing in it is hashed.
func copyNestedFields(path, on string, table fieldTable, src, dst reflect.Value) error {
	if src.Kind() != reflect.Pointer {
		return copyFields(path, on, table, src, dst)
	}
	if src.IsNil() {
		return nil
	}
	copied := reflect.New(src.Type().Elem())
	if err := copyFields(path, on, table, src.Elem(), copied.Elem()); err != nil {
		return err
	}
	if !copied.Elem().IsZero() {
		dst.Set(copied)
	}
	return nil
}

// isUnset reports whether a rejected field is effectively unset: a nil or
// empty value, or a pointer to a zero value (such as `daemon: false`).
func isUnset(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer:
		return v.IsNil() || isUnset(v.Elem())
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

// joinFieldPath appends a field's JSON name to path. Inlined fields add
// nothing.
func joinFieldPath(path string, sf reflect.StructField) string {
	name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
	switch {
	case name == "":
		return path
	case path == "":
		return name
	default:
		return path + "." + name
	}
}

// poolEnv is the part of a container's env that is the same for every task of
// a pool: the valueFrom entries.
func poolEnv(env []apiv1.EnvVar) []apiv1.EnvVar {
	var out []apiv1.EnvVar
	for _, e := range env {
		if e.ValueFrom != nil {
			out = append(out, e)
		}
	}
	return out
}

// Leaf is a field the tables classify with a class rather than nested fields.
type Leaf struct {
	// Path is the field's Go field names, starting with its table's root
	// type: Template or WorkflowSpec.
	Path []string
	// Class is how the field reaches a pooled task.
	Class Class
	// ResolvedInto, for a hashed field the controller resolves rather than
	// copies, is the dotted Go path of the PoolSpec field it resolves into.
	ResolvedInto string
}

// String is the leaf's dotted path.
func (l Leaf) String() string { return strings.Join(l.Path, ".") }

// Leaves lists every leaf of the template and workflow spec tables, sorted
// by path within each.
func Leaves() []Leaf {
	return append(leaves(templateFields, "Template"), leaves(workflowSpecFields, "WorkflowSpec")...)
}

func leaves(table fieldTable, path ...string) []Leaf {
	var out []Leaf
	for _, name := range slices.Sorted(maps.Keys(table)) {
		f := table[name]
		p := append(slices.Clone(path), name)
		if f.fields != nil {
			out = append(out, leaves(f.fields, p...)...)
			continue
		}
		out = append(out, Leaf{Path: p, Class: f.class, ResolvedInto: f.into})
	}
	return out
}
