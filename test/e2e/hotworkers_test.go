//go:build hotworkers

package e2e

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// HotWorkersSuite runs pooled templates on a controller with hot workers
// enabled (the hotworkers profile). A test whose tasks run uses its own pool
// names, so tests never share a worker; pools outlive their workflows until
// their retention ends.
type HotWorkersSuite struct {
	fixtures.E2ESuite
}

// The default artifact repository's bucket.
const hotWorkersBucket = "my-bucket"

// poolPods lists the pods of every pool named poolName.
func (s *HotWorkersSuite) poolPods(poolName string) []apiv1.Pod {
	s.T().Helper()
	ctx := logging.TestContext(s.T().Context())
	list, err := s.KubeClient.CoreV1().Pods(fixtures.Namespace).List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyPoolName + "=" + poolName})
	s.CheckError(err)
	return list.Items
}

// pooledNodes returns the workflow's pooled nodes, checking each is pooled in
// poolName and was run by a worker.
func pooledNodes(t *testing.T, status *wfv1.WorkflowStatus, poolName string) []wfv1.NodeStatus {
	t.Helper()
	nodes := podNodes(status)
	for _, n := range nodes {
		require.NotNil(t, n.PooledTask, "node %s is not pooled", n.DisplayName)
		assert.Equal(t, poolName, n.PooledTask.PoolName)
		assert.NotEmpty(t, n.PooledTask.TaskName)
		assert.NotEmpty(t, n.PooledTask.WorkerPodUID, "node %s has no worker pod", n.DisplayName)
	}
	return nodes
}

// artifactKey returns the repository key of the node's output artifact.
func artifactKey(t *testing.T, n wfv1.NodeStatus, name string) string {
	t.Helper()
	a := n.Outputs.GetArtifactByName(name)
	require.NotNil(t, a, "node %s has no output artifact %s", n.DisplayName, name)
	key, err := a.GetKey()
	require.NoError(t, err)
	return key
}

// TestPoolReuse runs a pooled workflow twice: the second run is served by the
// pool the first one created, without any new pod, and each run gets its own
// outputs and archived logs.
func (s *HotWorkersSuite) TestPoolReuse() {
	const poolName = "e2e-reuse"

	run := func() map[types.UID]string {
		workers := map[types.UID]string{}
		var name string
		s.Given().
			Workflow("@testdata/hotworkers/pool-reuse.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				name = md.Name
				nodes := pooledNodes(t, status, poolName)
				require.Len(t, nodes, 2)
				for _, n := range nodes {
					workers[n.PooledTask.WorkerPodUID] = n.PooledTask.WorkerPodName
				}
				say := node(t, status, "say")
				require.NotNil(t, say.Outputs)
				require.Len(t, say.Outputs.Parameters, 1)
				assert.Equal(t, md.Name, say.Outputs.Parameters[0].Value.String())
				assert.Contains(t, fixtures.ArtifactContent(t, hotWorkersBucket, artifactKey(t, *say, "message")), md.Name)
				assert.Contains(t, fixtures.ArtifactContent(t, hotWorkersBucket, artifactKey(t, *say, "main-logs")), ":) Hello from "+md.Name)
			}).
			ExpectPods(func(t *testing.T, pods []apiv1.Pod) {
				assert.Empty(t, pods, "a pooled workflow has no pods of its own")
			})
		s.Require().NotEmpty(name)
		return workers
	}

	first := run()
	before := s.poolPods(poolName)
	s.Require().NotEmpty(before)
	for uid, name := range first {
		s.True(hasPod(before, uid), "worker %s isn't a pod of the pool", name)
	}
	// The pods follow the layout under test: the legacy layout's init
	// container installs argoexec, the init-less one has none.
	initless := fixtures.AuxContainerName() == common.SupervisorContainerName
	for _, p := range before {
		var inits, names []string
		for _, c := range p.Spec.InitContainers {
			inits = append(inits, c.Name)
		}
		for _, c := range p.Spec.Containers {
			names = append(names, c.Name)
		}
		s.Equal(!initless, slices.Contains(inits, common.InitContainerName), "pool pod %s init containers %v", p.Name, inits)
		s.Subset(names, []string{common.WorkerContainerName, common.MainContainerName})
		s.NotContains(p.Labels, common.LabelKeyWorkflow)
	}

	second := run()
	after := s.poolPods(poolName)
	s.ElementsMatch(podUIDs(before), podUIDs(after), "the second run created or replaced pool pods")
	for uid, name := range second {
		s.True(hasPod(before, uid), "worker %s of the second run isn't a pod from the first", name)
	}
}

