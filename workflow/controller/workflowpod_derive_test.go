package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

var deriveTestNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func TestTaskDeadline(t *testing.T) {
	wfDeadline := deriveTestNow.Add(time.Hour)
	earlier := deriveTestNow.Add(time.Minute)
	later := deriveTestNow.Add(2 * time.Hour)
	tests := []struct {
		name              string
		workflowDeadline  *time.Time
		executionDeadline time.Time
		onExit            bool
		want              time.Time
	}{
		{name: "none", want: time.Time{}},
		{name: "workflow deadline", workflowDeadline: &wfDeadline, want: wfDeadline},
		{name: "exit handler ignores workflow deadline", workflowDeadline: &wfDeadline, onExit: true, want: time.Time{}},
		{name: "execution deadline alone", executionDeadline: earlier, want: earlier},
		{name: "earlier execution deadline wins", workflowDeadline: &wfDeadline, executionDeadline: earlier, want: earlier},
		{name: "earlier workflow deadline wins", workflowDeadline: &wfDeadline, executionDeadline: later, want: wfDeadline},
		{name: "exit handler keeps execution deadline", workflowDeadline: &wfDeadline, executionDeadline: later, onExit: true, want: later},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, taskDeadline(tt.workflowDeadline, tt.executionDeadline, tt.onExit))
		})
	}
}

