package fixtures

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

type Then struct {
	t           *testing.T
	wf          *wfv1.Workflow
	cronWf      *wfv1.CronWorkflow
	client      v1alpha1.WorkflowInterface
	wftsClient  v1alpha1.WorkflowTaskSetInterface
	cronClient  v1alpha1.CronWorkflowInterface
	hydrator    hydrator.Interface
	kubeClient  kubernetes.Interface
	bearerToken string
	restConfig  *rest.Config
}

func (t *Then) ExpectWorkflow(block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	t.t.Helper()
	if t.wf == nil {
		t.t.Error("workflows is nil")
		return t
	}
	return t.expectWorkflow(t.wf.Name, block)
}

func (t *Then) ExpectWorkflowName(workflowName string, block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	t.t.Helper()
	return t.expectWorkflow(workflowName, block)
}

func (t *Then) expectWorkflow(workflowName string, block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	t.t.Helper()
	if workflowName == "" {
		t.t.Fatal("No workflow to test")
	}
	_, _ = fmt.Println("Checking expectation", workflowName)

	ctx := logging.TestContext(t.t.Context())
	wf, err := t.client.Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	err = t.hydrator.Hydrate(ctx, wf)
	if err != nil {
		t.t.Fatal(err)
	}
	_, _ = fmt.Println(wf.Name, ":", wf.Status.Phase, wf.Status.Message)
	block(t.t, &wf.ObjectMeta, &wf.Status)
	return t
}

func (t *Then) ExpectWorkflowDeleted() *Then {
	ctx := logging.TestContext(t.t.Context())
	_, err := t.client.Get(ctx, t.wf.Name, metav1.GetOptions{})
	if err == nil || !apierr.IsNotFound(err) {
		t.t.Errorf("expected workflow to be deleted: %v", err)
	}
	return t
}

// ExpectWorkflowNode checks on a specific node in the workflow.
// If no node matches the selector, then the NodeStatus and Pod will be nil.
// If the pod does not exist (e.g. because it was deleted) then the Pod will be nil too.
func (t *Then) ExpectWorkflowNode(selector func(status wfv1.NodeStatus) bool, f func(t *testing.T, status *wfv1.NodeStatus, pod *apiv1.Pod)) *Then {
	return t.expectWorkflow(t.wf.Name, func(tt *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
		n := status.Nodes.Find(selector)
		var p *apiv1.Pod
		if n != nil {
			_, _ = fmt.Println("Found node", "id="+n.ID, "type="+n.Type)
			if n.Type == wfv1.NodeTypePod {
				wf := &wfv1.Workflow{
					ObjectMeta: *metadata,
				}
				version := util.GetWorkflowPodNameVersion(wf)
				podName := util.GeneratePodName(t.wf.Name, n.Name, util.GetTemplateFromNode(*n), n.ID, version)

				var err error
				ctx := logging.TestContext(t.t.Context())
				p, err = t.kubeClient.CoreV1().Pods(t.wf.Namespace).Get(ctx, podName, metav1.GetOptions{})
				if err != nil {
					if !apierr.IsNotFound(err) {
						t.t.Error(err)
					}
					p = nil // i did not expect to need to nil the pod, but here we are
				}
			}
			f(tt, n, p)
		} else {
			_, _ = fmt.Println("Did not find node")
			t.t.Error("Did not find expected node")
		}
	})
}

func (t *Then) ExpectCron(block func(t *testing.T, cronWf *wfv1.CronWorkflow)) *Then {
	t.t.Helper()
	if t.cronWf == nil {
		t.t.Fatal("No cron workflow to test")
	}
	_, _ = fmt.Println("Checking cron expectation")

	ctx := logging.TestContext(t.t.Context())
	cronWf, err := t.cronClient.Get(ctx, t.cronWf.Name, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, cronWf)
	return t
}

func (t *Then) ExpectWorkflowListFromCronWorkflow(block func(t *testing.T, wfList *wfv1.WorkflowList)) *Then {
	t.t.Helper()
	if t.cronWf == nil {
		t.t.Fatal("No cron workflow to match against")
	}
	labelSelector := common.LabelKeyCronWorkflow + "=" + t.cronWf.Name
	return t.ExpectWorkflowList(metav1.ListOptions{LabelSelector: labelSelector}, block)
}

func (t *Then) ExpectWorkflowList(listOptions metav1.ListOptions, block func(t *testing.T, wfList *wfv1.WorkflowList)) *Then {
	t.t.Helper()
	_, _ = fmt.Println("Listing workflows")

	ctx := logging.TestContext(t.t.Context())
	wfList, err := t.client.List(ctx, listOptions)
	if err != nil {
		t.t.Fatal(err)
	}
	_, _ = fmt.Println("Checking expectation")
	block(t.t, wfList)
	return t
}

var HasInvolvedObject = func(kind string, uid types.UID) func(event apiv1.Event) bool {
	return func(e apiv1.Event) bool {
		return e.InvolvedObject.Kind == kind && e.InvolvedObject.UID == uid
	}
}

var HasInvolvedObjectWithName = func(kind string, name string) func(event apiv1.Event) bool {
	return func(e apiv1.Event) bool {
		return e.InvolvedObject.Kind == kind && e.InvolvedObject.Name == name
	}
}

