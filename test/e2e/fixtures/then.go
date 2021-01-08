package fixtures

import (
	"context"
	"reflect"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/hydrator"
)

type Then struct {
	t          *testing.T
	wf         *wfv1.Workflow
	cronWf     *wfv1.CronWorkflow
	client     v1alpha1.WorkflowInterface
	cronClient v1alpha1.CronWorkflowInterface
	hydrator   hydrator.Interface
	kubeClient kubernetes.Interface
}

func (t *Then) ExpectWorkflow(block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	t.t.Helper()
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
	println("Checking expectation", workflowName)

	ctx := context.Background()
	wf, err := t.client.Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	err = t.hydrator.Hydrate(wf)
	if err != nil {
		t.t.Fatal(err)
	}
	println(wf.Name, ":", wf.Status.Phase, wf.Status.Message)
	block(t.t, &wf.ObjectMeta, &wf.Status)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) ExpectWorkflowDeleted() *Then {
	ctx := context.Background()
	_, err := t.client.Get(ctx, t.wf.Name, metav1.GetOptions{})
	if err == nil || !apierr.IsNotFound(err) {
		t.t.Fatalf("expected workflow to be deleted: %v", err)
	}
	return t
}

// Check on a specific node in the workflow.
// If no node matches the selector, then the NodeStatus and Pod will be nil.
// If the pod does not exist (e.g. because it was deleted) then the Pod will be nil too.
func (t *Then) ExpectWorkflowNode(selector func(status wfv1.NodeStatus) bool, f func(t *testing.T, status *wfv1.NodeStatus, pod *apiv1.Pod)) *Then {
	return t.expectWorkflow(t.wf.Name, func(tt *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
		n := status.Nodes.Find(selector)
		var p *apiv1.Pod
		if n != nil {
			println("Found node", "id="+n.ID, "type="+n.Type)
			if n.Type == wfv1.NodeTypePod {
				var err error
				ctx := context.Background()
				p, err = t.kubeClient.CoreV1().Pods(t.wf.Namespace).Get(ctx, n.ID, metav1.GetOptions{})
				if err != nil {
					if !apierr.IsNotFound(err) {
						t.t.Fatal(err)
					}
					p = nil // i did not expect to need to nil the pod, but here we are
				}
			}
		} else {
			println("Did not find node")
		}
		f(tt, n, p)
	})
}

func (t *Then) ExpectCron(block func(t *testing.T, cronWf *wfv1.CronWorkflow)) *Then {
	t.t.Helper()
	if t.cronWf == nil {
		t.t.Fatal("No cron workflow to test")
	}
	println("Checking cron expectation")

	ctx := context.Background()
	cronWf, err := t.cronClient.Get(ctx, t.cronWf.Name, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, cronWf)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) ExpectWorkflowList(listOptions metav1.ListOptions, block func(t *testing.T, wfList *wfv1.WorkflowList)) *Then {
	t.t.Helper()
	println("Listing workflows")

	ctx := context.Background()
	wfList, err := t.client.List(ctx, listOptions)
	if err != nil {
		t.t.Fatal(err)
	}
	println("Checking expectation")
	block(t.t, wfList)
	if t.t.Failed() {
		t.t.FailNow()
	}
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

	ctx := context.Background()
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
			t.t.Fatal("timeout waiting for events")
		case event := <-eventList.ResultChan():
			e, ok := event.Object.(*apiv1.Event)
			if !ok {
				t.t.Fatalf("event is not an event: %v", reflect.TypeOf(e).String())
			}
			if filter(*e) {
				println("event", e.InvolvedObject.Kind+"/"+e.InvolvedObject.Name, e.Reason)
				events = append(events, *e)
			}
		}
	}
	block(t.t, events)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) RunCli(args []string, block func(t *testing.T, output string, err error)) *Then {
	t.t.Helper()
	output, err := Exec("../../dist/argo", append([]string{"-n", Namespace}, args...)...)
	block(t.t, output, err)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) When() *When {
	return &When{
		t:          t.t,
		client:     t.client,
		cronClient: t.cronClient,
		hydrator:   t.hydrator,
		wf:         t.wf,
		kubeClient: t.kubeClient,
	}
}
