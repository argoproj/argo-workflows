package v1alpha1

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// protoMessage is what the generated protobuf code gives every API type.
type protoMessage interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

func protoRoundTrip[T any, P interface {
	*T
	protoMessage
}](t *testing.T, in P) P {
	t.Helper()
	data, err := in.Marshal()
	require.NoError(t, err)
	out := P(new(T))
	require.NoError(t, out.Unmarshal(data))
	return out
}

func jsonRoundTrip[T any](t *testing.T, in *T) (*T, map[string]any) {
	t.Helper()
	data, err := json.Marshal(in)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(data, &fields))
	out := new(T)
	require.NoError(t, json.Unmarshal(data, out))
	return out, fields
}

// TestUnpooledObjectsRoundTripUnchanged checks that objects without the pool
// fields serialise without them and come back unchanged.
func TestUnpooledObjectsRoundTripUnchanged(t *testing.T) {
	tmpl := &Template{Name: "main", Container: &apiv1.Container{Image: "argoproj/argosay:v2"}}
	gotTmpl, fields := jsonRoundTrip(t, tmpl)
	assert.NotContains(t, fields, "pool")
	assert.Equal(t, tmpl, gotTmpl)
	assert.Equal(t, tmpl, protoRoundTrip(t, tmpl))

	node := &NodeStatus{ID: "wf-123", Name: "wf", Type: NodeTypePod, Phase: NodeRunning}
	gotNode, fields := jsonRoundTrip(t, node)
	assert.NotContains(t, fields, "pooledTask")
	assert.Equal(t, node, gotNode)
	assert.Equal(t, node, protoRoundTrip(t, node))

	spec := &WorkflowTaskSetSpec{Tasks: map[string]Template{"wf-123": *tmpl}}
	gotSpec, fields := jsonRoundTrip(t, spec)
	assert.NotContains(t, fields, "pool")
	assert.NotContains(t, fields, "dispatch")
	assert.Equal(t, spec, gotSpec)
	assert.Equal(t, spec, protoRoundTrip(t, spec))
}

// TestPoolFieldsAreTagged checks every field of the pool types has a JSON
// and a protobuf tag. encoding/json falls back to the Go field name without
// one, so a round trip alone wouldn't notice.
func TestPoolFieldsAreTagged(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[Pool](), reflect.TypeFor[PoolSpec](), reflect.TypeFor[PoolArtifactPlugin](),
		reflect.TypeFor[PoolKubeConfig](), reflect.TypeFor[TaskDispatch](), reflect.TypeFor[PooledTaskStatus](),
	} {
		for field := range typ.Fields() {
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			assert.NotEmpty(t, name, "%s.%s has no json tag", typ.Name(), field.Name)
			assert.NotEqual(t, "-", name, "%s.%s isn't serialised", typ.Name(), field.Name)
			assert.NotEmpty(t, field.Tag.Get("protobuf"), "%s.%s has no protobuf tag", typ.Name(), field.Name)
		}
	}
}

