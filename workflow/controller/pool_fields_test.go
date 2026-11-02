package controller

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// setPoolField sets the field at path (Go field names) under the struct v
// points at, allocating nil pointers on the way.
func setPoolField(v reflect.Value, path []string, value reflect.Value) {
	for _, name := range path {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.FieldByName(name)
	}
	v.Set(value)
}

// fillValue returns a minimal non-zero value of typ: one element, one entry,
// or the first exported field filled.
func fillValue(typ reflect.Type) reflect.Value {
	switch typ {
	case reflect.TypeFor[resource.Quantity]():
		return reflect.ValueOf(resource.MustParse("1"))
	case reflect.TypeFor[intstr.IntOrString]():
		return reflect.ValueOf(intstr.FromInt32(1))
	case reflect.TypeFor[json.RawMessage]():
		return reflect.ValueOf(json.RawMessage(`{"fill":1}`))
	}
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.String:
		v.SetString("fill")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint8, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Pointer:
		p := reflect.New(typ.Elem())
		p.Elem().Set(fillValue(typ.Elem()))
		v.Set(p)
	case reflect.Slice:
		v.Set(reflect.Append(v, fillValue(typ.Elem())))
	case reflect.Map:
		v.Set(reflect.MakeMap(typ))
		v.SetMapIndex(fillValue(typ.Key()), fillValue(typ.Elem()))
	case reflect.Struct:
		for i := range typ.NumField() {
			if typ.Field(i).IsExported() {
				v.Field(i).Set(fillValue(typ.Field(i).Type))
				break
			}
		}
	default:
		panic(fmt.Sprintf("no fill value for %s", typ))
	}
	return v
}

// TestPoolFieldClassesBehave sets each classified field in turn on a pooled
// task and checks buildPoolSpec does what the field's class says: a hashed
// field changes the hash, a per-task one doesn't, a rejected one fails naming
// it. Resolved and split fields have their own tests.
func TestPoolFieldClassesBehave(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, leaf := range pool.Leaves() {
		if leaf.ResolvedInto != "" || leaf.Class == pool.SplitEnv {
			continue
		}
		t.Run(leaf.String(), func(t *testing.T) {
			in, x := poolTestInputs()
			if leaf.Path[1] == "Script" {
				in.tmpl.Script = &wfv1.ScriptTemplate{Container: *in.tmpl.Container, Source: "base"}
				in.tmpl.Container = nil
			}
			_, baseHash := mustPoolHash(ctx, t, in, x)

			root := reflect.ValueOf(in.tmpl).Elem()
			if leaf.Path[0] == "WorkflowSpec" {
				root = reflect.ValueOf(in.execWfSpec).Elem()
			}
			fieldType := reflect.TypeOf(root.Interface())
			for _, name := range leaf.Path[1:] {
				if fieldType.Kind() == reflect.Pointer {
					fieldType = fieldType.Elem()
				}
				sf, _ := fieldType.FieldByName(name)
				fieldType = sf.Type
			}
			value := fillValue(fieldType)
			if leaf.Path[len(leaf.Path)-1] == "VolumeMounts" {
				value = reflect.ValueOf([]apiv1.VolumeMount{{Name: "base-vol", MountPath: "/fill"}})
			}
			setPoolField(root, leaf.Path[1:], value)

			spec, err := buildPoolSpec(ctx, in, x)
			if leaf.Class == pool.Rejected {
				assert.ErrorContains(t, err, "can't be set on")
				return
			}
			require.NoError(t, err)
			hash, err := poolHash(spec)
			require.NoError(t, err)
			switch leaf.Class {
			case pool.Hashed:
				assert.NotEqual(t, baseHash, hash)
			case pool.PerTask:
				assert.Equal(t, baseHash, hash)
			default:
				t.Fatalf("unexpected class %d", leaf.Class)
			}
		})
	}
}