func (t *Then) ExpectAuditEvents(filter func(event apiv1.Event) bool, num int, block func(*testing.T, []apiv1.Event)) *Then {
	t.t.Helper()

	ctx := logging.TestContext(t.t.Context())
	eventList, err := t.kubeClient.CoreV1().Events(Namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	ticker := time.NewTicker(defaultTimeout)
	defer ticker.Stop()
	var events []apiv1.Event
	for num > len(events) {
		select {
		case <-ticker.C:
			t.t.Error("timeout waiting for events")
			return t
		case event := <-eventList.ResultChan():
			e, ok := event.Object.(*apiv1.Event)
			if !ok {
				t.t.Errorf("event is not an event: %v", reflect.TypeFor[*apiv1.Event]().String())
				return t
			}
			if filter(*e) {
				_, _ = fmt.Println("event", e.InvolvedObject.Kind+"/"+e.InvolvedObject.Name, e.Reason)
				events = append(events, *e)
			}
		}
	}
	block(t.t, events)
	return t
}

func (t *Then) ExpectPVCDeleted() *Then {
	t.t.Helper()
	timeout := defaultTimeout
	_, _ = fmt.Println("Checking", timeout.String(), "for expecting PVCs deletion")
	ctx, cancel := context.WithTimeout(func() context.Context {
		ctx := logging.TestContext(t.t.Context())
		return ctx
	}(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			t.t.Errorf("timeout after %v waiting for condition", timeout)
			return t
		default:
			num := len(t.wf.Status.PersistentVolumeClaims)
			pvcClient := t.kubeClient.CoreV1().PersistentVolumeClaims(t.wf.Namespace)
			for _, p := range t.wf.Status.PersistentVolumeClaims {
				_, err := pvcClient.Get(ctx, p.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
				switch {
				case err == nil:
					// PVC exists, continue checking others
				case apierr.IsNotFound(err):
					num--
				default:
					t.t.Fatal(err)
					return t
				}
			}
			if num == 0 {
				return t
			}
		}
	}
}

func (t *Then) ExpectArtifact(nodeName string, artifactName string, bucketName string, f func(t *testing.T, object minio.ObjectInfo, err error)) {
	t.t.Helper()

	if nodeName == "-" {
		nodeName = t.wf.Name
	}

	n, err := t.wf.GetNodeByName(nodeName)
	if err != nil {
		t.t.Error("was unable to get node by name")
		return
	}
	a := n.GetOutputs().GetArtifactByName(artifactName)
	if a == nil {
		t.t.Error("was unable to get artifact by name")
		return
	}
	key, _ := a.GetKey()

	t.ExpectArtifactByKey(key, bucketName, f)
}

func (t *Then) ExpectArtifactByKey(key string, bucketName string, f func(t *testing.T, object minio.ObjectInfo, err error)) {
	t.t.Helper()

	c, err := minio.New("localhost:9000", &minio.Options{
		Creds: credentials.NewStaticV4("admin", "password", ""),
	})

	if err != nil {
		t.t.Error(err)
	}

	ctx := logging.TestContext(t.t.Context())
	object, err := c.StatObject(ctx, bucketName, key, minio.StatObjectOptions{})
	f(t.t, object, err)
}

func (t *Then) ExpectPods(f func(t *testing.T, pods []apiv1.Pod)) *Then {
	t.t.Helper()

	ctx := logging.TestContext(t.t.Context())
	list, err := t.kubeClient.CoreV1().Pods(t.wf.Namespace).List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + t.wf.Name})
	if err != nil {
		t.t.Fatal(err)
	}

	f(t.t, list.Items)

	return t
}

func (t *Then) ExpectContainerLogs(container string, f func(t *testing.T, logs string)) *Then {
	t.t.Helper()

	ctx := logging.TestContext(t.t.Context())
	stream, err := t.kubeClient.CoreV1().Pods(t.wf.Namespace).GetLogs(t.wf.Name, &apiv1.PodLogOptions{Container: container}).Stream(ctx)
	if err != nil {
		t.t.Fatal(err)
	}

	defer stream.Close()
	logBytes, err := io.ReadAll(stream)
	if err != nil {
		t.t.Fatal(err)
	}

	f(t.t, string(logBytes))

	return t
}

func (t *Then) ExpectWorkflowTaskSet(block func(t *testing.T, wfts *wfv1.WorkflowTaskSet)) *Then {
	t.t.Helper()
	ctx := logging.TestContext(t.t.Context())
	wfts, err := t.wftsClient.Get(ctx, t.wf.Name, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, wfts)
	return t
}

func (t *Then) RunCli(args []string, block func(t *testing.T, output string, err error)) *Then {
	t.t.Helper()
	ctx := logging.TestContext(t.t.Context())
	output, err := Exec(ctx, "../../dist/argo", "", append([]string{"-n", Namespace}, args...)...)
	block(t.t, output, err)
	return t
}

func (t *Then) When() *When {
	return &When{
		t:           t.t,
		client:      t.client,
		wftsClient:  t.wftsClient,
		cronClient:  t.cronClient,
		hydrator:    t.hydrator,
		wf:          t.wf,
		kubeClient:  t.kubeClient,
		bearerToken: t.bearerToken,
		restConfig:  t.restConfig,
	}
}
