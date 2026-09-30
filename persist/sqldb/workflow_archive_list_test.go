//go:build !windows

package sqldb

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// newContainerArchive connects to the database in a started container, runs migrations, and returns a
// WorkflowArchive. dbConfig wraps the connection details in the driver's section of config.DBConfig.
func newContainerArchive(ctx context.Context, t *testing.T, c testcontainers.Container, containerPort string, dbConfig func(config.DatabaseConfig) config.DBConfig) WorkflowArchive {
	t.Helper()

	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(c); termErr != nil {
			t.Logf("failed to terminate container: %s", termErr)
		}
	})

	host, err := c.Host(ctx)
	require.NoError(t, err)
	p, err := c.MappedPort(ctx, containerPort)
	require.NoError(t, err)
	port, err := strconv.Atoi(p.Port())
	require.NoError(t, err)

	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: dbConfig(config.DatabaseConfig{
			Database: "argo",
			Host:     host,
			Port:     port,
		}),
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)

	err = Migrate(ctx, proxy.Session(ctx), "test", "argo_workflows", proxy.DBType())
	require.NoError(t, err)

	t.Cleanup(func() { proxy.Close() })

	return NewWorkflowArchive(proxy, "test", "", instanceid.NewService(""))
}

// testListWorkflowsPaging archives workflows started a minute apart in the "paging" namespace, plus
// some that share a start time in "paging-ties" and one in another namespace, and checks
// ListWorkflows returns them newest first, paged, filtered and fully populated. Workflows are
// archived out of start order so the result order can't come from insertion order.
func testListWorkflowsPaging(ctx context.Context, t *testing.T, archive WorkflowArchive) {
	t.Helper()
	const namespace = "paging"
	const tiesNamespace = "paging-ties"
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, i := range []int{2, 4, 0, 3, 1} {
		started := metav1.NewTime(base.Add(time.Duration(i) * time.Minute))
		wfLabels := map[string]string{"index": strconv.Itoa(i)}
		if i%2 == 0 {
			wfLabels["even"] = "true"
		}
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("wf-%d", i),
				Namespace:         namespace,
				UID:               types.UID(fmt.Sprintf("paging-uid-%d", i)),
				CreationTimestamp: metav1.NewTime(started.Add(-10 * time.Second)),
				Labels:            wfLabels,
				Annotations:       map[string]string{"note": fmt.Sprintf("n%d", i)},
			},
			Spec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{{Name: "i", Value: wfv1.AnyStringPtr(strconv.Itoa(i))}},
				},
			},
			Status: wfv1.WorkflowStatus{
				Phase:             wfv1.WorkflowSucceeded,
				StartedAt:         started,
				FinishedAt:        metav1.NewTime(started.Add(30 * time.Second)),
				Progress:          "1/1",
				Message:           fmt.Sprintf("message %d", i),
				EstimatedDuration: wfv1.EstimatedDuration(i),
				ResourcesDuration: wfv1.ResourcesDuration{"cpu": wfv1.ResourceDuration(i)},
			},
		}
		if i%2 == 1 {
			wf.Spec.Suspend = new(true)
		}
		require.NoError(t, archive.ArchiveWorkflow(ctx, wf))
	}
	// Started at the same second, so only the uid orders them.
	for _, i := range []int{2, 0, 1} {
		require.NoError(t, archive.ArchiveWorkflow(ctx, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("tie-%d", i),
				Namespace: tiesNamespace,
				UID:       types.UID(fmt.Sprintf("tie-uid-%d", i)),
				Labels:    map[string]string{},
			},
			Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded, StartedAt: metav1.NewTime(base), FinishedAt: metav1.NewTime(base)},
		}))
	}
	// Started after everything else, so it would come first if the namespace filter were lost.
	require.NoError(t, archive.ArchiveWorkflow(ctx, &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "paging-other", UID: "paging-other-uid", Labels: map[string]string{}},
		Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded, StartedAt: metav1.NewTime(base.Add(time.Hour)), FinishedAt: metav1.NewTime(base.Add(time.Hour))},
	}))

	names := func(wfs wfv1.Workflows) []string {
		out := make([]string, len(wfs))
		for i, wf := range wfs {
			out[i] = wf.Name
		}
		return out
	}

	even, err := labels.ParseToRequirements("even=true")
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		options sutils.ListOptions
		want    []string
	}{
		{"all", sutils.ListOptions{Namespace: namespace}, []string{"wf-4", "wf-3", "wf-2", "wf-1", "wf-0"}},
		{"first page", sutils.ListOptions{Namespace: namespace, Limit: 2}, []string{"wf-4", "wf-3"}},
		{"second page", sutils.ListOptions{Namespace: namespace, Limit: 2, Offset: 2}, []string{"wf-2", "wf-1"}},
		{"last page", sutils.ListOptions{Namespace: namespace, Limit: 2, Offset: 4}, []string{"wf-0"}},
		{"labels", sutils.ListOptions{Namespace: namespace, LabelRequirements: even}, []string{"wf-4", "wf-2", "wf-0"}},
		{"labels paged", sutils.ListOptions{Namespace: namespace, LabelRequirements: even, Limit: 1, Offset: 1}, []string{"wf-2"}},
		{"ties", sutils.ListOptions{Namespace: tiesNamespace}, []string{"tie-0", "tie-1", "tie-2"}},
		{"ties first page", sutils.ListOptions{Namespace: tiesNamespace, Limit: 2}, []string{"tie-0", "tie-1"}},
		{"ties second page", sutils.ListOptions{Namespace: tiesNamespace, Limit: 2, Offset: 2}, []string{"tie-2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wfs, err := archive.ListWorkflows(ctx, tc.options)
			require.NoError(t, err)
			assert.Equal(t, tc.want, names(wfs))
		})
	}

	t.Run("fields", func(t *testing.T) {
		wfs, err := archive.ListWorkflows(ctx, sutils.ListOptions{Namespace: namespace, Limit: 2, Offset: 1})
		require.NoError(t, err)
		require.Len(t, wfs, 2)
		wf := wfs[0]
		assert.Equal(t, "wf-3", wf.Name)
		assert.Equal(t, namespace, wf.Namespace)
		assert.Equal(t, types.UID("paging-uid-3"), wf.UID)
		assert.Equal(t, "3", wf.Labels["index"])
		assert.Equal(t, "Persisted", wf.Labels[common.LabelKeyWorkflowArchivingStatus])
		assert.Equal(t, "n3", wf.Annotations["note"])
		assert.Equal(t, new(true), wf.Spec.Suspend)
		assert.Equal(t, "3", wf.Spec.Arguments.Parameters[0].Value.String())
		assert.Equal(t, wfv1.WorkflowSucceeded, wf.Status.Phase)
		assert.True(t, base.Add(3*time.Minute).Equal(wf.Status.StartedAt.Time))
		assert.True(t, base.Add(3*time.Minute+30*time.Second).Equal(wf.Status.FinishedAt.Time))
		assert.True(t, base.Add(3*time.Minute-10*time.Second).Equal(wf.CreationTimestamp.Time))
		assert.Equal(t, wfv1.Progress("1/1"), wf.Status.Progress)
		assert.Equal(t, "message 3", wf.Status.Message)
		assert.Equal(t, wfv1.EstimatedDuration(3), wf.Status.EstimatedDuration)
		assert.Equal(t, wfv1.ResourcesDuration{"cpu": wfv1.ResourceDuration(3)}, wf.Status.ResourcesDuration)
		assert.Equal(t, "wf-2", wfs[1].Name)
		assert.Nil(t, wfs[1].Spec.Suspend)
	})
}
