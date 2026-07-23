package cron

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/pkg/sync"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/scheme"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

var secondParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// create new controller configured with fake clients.
func newTestCronWorkflowController(t *testing.T) (context.Context, context.CancelFunc, *Controller) {
	objects := []runtime.Object{}
	wfClientset := fakewfclientset.NewSimpleClientset(objects...)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme, objects...)
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logging.TestContext(ctx)

	testMetrics, err := metrics.New(logging.TestContext(t.Context()), telemetry.TestScopeName, telemetry.TestScopeName, &telemetry.MetricsConfig{}, metrics.Callbacks{})
	if err != nil {
		log.Fatal(err.Error())
	}
	cc := &Controller{
		wfClientset:      wfClientset,
		namespace:        "default",
		managedNamespace: "default",
		instanceID:       "test",
		keyLock:          sync.NewKeyLock(),
		dynamicInterface: dynamicClient,
		cronWfQueue:      testMetrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultTypedControllerRateLimiter[string](), "cron_wf_queue"),
		metrics:          testMetrics,
	}

	cc.cronWfInformer = dynamicinformer.NewFilteredDynamicSharedInformerFactory(cc.dynamicInterface, cronWorkflowResyncPeriod,
		cc.managedNamespace, func(_ *metav1.ListOptions) {}).
		ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural})
	_ = cc.addCronWorkflowInformerHandler(ctx)
	go cc.cronWfInformer.Informer().Run(ctx.Done())

	cc.wfInformer = util.NewWorkflowInformer(ctx, cc.dynamicInterface, cc.managedNamespace, cronWorkflowResyncPeriod,
		func(_ *metav1.ListOptions) {}, func(_ *metav1.ListOptions) {}, indexes)
	go cc.wfInformer.Run(ctx.Done())
	cc.wfLister = util.NewWorkflowLister(ctx, cc.wfInformer)

	for _, c := range []cache.SharedIndexInformer{
		cc.wfInformer,
		cc.cronWfInformer.Informer(),
	} {
		for !c.HasSynced() {
			time.Sleep(5 * time.Millisecond)
		}
	}

	return ctx, cancel, cc
}

var helloWf = `
  apiVersion: argoproj.io/v1alpha1
  kind: CronWorkflow
  metadata:
    name: hello-world
  spec:
    schedules:
      - "* * * * *"
    concurrencyPolicy: Replace
    startingDeadlineSeconds: 120
    workflowSpec:
      entrypoint: whalesay
      templates:
        - name: whalesay
          container:
            image: argoproj/argosay:v2
`

// Ensure that cron job runs are blocked by the keyLock being held.
func TestCronJobRace(t *testing.T) {
	ctx, cancel, cc := newTestCronWorkflowController(t)
	defer cancel()

	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(helloWf), &cronWf)
	cwf, err := cc.wfClientset.ArgoprojV1alpha1().CronWorkflows("default").Create(ctx, &cronWf, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating cronworkflow: %s", err.Error())
	}
	un, err := util.CronToUnstructured(cwf)
	if err != nil {
		t.Fatalf("error converting to unstructured: %s", err.Error())
	}
	err = cc.cronWfInformer.Informer().GetStore().Add(un)
	if err != nil {
		t.Fatalf("error adding cronworkflow to informer cache: %s", err.Error())
	}
	cwoc := newCronWfOperationCtx(ctx, cwf, cc.wfClientset, cc.metrics, cc.wftmplInformer, cc.cwftmplInformer, cc.wfDefaults, cc.keyLock)

	// start cron scheduler with second resolution schedule support to make the test run quicker.
	cron := newCronFacade(cron.WithParser(secondParser))
	cron.Start()
	defer cron.Stop()
	key, err := cache.MetaNamespaceKeyFunc(cwf)
	if err != nil {
		t.Fatalf("error getting key from cronworkflow: %s", err.Error())
	}

	// Lock is held until test ends which should prevent the cron run from mutating the cronworkflow.
	// This emulates an active execution of processNextCronItem or syncCronWorkflow.
	cc.keyLock.Lock(key)
	defer cc.keyLock.Unlock(key)

	// Schedule job to run every second
	_, err = cron.AddJob(key, "* * * * * *", cwoc)
	if err != nil {
		t.Fatalf("error adding cron job: %s", err.Error())
	}
	// Sleep for slightly longer than a second to let the job attempt to run.
	time.Sleep(1*time.Second + 50*time.Millisecond)
	cron.Delete(key)

	afterCwf, err := cc.wfClientset.ArgoprojV1alpha1().CronWorkflows("default").Get(ctx, "hello-world", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	// Since the lock is still held this should still be 0
	if len(afterCwf.Status.Active) > 0 {
		t.Errorf("cronworkflow has non-empty .status.active")
	}
}