func podUIDs(pods []apiv1.Pod) []types.UID {
	var uids []types.UID
	for _, p := range pods {
		uids = append(uids, p.UID)
	}
	return uids
}

func hasPod(pods []apiv1.Pod, uid types.UID) bool {
	return slices.ContainsFunc(pods, func(p apiv1.Pod) bool { return p.UID == uid })
}

// TestHTTPInputArtifactWithCredentials stages an HTTP input artifact that
// needs basic auth credentials from a Secret in a pooled task.
func (s *HotWorkersSuite) TestHTTPInputArtifactWithCredentials() {
	s.Given().
		Workflow("@testdata/hotworkers/http-artifact.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			nodes := pooledNodes(t, status, "e2e-http")
			require.Len(t, nodes, 1)
			assert.Contains(t, fixtures.ArtifactContent(t, hotWorkersBucket, artifactKey(t, nodes[0], "main-logs")), `"authorized": true`)
		})
}

// TestFanOut runs a pooled fan-out on the default artifact repository: each
// task archives its output artifact and logs under its own key, named after
// its task name.
func (s *HotWorkersSuite) TestFanOut() {
	s.Given().
		Workflow("@testdata/hotworkers/fan-out.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			nodes := pooledNodes(t, status, "e2e-fan-out")
			require.Len(t, nodes, 4)
			keys := map[string]string{}
			for _, n := range nodes {
				require.NotNil(t, n.Inputs)
				require.Len(t, n.Inputs.Parameters, 1)
				i := n.Inputs.Parameters[0].Value.String()
				prefix := md.Name + "/" + n.PooledTask.TaskName + "/"
				for name, want := range map[string]string{"out": "output of task " + i, "main-logs": "log of task " + i} {
					key := artifactKey(t, n, name)
					assert.True(t, strings.HasPrefix(key, prefix), "key %s of %s isn't under %s", key, n.DisplayName, prefix)
					if other, ok := keys[key]; ok {
						assert.Fail(t, "shared key", "%s and %s/%s share key %s", other, n.DisplayName, name, key)
					}
					keys[key] = n.DisplayName + "/" + name
					assert.Contains(t, fixtures.ArtifactContent(t, hotWorkersBucket, key), want)
				}
			}
		}).
		ExpectPods(func(t *testing.T, pods []apiv1.Pod) {
			assert.Empty(t, pods, "a pooled workflow has no pods of its own")
		})
}

// TestHotWorkersExample runs the hot workers example, which the examples suite
// skips, as it needs hot workers enabled.
func (s *HotWorkersSuite) TestHotWorkersExample() {
	ctx := logging.TestContext(s.T().Context())
	data, err := os.ReadFile("../../examples/hot-workers.yaml")
	s.CheckError(err)
	wfs, err := common.SplitWorkflowYAMLFile(ctx, data, true)
	s.CheckError(err)
	s.Require().Len(wfs, 1)
	s.Given().
		ExampleWorkflow(&wfs[0]).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 3*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			nodes := pooledNodes(t, status, "python-squares")
			require.Len(t, nodes, 11)
			sum := node(t, status, "sum")
			assert.Contains(t, fixtures.ArtifactContent(t, hotWorkersBucket, artifactKey(t, *sum, "main-logs")), "285", "the sum of the squares of 0 to 9")
		})
}

func TestHotWorkersSuite(t *testing.T) {
	suite.Run(t, new(HotWorkersSuite))
}