func TestTaskActiveDeadlineSeconds(t *testing.T) {
	inAMinute := deriveTestNow.Add(time.Minute)
	passed := deriveTestNow.Add(-time.Second)
	tests := []struct {
		name             string
		tmplADS          string
		workflowDeadline *time.Time
		onExit           bool
		want             *int64
		wantStart        bool
		wantErr          bool
	}{
		{name: "none", wantStart: true},
		{name: "template only", tmplADS: "30", want: new(int64(30)), wantStart: true},
		{name: "workflow deadline only", workflowDeadline: &inAMinute, want: new(int64(60)), wantStart: true},
		{name: "workflow deadline lowers template", tmplADS: "120", workflowDeadline: &inAMinute, want: new(int64(60)), wantStart: true},
		{name: "template below workflow deadline", tmplADS: "30", workflowDeadline: &inAMinute, want: new(int64(30)), wantStart: true},
		{name: "workflow deadline passed", tmplADS: "30", workflowDeadline: &passed},
		{name: "exit handler ignores passed workflow deadline", tmplADS: "30", workflowDeadline: &passed, onExit: true, want: new(int64(30)), wantStart: true},
		{name: "invalid template value", tmplADS: "not-a-number", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &wfv1.Template{}
			if tt.tmplADS != "" {
				tmpl.ActiveDeadlineSeconds = intstr.ParsePtr(tt.tmplADS)
			}
			got, start, err := taskActiveDeadlineSeconds(tmpl, tt.workflowDeadline, tt.onExit, deriveTestNow)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateTimeoutDeadlines(t *testing.T) {
	startedAt := metav1.NewTime(deriveTestNow.Add(-10 * time.Second))
	tmpl := &wfv1.Template{Timeout: "60s", PendingTimeout: "30s"}

	t.Run("no node", func(t *testing.T) {
		deadline, pendingDeadline, err := templateTimeoutDeadlines(tmpl, nil, deriveTestNow)
		require.NoError(t, err)
		assert.Nil(t, deadline)
		assert.Nil(t, pendingDeadline)
	})
	t.Run("pending node gets both", func(t *testing.T) {
		node := &wfv1.NodeStatus{Phase: wfv1.NodePending, StartedAt: startedAt}
		deadline, pendingDeadline, err := templateTimeoutDeadlines(tmpl, node, deriveTestNow)
		require.NoError(t, err)
		assert.Equal(t, startedAt.Add(60*time.Second), *deadline)
		assert.Equal(t, startedAt.Add(30*time.Second), *pendingDeadline)
	})
	t.Run("running node gets no pending deadline", func(t *testing.T) {
		node := &wfv1.NodeStatus{Phase: wfv1.NodeRunning, StartedAt: startedAt}
		deadline, pendingDeadline, err := templateTimeoutDeadlines(tmpl, node, deriveTestNow)
		require.NoError(t, err)
		assert.Equal(t, startedAt.Add(60*time.Second), *deadline)
		assert.Nil(t, pendingDeadline)
	})
	t.Run("pending past pending timeout", func(t *testing.T) {
		node := &wfv1.NodeStatus{Phase: wfv1.NodePending, StartedAt: startedAt}
		_, _, err := templateTimeoutDeadlines(tmpl, node, deriveTestNow.Add(time.Minute))
		require.ErrorIs(t, err, ErrTimeout)
	})
	t.Run("running past timeout still returns the deadline", func(t *testing.T) {
		node := &wfv1.NodeStatus{Phase: wfv1.NodeRunning, StartedAt: startedAt}
		deadline, _, err := templateTimeoutDeadlines(tmpl, node, deriveTestNow.Add(time.Hour))
		require.NoError(t, err)
		assert.Equal(t, startedAt.Add(60*time.Second), *deadline)
	})
}

func TestApplyTemplateDeadline(t *testing.T) {
	inAMinute := deriveTestNow.Add(time.Minute)
	inASecond := deriveTestNow.Add(time.Second)
	ads := new(int64(30))

	got, err := applyTemplateDeadline("node", ads, nil, deriveTestNow)
	require.NoError(t, err)
	assert.Same(t, ads, got, "no template deadline leaves activeDeadlineSeconds alone")

	got, err = applyTemplateDeadline("node", nil, &inAMinute, deriveTestNow)
	require.NoError(t, err)
	assert.Equal(t, new(int64(60)), got)

	got, err = applyTemplateDeadline("node", ads, &inAMinute, deriveTestNow)
	require.NoError(t, err)
	assert.Equal(t, new(int64(60)), got, "pods have always replaced activeDeadlineSeconds with the template deadline")

	_, err = applyTemplateDeadline("node", ads, &inASecond, deriveTestNow)
	require.EqualError(t, err, "node exceeded its deadline")
}

// deadlineStubDeps is the build-test stub with a node and a template timeout
// deadline, so build applies the template deadline.
type deadlineStubDeps struct {
	*stubPodBuilderDeps
	templateDeadline *time.Time
}

func (d *deadlineStubDeps) getNodeByName(_ string) (*wfv1.NodeStatus, error) {
	return &wfv1.NodeStatus{}, nil
}

func (d *deadlineStubDeps) checkTemplateTimeouts(_ *wfv1.Template, _ *wfv1.NodeStatus, _ time.Time) (deadline, pendingDeadline *time.Time, err error) {
	return d.templateDeadline, nil, nil
}

// TestBuild_DeadlinesMatchDerivation checks that a pod's activeDeadlineSeconds
// and ARGO_DEADLINE are what the extracted deadline functions compute, so
// anything else using them bounds a task as a pod would be.
func TestBuild_DeadlinesMatchDerivation(t *testing.T) {
	wfDeadline := deriveTestNow.Add(10 * time.Minute)
	passed := deriveTestNow.Add(-time.Second)
	executionDeadline := deriveTestNow.Add(5 * time.Minute)
	templateDeadline := deriveTestNow.Add(2 * time.Minute)
	tests := []struct {
		name              string
		tmplADS           string
		workflowDeadline  *time.Time
		executionDeadline time.Time
		templateDeadline  *time.Time
		onExit            bool
		debugPause        bool
		wantNoPod         bool
		wantADS           *int64
		wantDeadline      time.Time
	}{
		{name: "no deadlines"},
		{name: "template activeDeadlineSeconds", tmplADS: "30", wantADS: new(int64(30))},
		{name: "workflow deadline", workflowDeadline: &wfDeadline, wantADS: new(int64(600)), wantDeadline: wfDeadline},
		{name: "workflow deadline passed", workflowDeadline: &passed, wantNoPod: true},
		{name: "exit handler", tmplADS: "30", workflowDeadline: &passed, onExit: true, wantADS: new(int64(30))},
		{name: "execution deadline", workflowDeadline: &wfDeadline, executionDeadline: executionDeadline, wantADS: new(int64(600)), wantDeadline: executionDeadline},
		{name: "template timeout", tmplADS: "300", workflowDeadline: &wfDeadline, templateDeadline: &templateDeadline, wantADS: new(int64(120)), wantDeadline: wfDeadline},
		{name: "debug pause clears activeDeadlineSeconds", tmplADS: "30", debugPause: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			tmpl := simpleContainerTemplate()
			if tt.tmplADS != "" {
				tmpl.ActiveDeadlineSeconds = intstr.ParsePtr(tt.tmplADS)
			}
			main := tmpl.Container.DeepCopy()
			if tt.debugPause {
				main.Env = append(main.Env, apiv1.EnvVar{Name: "ARGO_DEBUG_PAUSE_BEFORE", Value: "true"})
			}
			deps := &deadlineStubDeps{stubPodBuilderDeps: newStubDeps(), templateDeadline: tt.templateDeadline}
			pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, false, deps)
			pb.in.now = deriveTestNow
			pb.in.workflowDeadline = tt.workflowDeadline
			pb.in.computedWorkflowDeadline = tt.workflowDeadline
			pb.in.opts = &createWorkflowPodOpts{onExitPod: tt.onExit, executionDeadline: tt.executionDeadline}

			result, err := pb.build(ctx)
			require.NoError(t, err)

			ads, start, err := taskActiveDeadlineSeconds(tmpl, tt.workflowDeadline, tt.onExit, deriveTestNow)
			require.NoError(t, err)
			if tt.wantNoPod {
				assert.Nil(t, result)
				assert.False(t, start)
				return
			}
			require.NotNil(t, result)
			if tt.debugPause {
				ads = nil
			}
			ads, err = applyTemplateDeadline("node", ads, tt.templateDeadline, deriveTestNow)
			require.NoError(t, err)

			assert.Equal(t, tt.wantADS, result.Pod.Spec.ActiveDeadlineSeconds)
			assert.Equal(t, ads, result.Pod.Spec.ActiveDeadlineSeconds)

			deadline := taskDeadline(tt.workflowDeadline, tt.executionDeadline, tt.onExit)
			assert.Equal(t, tt.wantDeadline, deadline)
			mainCtr := containerByName(result.Pod.Spec.Containers, common.MainContainerName)
			require.NotNil(t, mainCtr)
			got, ok := envValue(mainCtr, common.EnvVarDeadline)
			require.True(t, ok)
			assert.Equal(t, deadline.Format(time.RFC3339), got)
		})
	}
}

func TestApplyMainContainerDefaults(t *testing.T) {
	defaults := &apiv1.Container{
		Name: "main",
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{apiv1.ResourceCPU: resource.MustParse("1")},
		},
		Env: []apiv1.EnvVar{{Name: "FROM_DEFAULTS", Value: "yes"}},
	}

	t.Run("container template", func(t *testing.T) {
		ctrs := []apiv1.Container{{Name: "ignored", Image: "img", Env: []apiv1.EnvVar{{Name: "FROM_TMPL", Value: "yes"}}}}
		require.NoError(t, applyMainContainerDefaults(ctrs, &wfv1.Template{Container: &apiv1.Container{}}, defaults))
		assert.Equal(t, common.MainContainerName, ctrs[0].Name)
		assert.Equal(t, "img", ctrs[0].Image)
		assert.Equal(t, resource.MustParse("1"), ctrs[0].Resources.Limits[apiv1.ResourceCPU])
		assert.ElementsMatch(t, []apiv1.EnvVar{{Name: "FROM_DEFAULTS", Value: "yes"}, {Name: "FROM_TMPL", Value: "yes"}}, ctrs[0].Env)
	})
	t.Run("container set members keep their names", func(t *testing.T) {
		ctrs := []apiv1.Container{{Name: "a"}, {}}
		tmpl := &wfv1.Template{ContainerSet: &wfv1.ContainerSetTemplate{}}
		require.NoError(t, applyMainContainerDefaults(ctrs, tmpl, nil))
		assert.Equal(t, "a", ctrs[0].Name)
		assert.Equal(t, common.MainContainerName, ctrs[1].Name)
	})
}