// TestPoolRejectedFieldsNamedByJSONPath checks a rejected field's error names
// it as a user writes it.
func TestPoolRejectedFieldsNamedByJSONPath(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, tt := range []struct {
		name string
		set  func(in *podBuilderInputs)
		want string
	}{
		{"initContainers", func(in *podBuilderInputs) { in.tmpl.InitContainers = []wfv1.UserContainer{{}} }, "initContainers can't be set"},
		{"container lifecycle", func(in *podBuilderInputs) {
			in.tmpl.Container.Lifecycle = &apiv1.Lifecycle{PreStop: &apiv1.LifecycleHandler{}}
		}, "container.lifecycle can't be set"},
		{"retry affinity", func(in *podBuilderInputs) {
			in.tmpl.RetryStrategy = &wfv1.RetryStrategy{Affinity: &wfv1.RetryAffinity{NodeAntiAffinity: &wfv1.RetryNodeAntiAffinity{}}}
		}, "retryStrategy.affinity can't be set"},
		{"workflow podSpecPatch", func(in *podBuilderInputs) { in.execWfSpec.PodSpecPatch = "{}" }, "spec.podSpecPatch can't be set"},
		{"steps", func(in *podBuilderInputs) { in.tmpl.Container = nil; in.tmpl.Steps = []wfv1.ParallelSteps{{}} }, "steps can't be set"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in, x := poolTestInputs()
			tt.set(in)
			_, err := buildPoolSpec(ctx, in, x)
			assert.ErrorContains(t, err, tt.want)
		})
	}

	t.Run("pointer to false is unset", func(t *testing.T) {
		in, x := poolTestInputs()
		in.tmpl.Daemon = new(false)
		in.tmpl.RetryStrategy = &wfv1.RetryStrategy{Limit: new(intstr.FromInt32(2)), Affinity: &wfv1.RetryAffinity{}}
		_, err := buildPoolSpec(ctx, in, x)
		assert.NoError(t, err)
	})
}

// TestPoolEnvSplit checks that valueFrom and envFrom are pool constants and
// plain value entries are per task.
func TestPoolEnvSplit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	secretEnv := func(key string) apiv1.EnvVar {
		return apiv1.EnvVar{Name: "TOKEN", ValueFrom: &apiv1.EnvVarSource{SecretKeyRef: &apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "s"}, Key: key}}}
	}
	in, x := poolTestInputs()
	in.tmpl.Container.Env = []apiv1.EnvVar{{Name: "A", Value: "1"}, secretEnv("k1"), {Name: "B", Value: "2"}}
	spec, base := mustPoolHash(ctx, t, in, x)
	assert.Equal(t, []apiv1.EnvVar{secretEnv("k1")}, spec.Template.Container.Env)

	in.tmpl.Container.Env = []apiv1.EnvVar{{Name: "A", Value: "changed"}, secretEnv("k1")}
	_, hash := mustPoolHash(ctx, t, in, x)
	assert.Equal(t, base, hash, "plain values are per task")

	in.tmpl.Container.Env = []apiv1.EnvVar{{Name: "A", Value: "1"}, secretEnv("k2")}
	_, hash = mustPoolHash(ctx, t, in, x)
	assert.NotEqual(t, base, hash, "valueFrom is a pool constant")
}

// TestFoldBoundarySchedulingConstraints checks that folding the boundary
// template into the template gives the pod's scheduling constraints, so a
// pool built without a boundary template schedules like the task's pod.
func TestFoldBoundarySchedulingConstraints(t *testing.T) {
	affinity := func(key string) *apiv1.Affinity {
		return &apiv1.Affinity{NodeAffinity: &apiv1.NodeAffinity{PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.PreferredSchedulingTerm{{Weight: 1, Preference: apiv1.NodeSelectorTerm{MatchExpressions: []apiv1.NodeSelectorRequirement{{Key: key}}}}}}}
	}
	own := &wfv1.Template{NodeSelector: map[string]string{"own": "x"}, Affinity: affinity("own"), Tolerations: []apiv1.Toleration{{Key: "own"}}}
	boundary := &wfv1.Template{NodeSelector: map[string]string{"boundary": "x"}, Affinity: affinity("boundary"), Tolerations: []apiv1.Toleration{{Key: "boundary"}}}
	wfSpec := &wfv1.WorkflowSpec{NodeSelector: map[string]string{"wf": "x"}, Affinity: affinity("wf"), Tolerations: []apiv1.Toleration{{Key: "wf"}}}
	for _, tt := range []struct {
		name     string
		tmpl     *wfv1.Template
		boundary *wfv1.Template
	}{
		{"own", own, boundary},
		{"boundary", &wfv1.Template{}, boundary},
		{"workflow", &wfv1.Template{}, &wfv1.Template{}},
		{"no boundary", &wfv1.Template{}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var want, got apiv1.PodSpec
			applySchedulingConstraints(&want, wfSpec, tt.tmpl, tt.boundary)
			folded := tt.tmpl.DeepCopy()
			foldBoundarySchedulingConstraints(folded, tt.boundary)
			applySchedulingConstraints(&got, wfSpec, folded, nil)
			assert.Equal(t, want, got)
		})
	}
}