// TestPooledObjectsRoundTrip checks every pool field, each set to a value,
// survives JSON and protobuf, so none is missing generated code.
func TestPooledObjectsRoundTrip(t *testing.T) {
	// metav1.Time unmarshals into time.Local, so build the times there.
	now := metav1.Unix(time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC).Unix(), 0)
	later := metav1.NewTime(now.Add(time.Hour))

	tmpl := &Template{
		Name:      "main",
		Container: &apiv1.Container{Image: "argoproj/argosay:v2"},
		Pool:      &Pool{Name: "my-pool", InitialReplicas: new(intstr.FromInt32(2)), RetentionSeconds: new(intstr.FromString("{{inputs.parameters.retention}}"))},
	}
	gotTmpl, _ := jsonRoundTrip(t, tmpl)
	assert.Equal(t, tmpl, gotTmpl)
	assert.Equal(t, tmpl, protoRoundTrip(t, tmpl))

	node := &NodeStatus{
		ID:    "wf-123",
		Name:  "wf",
		Type:  NodeTypePod,
		Phase: NodeRunning,
		PooledTask: &PooledTaskStatus{
			PoolName:              "my-pool",
			PoolHash:              "abc123",
			TaskName:              "wf-main-123",
			TaskSetUID:            "taskset-uid",
			Deadline:              &later,
			ActiveDeadlineSeconds: new(int64(30)),
		},
	}
	node.PooledTask.Enqueue(now)
	node.PooledTask.Claim("old-claim-uid", now, "my-pool-abc123-old", "old-pod-uid", 0)
	node.PooledTask.Release(now)
	node.PooledTask.Claim("claim-uid", now, "my-pool-abc123-xyz", "worker-pod-uid", 1)
	node.PooledTask.Interrupt(later)
	gotNode, _ := jsonRoundTrip(t, node)
	assert.Equal(t, node, gotNode)
	assert.Equal(t, node, protoRoundTrip(t, node))

	spec := &WorkflowTaskSetSpec{
		Tasks: map[string]Template{"wf-123": *tmpl},
		Pool: &PoolSpec{
			Name:                                  "my-pool",
			InitialReplicas:                       2,
			RetentionSeconds:                      600,
			Template:                              &Template{Container: &apiv1.Container{Image: "argoproj/argosay:v2"}},
			WorkflowSpec:                          &WorkflowSpec{NodeSelector: map[string]string{"pool": "yes"}},
			ServiceAccountName:                    "runner",
			AutomountServiceAccountToken:          new(false),
			ExecutorServiceAccountTokenSecretName: "executor-token",
			ArtifactRepository:                    &ArtifactRepository{S3: &S3ArtifactRepository{KeyFormat: "{{workflow.name}}/{{node.taskName}}"}},
			ArtifactSecrets:                       []apiv1.SecretKeySelector{{LocalObjectReference: apiv1.LocalObjectReference{Name: "s3"}, Key: "accessKey"}},
			ArtifactPlugins:                       []PoolArtifactPlugin{{Name: "my-plugin", Image: "plugin:v1", ConnectionTimeoutSeconds: 5, Command: []string{"/plugin", "serve"}}},
			InstanceID:                            "instance",
			ControllerVersion:                     "v4.2.0",
			Executor:                              &apiv1.Container{Args: []string{"--x"}},
			MainContainer:                         &apiv1.Container{Env: []apiv1.EnvVar{{Name: "MAIN_DEFAULT", Value: "yes"}}},
			ExecutorImage:                         "quay.io/argoproj/argoexec:latest",
			ExecutorImagePullPolicy:               apiv1.PullIfNotPresent,
			ExecutorLogArgs:                       []string{"--loglevel", "info"},
			KubeConfig:                            &PoolKubeConfig{SecretName: "kubeconfig", SecretKey: "config", VolumeName: "kubeconfig", MountPath: "/kube/config"},
			InitlessPod:                           true,
			TerminationGracePeriodSeconds:         30,
			ReleaseReserveSeconds:                 10,
		},
		Dispatch: &TaskDispatch{
			TaskName:                  "wf-main-123",
			IncludeScriptOutput:       true,
			InputArtifactPlugins:      []ArtifactPluginName{"in-plugin"},
			Command:                   []string{"/bin/sh"},
			Deadline:                  &later,
			ActiveDeadlineSeconds:     new(int64(30)),
			TraceContext:              map[string]string{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
			ProgressPatchTickDuration: &metav1.Duration{Duration: time.Minute},
			ProgressFileTickDuration:  &metav1.Duration{Duration: 3 * time.Second},
		},
	}
	gotSpec, _ := jsonRoundTrip(t, spec)
	assert.Equal(t, spec, gotSpec)
	assert.Equal(t, spec, protoRoundTrip(t, spec))
}