func TestDerivePodServiceAccount(t *testing.T) {
	falseVal := false
	trueVal := true
	tests := []struct {
		name    string
		tmpl    wfv1.Template
		wfSpec  wfv1.WorkflowSpec
		want    podServiceAccount
		wantErr string
	}{
		{name: "none"},
		{
			name:   "template wins over workflow",
			tmpl:   wfv1.Template{ServiceAccountName: "tmpl-sa", Executor: &wfv1.ExecutorConfig{ServiceAccountName: "tmpl-exec"}},
			wfSpec: wfv1.WorkflowSpec{ServiceAccountName: "wf-sa", Executor: &wfv1.ExecutorConfig{ServiceAccountName: "wf-exec"}},
			want:   podServiceAccount{serviceAccountName: "tmpl-sa", executorServiceAccountName: "tmpl-exec"},
		},
		{
			name:   "workflow fallback",
			wfSpec: wfv1.WorkflowSpec{ServiceAccountName: "wf-sa", Executor: &wfv1.ExecutorConfig{ServiceAccountName: "wf-exec"}},
			want:   podServiceAccount{serviceAccountName: "wf-sa", executorServiceAccountName: "wf-exec"},
		},
		{
			name:   "automount true is left to the default",
			wfSpec: wfv1.WorkflowSpec{AutomountServiceAccountToken: &trueVal},
		},
		{
			name:   "automount false with executor account",
			wfSpec: wfv1.WorkflowSpec{AutomountServiceAccountToken: &falseVal, Executor: &wfv1.ExecutorConfig{ServiceAccountName: "wf-exec"}},
			want:   podServiceAccount{automountServiceAccountToken: &falseVal, executorServiceAccountName: "wf-exec"},
		},
		{
			name:    "automount false needs an executor account",
			tmpl:    wfv1.Template{AutomountServiceAccountToken: &falseVal},
			wfSpec:  wfv1.WorkflowSpec{AutomountServiceAccountToken: &trueVal},
			wantErr: "executor.serviceAccountName must not be empty if automountServiceAccountToken is false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := derivePodServiceAccount(&tt.tmpl, &tt.wfSpec)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildExecContainer(t *testing.T) {
	cfg := execContainerConfig{
		image:           "argoexec:test",
		imagePullPolicy: apiv1.PullAlways,
		executor:        &apiv1.Container{Env: []apiv1.EnvVar{{Name: "EXTRA", Value: "x"}}},
		kubeConfig:      &config.KubeConfig{SecretName: "kc", SecretKey: "kubeconfig"},
		instanceID:      "instance",
		workflowUID:     "wf-uid",
	}
	ctr := buildExecContainer(common.WaitContainerName, &wfv1.Template{Container: &apiv1.Container{}}, cfg, "exec-sa")
	assert.Equal(t, "argoexec:test", ctr.Image)
	assert.Equal(t, apiv1.PullAlways, ctr.ImagePullPolicy)
	assert.Equal(t, []string{"--kubeconfig=" + common.KubeConfigDefaultMountPath}, ctr.Args)
	uid, _ := envValue(ctr, common.EnvVarWorkflowUID)
	assert.Equal(t, "wf-uid", uid)
	instance, _ := envValue(ctr, common.EnvVarInstanceID)
	assert.Equal(t, "instance", instance)
	extra, _ := envValue(ctr, "EXTRA")
	assert.Equal(t, "x", extra)
	assert.True(t, hasMount(ctr, common.KubeConfigDefaultVolumeName))
	assert.True(t, hasMount(ctr, common.ServiceAccountTokenVolumeName))

	resourceCtr := buildExecContainer(common.MainContainerName, &wfv1.Template{Resource: &wfv1.ResourceTemplate{}}, execContainerConfig{executor: &apiv1.Container{}}, "")
	require.NotNil(t, resourceCtr.SecurityContext)
	assert.Equal(t, new(true), resourceCtr.SecurityContext.ReadOnlyRootFilesystem)
	assert.False(t, hasMount(resourceCtr, common.ServiceAccountTokenVolumeName))
}

func TestApplySchedulingConstraints(t *testing.T) {
	wfSpec := &wfv1.WorkflowSpec{NodeSelector: map[string]string{"from": "wf"}, SchedulerName: "wf-scheduler", PodPriorityClassName: "wf-priority"}
	boundary := &wfv1.Template{NodeSelector: map[string]string{"from": "boundary"}}
	tmpl := &wfv1.Template{PriorityClassName: "tmpl-priority"}

	var spec apiv1.PodSpec
	applySchedulingConstraints(&spec, wfSpec, tmpl, boundary)
	assert.Equal(t, map[string]string{"from": "boundary"}, spec.NodeSelector)
	assert.Equal(t, "wf-scheduler", spec.SchedulerName)
	assert.Equal(t, "tmpl-priority", spec.PriorityClassName)

	spec = apiv1.PodSpec{}
	applySchedulingConstraints(&spec, wfSpec, tmpl, nil)
	assert.Equal(t, map[string]string{"from": "wf"}, spec.NodeSelector)
}