// TestPooledTaskConditions checks that the state is derived from the
// conditions, which change only through the transitions, and that an
// interrupted task never goes back to claimed or enqueued.
func TestPooledTaskConditions(t *testing.T) {
	enqueuedAt := metav1.Unix(100, 0)
	claimedAt := metav1.Unix(200, 0)
	releasedAt := metav1.Unix(300, 0)
	reclaimedAt := metav1.Unix(400, 0)
	interruptedAt := metav1.Unix(500, 0)
	task := &PooledTaskStatus{TaskName: "wf-main-123"}

	assert.Equal(t, PooledTaskEnqueued, task.State(), "a ledger with no conditions is enqueued")
	require.True(t, task.Enqueue(enqueuedAt))
	assert.False(t, task.Enqueue(claimedAt), "only a new ledger can be enqueued")
	assert.Equal(t, enqueuedAt, task.EnqueuedAt)
	assert.Equal(t, PooledTaskEnqueued, task.State())
	assert.Equal(t, enqueuedAt, task.EnqueuedAt)
	assert.Nil(t, task.ClaimedAt())
	assert.False(t, task.Released())
	assert.False(t, task.Release(releasedAt), "only a claimed task can be released")
	assert.False(t, task.Interrupt(interruptedAt), "only a claimed task can be interrupted")

	require.True(t, task.Claim("claim-1", claimedAt, "pod-1", "pod-uid-1", 2))
	assert.Equal(t, PooledTaskClaimed, task.State())
	assert.Equal(t, &claimedAt, task.ClaimedAt())
	assert.Equal(t, "pod-1", task.WorkerPodName)
	assert.Equal(t, int32(2), task.WorkerRestartCount)
	claimed := task.DeepCopy()
	assert.False(t, task.Enqueue(releasedAt), "a claimed task can't be enqueued")
	assert.Equal(t, claimed, task, "and keeps its claimant")

	require.True(t, task.Release(releasedAt))
	assert.Equal(t, PooledTaskEnqueued, task.State())
	assert.True(t, task.Released())
	assert.Nil(t, task.ClaimedAt())
	assert.Equal(t, int32(1), task.Releases)
	assert.Equal(t, PooledTaskStatus{TaskName: "wf-main-123", EnqueuedAt: enqueuedAt, Releases: 1, Conditions: task.Conditions}, *task,
		"a release clears the claimant")
	assert.Equal(t, metav1.Condition{Type: PooledTaskConditionClaimed, Status: metav1.ConditionFalse, Reason: PooledTaskReasonReleased, Message: "pod-1", LastTransitionTime: releasedAt},
		task.Conditions[0])
	assert.Equal(t, enqueuedAt, task.EnqueuedAt, "a release keeps the enqueue time")

	require.True(t, task.Claim("claim-2", reclaimedAt, "pod-2", "pod-uid-2", 0))
	require.True(t, task.Claim("claim-3", interruptedAt, "pod-3", "pod-uid-3", 0), "a claim replaces a claim")
	assert.Equal(t, &interruptedAt, task.ClaimedAt(), "and records the new claim's time")
	assert.False(t, task.Released())

	require.True(t, task.Interrupt(interruptedAt))
	assert.Equal(t, PooledTaskInterrupted, task.State())
	assert.Equal(t, &interruptedAt, task.InterruptedAt())
	assert.False(t, task.Claim("claim-4", interruptedAt, "pod-4", "pod-uid-4", 0))
	assert.False(t, task.Release(interruptedAt))
	assert.Equal(t, PooledTaskInterrupted, task.State())
	assert.Equal(t, "pod-3", task.WorkerPodName)
	assert.Len(t, task.Conditions, 2)
	interrupted := task.DeepCopy()
	assert.False(t, task.Enqueue(reclaimedAt), "an interrupted task can't be enqueued")
	assert.Equal(t, interrupted, task, "and keeps its enqueue time")
}
