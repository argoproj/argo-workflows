package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/argoproj/pkg/errors"
	syncpkg "github.com/argoproj/pkg/sync"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	apiwatch "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/workqueue"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/config"
	argoErr "github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	authutil "github.com/argoproj/argo-workflows/v3/util/auth"
	"github.com/argoproj/argo-workflows/v3/util/diff"
	"github.com/argoproj/argo-workflows/v3/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/estimation"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/informer"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/pod"
	"github.com/argoproj/argo-workflows/v3/workflow/cron"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/signal"
	"github.com/argoproj/argo-workflows/v3/workflow/sync"
	"github.com/argoproj/argo-workflows/v3/workflow/ttlcontroller"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

// WorkflowController is the controller for workflow resources
type WorkflowController struct {
	// namespace of the workflow controller
	namespace        string
	managedNamespace string

	configController config.Controller
	// Config is the workflow controller's configuration
	Config config.Config
	// get the artifact repository
	artifactRepositories artifactrepositories.Interface

	// cliExecutorImage is the executor image as specified from the command line
	cliExecutorImage string

	// cliExecutorImagePullPolicy is the executor imagePullPolicy as specified from the command line
	cliExecutorImagePullPolicy string
	containerRuntimeExecutor   string

	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig       *rest.Config
	kubeclientset    kubernetes.Interface
	rateLimiter      *rate.Limiter
	dynamicInterface dynamic.Interface
	wfclientset      wfclientset.Interface

	// datastructures to support the processing of workflows and workflow pods
	wfInformer            cache.SharedIndexInformer
	wftmplInformer        wfextvv1alpha1.WorkflowTemplateInformer
	cwftmplInformer       wfextvv1alpha1.ClusterWorkflowTemplateInformer
	podInformer           cache.SharedIndexInformer
	wfQueue               workqueue.RateLimitingInterface
	podQueue              workqueue.RateLimitingInterface
	podCleanupQueue       workqueue.RateLimitingInterface // pods to be deleted or labelled depend on GC strategy
	throttler             sync.Throttler
	workflowKeyLock       syncpkg.KeyLock // used to lock workflows for exclusive modification or access
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	hydrator              hydrator.Interface
	wfArchive             sqldb.WorkflowArchive
	estimatorFactory      estimation.EstimatorFactory
	syncManager           *sync.Manager
	metrics               *metrics.Metrics
	eventRecorderManager  events.EventRecorderManager
	archiveLabelSelector  labels.Selector
	cacheFactory          controllercache.Factory
}

const (
	workflowResyncPeriod                = 20 * time.Minute
	workflowTemplateResyncPeriod        = 20 * time.Minute
	podResyncPeriod                     = 30 * time.Minute
	clusterWorkflowTemplateResyncPeriod = 20 * time.Minute
	workflowExistenceCheckPeriod        = 1 * time.Minute
)

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(ctx context.Context, restConfig *rest.Config, kubeclientset kubernetes.Interface, wfclientset wfclientset.Interface, namespace, managedNamespace, executorImage, executorImagePullPolicy, containerRuntimeExecutor, configMap string) (*WorkflowController, error) {
	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	wfc := WorkflowController{
		restConfig:                 restConfig,
		kubeclientset:              kubeclientset,
		dynamicInterface:           dynamicInterface,
		wfclientset:                wfclientset,
		namespace:                  namespace,
		managedNamespace:           managedNamespace,
		cliExecutorImage:           executorImage,
		cliExecutorImagePullPolicy: executorImagePullPolicy,
		containerRuntimeExecutor:   containerRuntimeExecutor,
		configController:           config.NewController(namespace, configMap, kubeclientset, config.EmptyConfigFunc),
		workflowKeyLock:            syncpkg.NewKeyLock(),
		cacheFactory:               controllercache.NewCacheFactory(kubeclientset, namespace),
		eventRecorderManager:       events.NewEventRecorderManager(kubeclientset),
	}

	wfc.UpdateConfig(ctx)

	wfc.metrics = metrics.New(wfc.getMetricsServerConfig())

	workqueue.SetProvider(wfc.metrics) // must execute SetProvider before we created the queues
	wfc.wfQueue = wfc.metrics.RateLimiterWithBusyWorkers(&fixedItemIntervalRateLimiter{}, "workflow_queue")
	wfc.throttler = wfc.newThrottler()
	wfc.podQueue = wfc.metrics.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "pod_queue")
	wfc.podCleanupQueue = wfc.metrics.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "pod_cleanup_queue")

	return &wfc, nil
}

func (wfc *WorkflowController) newThrottler() sync.Throttler {
	f := func(key string) { wfc.wfQueue.AddRateLimited(key) }
	return sync.ChainThrottler{
		sync.NewThrottler(wfc.Config.Parallelism, sync.SingleBucket, f),
		sync.NewThrottler(wfc.Config.NamespaceParallelism, sync.NamespaceBucket, f),
	}
}

// RunTTLController runs the workflow TTL controller
func (wfc *WorkflowController) runTTLController(ctx context.Context, workflowTTLWorkers int) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	ttlCtrl := ttlcontroller.NewController(wfc.wfclientset, wfc.wfInformer, wfc.metrics)
	err := ttlCtrl.Run(ctx.Done(), workflowTTLWorkers)
	if err != nil {
		panic(err)
	}
}

func (wfc *WorkflowController) runCronController(ctx context.Context) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	cronController := cron.NewCronController(wfc.wfclientset, wfc.dynamicInterface, wfc.namespace, wfc.GetManagedNamespace(), wfc.Config.InstanceID, wfc.metrics, wfc.eventRecorderManager)
	cronController.Run(ctx)
}

var indexers = cache.Indexers{
	indexes.ClusterWorkflowTemplateIndex: indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyClusterWorkflowTemplate),
	indexes.CronWorkflowIndex:            indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyCronWorkflow),
	indexes.WorkflowTemplateIndex:        indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyWorkflowTemplate),
	indexes.SemaphoreConfigIndexName:     indexes.WorkflowSemaphoreKeysIndexFunc(),
	indexes.WorkflowPhaseIndex:           indexes.MetaWorkflowPhaseIndexFunc(),
	indexes.ConditionsIndex:              indexes.ConditionsIndexFunc,
	indexes.UIDIndex:                     indexes.MetaUIDFunc,
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context, wfWorkers, workflowTTLWorkers, podWorkers, podCleanupWorkers int) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()
	defer wfc.podCleanupQueue.ShutDown()

	log.WithField("version", argo.GetVersion().Version).Info("Starting Workflow Controller")
	log.Infof("Workers: workflow: %d, pod: %d, pod cleanup: %d", wfWorkers, podWorkers, podCleanupWorkers)

	wfc.wfInformer = util.NewWorkflowInformer(wfc.dynamicInterface, wfc.GetManagedNamespace(), workflowResyncPeriod, wfc.tweakListOptions, indexers)
	wfc.wftmplInformer = informer.NewTolerantWorkflowTemplateInformer(wfc.dynamicInterface, workflowTemplateResyncPeriod, wfc.managedNamespace)

	wfc.addWorkflowInformerHandlers(ctx)
	wfc.podInformer = wfc.newPodInformer(ctx)
	wfc.updateEstimatorFactory()

	go wfc.runConfigMapWatcher(ctx.Done())
	go wfc.configController.Run(ctx.Done(), wfc.updateConfig)
	go wfc.wfInformer.Run(ctx.Done())
	go wfc.wftmplInformer.Informer().Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(ctx.Done(), wfc.wfInformer.HasSynced, wfc.wftmplInformer.Informer().HasSynced, wfc.podInformer.HasSynced) {
		log.Fatal("Timed out waiting for caches to sync")
	}

	wfc.createClusterWorkflowTemplateInformer(ctx)

	// Create Synchronization Manager
	err := wfc.createSynchronizationManager(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Start the metrics server
	go wfc.metrics.RunServer(ctx)

	leaderElectionOff := os.Getenv("LEADER_ELECTION_DISABLE")
	if leaderElectionOff == "true" {
		log.Info("Leader election is turned off. Running in single-instance mode")
		logCtx := log.WithField("id", "single-instance")
		go wfc.startLeading(ctx, logCtx, podCleanupWorkers, workflowTTLWorkers, wfWorkers, podWorkers)
	} else {
		nodeID, ok := os.LookupEnv("LEADER_ELECTION_IDENTITY")
		if !ok {
			log.Fatal("LEADER_ELECTION_IDENTITY must be set so that the workflow controllers can elect a leader")
		}
		logCtx := log.WithField("id", nodeID)

		leaderName := "workflow-controller"
		if wfc.Config.InstanceID != "" {
			leaderName = fmt.Sprintf("%s-%s", leaderName, wfc.Config.InstanceID)
		}

		go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.LeaseLock{
				LeaseMeta: metav1.ObjectMeta{Name: leaderName, Namespace: wfc.namespace}, Client: wfc.kubeclientset.CoordinationV1(),
				LockConfig: resourcelock.ResourceLockConfig{Identity: nodeID, EventRecorder: wfc.eventRecorderManager.Get(wfc.namespace)},
			},
			ReleaseOnCancel: true,
			LeaseDuration:   env.LookupEnvDurationOr("LEADER_ELECTION_LEASE_DURATION", 15*time.Second),
			RenewDeadline:   env.LookupEnvDurationOr("LEADER_ELECTION_RENEW_DEADLINE", 10*time.Second),
			RetryPeriod:     env.LookupEnvDurationOr("LEADER_ELECTION_RETRY_PERIOD", 5*time.Second),
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					wfc.startLeading(ctx, logCtx, podCleanupWorkers, workflowTTLWorkers, wfWorkers, podWorkers)
				},
				OnStoppedLeading: func() {
					logCtx.Info("stopped leading")
					cancel()
				},
				OnNewLeader: func(identity string) {
					logCtx.WithField("leader", identity).Info("new leader")
				},
			},
		})
	}
	<-ctx.Done()
}

func (wfc *WorkflowController) startLeading(ctx context.Context, logCtx *log.Entry, podCleanupWorkers int, workflowTTLWorkers int, wfWorkers int, podWorkers int) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	logCtx.Info("started leading")

	for i := 0; i < podCleanupWorkers; i++ {
		go wait.UntilWithContext(ctx, wfc.runPodCleanup, time.Second)
	}
	go wfc.workflowGarbageCollector(ctx.Done())
	go wfc.archivedWorkflowGarbageCollector(ctx.Done())

	go wfc.runTTLController(ctx, workflowTTLWorkers)
	go wfc.runCronController(ctx)
	go wait.Until(wfc.syncWorkflowPhaseMetrics, 15*time.Second, ctx.Done())
	go wait.Until(wfc.syncPodPhaseMetrics, 15*time.Second, ctx.Done())

	go wait.Until(wfc.syncManager.CheckWorkflowExistence, workflowExistenceCheckPeriod, ctx.Done())

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}
}

func (wfc *WorkflowController) waitForCacheSync(ctx context.Context) {
	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(ctx.Done(), wfc.wfInformer.HasSynced, wfc.wftmplInformer.Informer().HasSynced, wfc.podInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	if wfc.cwftmplInformer != nil {
		if !cache.WaitForCacheSync(ctx.Done(), wfc.cwftmplInformer.Informer().HasSynced) {
			panic("Timed out waiting for caches to sync")
		}
	}
}

// Create and initialize the Synchronization Manager
func (wfc *WorkflowController) createSynchronizationManager(ctx context.Context) error {
	getSyncLimit := func(lockKey string) (int, error) {
		lockName, err := sync.DecodeLockName(lockKey)
		if err != nil {
			return 0, err
		}
		configMap, err := wfc.kubeclientset.CoreV1().ConfigMaps(lockName.Namespace).Get(ctx, lockName.ResourceName, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}

		value, found := configMap.Data[lockName.Key]
		if !found {
			return 0, argoErr.New(argoErr.CodeBadRequest, fmt.Sprintf("Sync configuration key '%s' not found in ConfigMap", lockName.Key))
		}
		return strconv.Atoi(value)
	}

	nextWorkflow := func(key string) {
		wfc.wfQueue.AddRateLimited(key)
	}

	isWFDeleted := func(key string) bool {
		_, exists, err := wfc.wfInformer.GetIndexer().GetByKey(key)
		if err != nil {
			log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get workflow from informer")
			return false
		}
		return exists
	}

	wfc.syncManager = sync.NewLockManager(getSyncLimit, nextWorkflow, isWFDeleted)

	labelSelector := labels.NewSelector()
	req, _ := labels.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(wfv1.NodeRunning)})
	if req != nil {
		labelSelector = labelSelector.Add(*req)
	}

	listOpts := metav1.ListOptions{LabelSelector: labelSelector.String()}
	wfList, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wfc.namespace).List(ctx, listOpts)
	if err != nil {
		return err
	}

	wfc.syncManager.Initialize(wfList.Items)
	return nil
}

func (wfc *WorkflowController) runConfigMapWatcher(stopCh <-chan struct{}) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	ctx := context.Background()
	retryWatcher, err := apiwatch.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return wfc.kubeclientset.CoreV1().ConfigMaps(wfc.managedNamespace).Watch(ctx, metav1.ListOptions{})
		},
	})
	if err != nil {
		panic(err)
	}
	defer retryWatcher.Stop()

	for {
		select {
		case event := <-retryWatcher.ResultChan():
			cm, ok := event.Object.(*apiv1.ConfigMap)
			if !ok {
				log.Errorf("invalid config map object received in config watcher. Ignored processing")
				continue
			}
			log.Debugf("received config map %s/%s update", cm.Namespace, cm.Name)
			wfc.notifySemaphoreConfigUpdate(cm)

		case <-stopCh:
			return
		}
	}
}

// notifySemaphoreConfigUpdate will notify semaphore config update to pending workflows
func (wfc *WorkflowController) notifySemaphoreConfigUpdate(cm *apiv1.ConfigMap) {
	wfs, err := wfc.wfInformer.GetIndexer().ByIndex(indexes.SemaphoreConfigIndexName, fmt.Sprintf("%s/%s", cm.Namespace, cm.Name))
	if err != nil {
		log.Errorf("failed get the workflow from informer. %v", err)
	}

	for _, obj := range wfs {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			log.Warnf("received object from indexer %s is not an unstructured", indexes.SemaphoreConfigIndexName)
			continue
		}
		wfc.wfQueue.AddRateLimited(fmt.Sprintf("%s/%s", un.GetNamespace(), un.GetName()))
	}
}

// Check if the controller has RBAC access to ClusterWorkflowTemplates
func (wfc *WorkflowController) createClusterWorkflowTemplateInformer(ctx context.Context) {
	cwftGetAllowed, err := authutil.CanI(ctx, wfc.kubeclientset, "get", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)
	cwftListAllowed, err := authutil.CanI(ctx, wfc.kubeclientset, "list", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)
	cwftWatchAllowed, err := authutil.CanI(ctx, wfc.kubeclientset, "watch", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)

	if cwftGetAllowed && cwftListAllowed && cwftWatchAllowed {
		wfc.cwftmplInformer = informer.NewTolerantClusterWorkflowTemplateInformer(wfc.dynamicInterface, clusterWorkflowTemplateResyncPeriod)
		go wfc.cwftmplInformer.Informer().Run(ctx.Done())
	} else {
		log.Warnf("Controller doesn't have RBAC access for ClusterWorkflowTemplates")
	}
}

func (wfc *WorkflowController) UpdateConfig(ctx context.Context) {
	config, err := wfc.configController.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to register watch for controller config map: %v", err)
	}
	err = wfc.updateConfig(config)
	if err != nil {
		log.Fatalf("Failed to update config: %v", err)
	}
}

func (wfc *WorkflowController) queuePodForCleanup(namespace string, podName string, action podCleanupAction) {
	wfc.podCleanupQueue.AddRateLimited(newPodCleanupKey(namespace, podName, action))
}

func (wfc *WorkflowController) queuePodForCleanupAfter(namespace string, podName string, action podCleanupAction, duration time.Duration) {
	wfc.podCleanupQueue.AddAfter(newPodCleanupKey(namespace, podName, action), duration)
}

func (wfc *WorkflowController) runPodCleanup(ctx context.Context) {
	for wfc.processNextPodCleanupItem(ctx) {
	}
}

// all pods will ultimately be cleaned up by either deleting them, or labelling them
func (wfc *WorkflowController) processNextPodCleanupItem(ctx context.Context) bool {
	key, quit := wfc.podCleanupQueue.Get()
	if quit {
		return false
	}
	defer wfc.podCleanupQueue.Done(key)

	namespace, podName, action := parsePodCleanupKey(key.(podCleanupKey))
	logCtx := log.WithFields(log.Fields{"key": key, "action": action})
	logCtx.Info("cleaning up pod")
	err := func() error {
		pods := wfc.kubeclientset.CoreV1().Pods(namespace)
		switch action {
		case shutdownPod:
			// to shutdown a pod, we signal the wait container to terminate, the wait container in turn will
			// kill the main container (using whatever mechanism the executor uses), and will then exit itself
			// once the main container exited
			pod, err := wfc.getPod(namespace, podName)
			if pod == nil || err != nil {
				return err
			}
			for _, c := range pod.Spec.Containers {
				if c.Name == common.WaitContainerName {
					if err := signal.SignalContainer(wfc.restConfig, pod, common.WaitContainerName, syscall.SIGTERM); err != nil {
						return err
					}
					return nil // done
				}
			}
			// no wait container found
			fallthrough
		case terminateContainers:
			if terminationGracePeriod, err := wfc.signalContainers(namespace, podName, syscall.SIGTERM); err != nil {
				return err
			} else if terminationGracePeriod > 0 {
				wfc.queuePodForCleanupAfter(namespace, podName, killContainers, terminationGracePeriod)
			}
		case killContainers:
			if _, err := wfc.signalContainers(namespace, podName, syscall.SIGKILL); err != nil {
				return err
			}
		case labelPodCompleted:
			_, err := pods.Patch(
				ctx,
				podName,
				types.MergePatchType,
				[]byte(`{"metadata": {"labels": {"workflows.argoproj.io/completed": "true"}}}`),
				metav1.PatchOptions{},
			)
			if err != nil {
				return err
			}
		case deletePod:
			propagation := metav1.DeletePropagationBackground
			err := pods.Delete(ctx, podName, metav1.DeleteOptions{
				PropagationPolicy:  &propagation,
				GracePeriodSeconds: wfc.Config.PodGCGracePeriodSeconds,
			})
			if err != nil && !apierr.IsNotFound(err) {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		logCtx.WithError(err).Warn("failed to clean-up pod")
		if errorsutil.IsTransientErr(err) {
			wfc.podCleanupQueue.AddRateLimited(key)
		}
	}
	return true
}

func (wfc *WorkflowController) getPod(namespace string, podName string) (*apiv1.Pod, error) {
	obj, exists, err := wfc.podInformer.GetStore().GetByKey(namespace + "/" + podName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		return nil, fmt.Errorf("object is not a pod")
	}
	return pod, nil
}

func (wfc *WorkflowController) signalContainers(namespace string, podName string, sig syscall.Signal) (time.Duration, error) {
	pod, err := wfc.getPod(namespace, podName)
	if pod == nil || err != nil {
		return 0, err
	}

	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == common.WaitContainerName || c.State.Terminated != nil {
			continue
		}
		if err := signal.SignalContainer(wfc.restConfig, pod, c.Name, sig); err != nil {
			return 0, err
		}
	}
	if pod.Spec.TerminationGracePeriodSeconds == nil {
		return 30 * time.Second, nil
	}
	return time.Duration(*pod.Spec.TerminationGracePeriodSeconds) * time.Second, nil
}

func (wfc *WorkflowController) workflowGarbageCollector(stopCh <-chan struct{}) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	periodicity := env.LookupEnvDurationOr("WORKFLOW_GC_PERIOD", 5*time.Minute)
	log.Infof("Performing periodic GC every %v", periodicity)
	ticker := time.NewTicker(periodicity)
	for {
		select {
		case <-stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			if wfc.offloadNodeStatusRepo.IsEnabled() {
				log.Info("Performing periodic workflow GC")
				oldRecords, err := wfc.offloadNodeStatusRepo.ListOldOffloads(wfc.GetManagedNamespace())
				if err != nil {
					log.WithField("err", err).Error("Failed to list old offloaded nodes")
					continue
				}
				log.WithField("len_wfs", len(oldRecords)).Info("Deleting old offloads that are not live")
				for uid, versions := range oldRecords {
					if err := wfc.deleteOffloadedNodesForWorkflow(uid, versions); err != nil {
						log.WithError(err).WithField("uid", uid).Error("Failed to delete old offloaded nodes")
					}
				}
				log.Info("Workflow GC finished")
			}
		}
	}
}

func (wfc *WorkflowController) deleteOffloadedNodesForWorkflow(uid string, versions []string) error {
	workflows, err := wfc.wfInformer.GetIndexer().ByIndex(indexes.UIDIndex, uid)
	if err != nil {
		return err
	}
	var wf *wfv1.Workflow
	switch l := len(workflows); l {
	case 0:
		log.WithField("uid", uid).Info("Workflow missing, probably deleted")
	case 1:
		un := workflows[0].(*unstructured.Unstructured)
		wf, err = util.FromUnstructured(un)
		if err != nil {
			return err
		}
		key := wf.ObjectMeta.Namespace + "/" + wf.ObjectMeta.Name
		wfc.workflowKeyLock.Lock(key)
		defer wfc.workflowKeyLock.Unlock(key)
		// workflow might still be hydrated
		if wfc.hydrator.IsHydrated(wf) {
			log.WithField("uid", wf.UID).Info("Hydrated workflow encountered")
			err = wfc.hydrator.Dehydrate(wf)
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("expected no more than 1 workflow, got %d", l)
	}
	for _, version := range versions {
		// skip delete if offload is live
		if wf != nil && wf.Status.OffloadNodeStatusVersion == version {
			continue
		}
		if err := wfc.offloadNodeStatusRepo.Delete(uid, version); err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) archivedWorkflowGarbageCollector(stopCh <-chan struct{}) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	periodicity := env.LookupEnvDurationOr("ARCHIVED_WORKFLOW_GC_PERIOD", 24*time.Hour)
	if wfc.Config.Persistence == nil {
		log.Info("Persistence disabled - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	if !wfc.Config.Persistence.Archive {
		log.Info("Archive disabled - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	ttl := wfc.Config.Persistence.ArchiveTTL
	if ttl == config.TTL(0) {
		log.Info("Archived workflows TTL zero - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	log.WithFields(log.Fields{"ttl": ttl, "periodicity": periodicity}).Info("Performing archived workflow GC")
	ticker := time.NewTicker(periodicity)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			log.Info("Performing archived workflow GC")
			err := wfc.wfArchive.DeleteExpiredWorkflows(time.Duration(ttl))
			if err != nil {
				log.WithField("err", err).Error("Failed to delete archived workflows")
			}
		}
	}
}

func (wfc *WorkflowController) runWorker() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	ctx := context.Background()
	for wfc.processNextItem(ctx) {
	}
}

// processNextItem is the worker logic for handling workflow updates
func (wfc *WorkflowController) processNextItem(ctx context.Context) bool {
	key, quit := wfc.wfQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfQueue.Done(key)

	obj, exists, err := wfc.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get workflow from informer")
		return true
	}
	if !exists {
		// This happens after a workflow was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}

	wfc.workflowKeyLock.Lock(key.(string))
	defer wfc.workflowKeyLock.Unlock(key.(string))

	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.WithFields(log.Fields{"key": key}).Warn("Index is not an unstructured")
		return true
	}

	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Warn("Failed to unmarshal key to workflow object")
		woc := newWorkflowOperationCtx(wf, wfc)
		woc.markWorkflowFailed(ctx, fmt.Sprintf("cannot unmarshall spec: %s", err.Error()))
		woc.persistUpdates(ctx)
		return true
	}

	if wf.Labels[common.LabelKeyCompleted] == "true" {
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return true
	}
	// this will ensure we process every incomplete workflow once every 20m
	wfc.wfQueue.AddAfter(key, workflowResyncPeriod)

	woc := newWorkflowOperationCtx(wf, wfc)

	if !wfc.throttler.Admit(key.(string)) {
		log.WithField("key", key).Info("Workflow processing has been postponed due to max parallelism limit")
		if woc.wf.Status.Phase == wfv1.WorkflowUnknown {
			woc.markWorkflowPhase(ctx, wfv1.WorkflowPending, "Workflow processing has been postponed because too many workflows are already running")
			woc.persistUpdates(ctx)
		}
		return true
	}

	// make sure this is removed from the throttler is complete
	defer func() {
		// must be done with woc
		if woc.wf.Labels[common.LabelKeyCompleted] == "true" {
			wfc.throttler.Remove(key.(string))
		}
	}()

	err = wfc.hydrator.Hydrate(woc.wf)
	if err != nil {
		woc.log.Errorf("hydration failed: %v", err)
		woc.markWorkflowError(ctx, err)
		woc.persistUpdates(ctx)
		return true
	}

	startTime := time.Now()
	woc.operate(ctx)
	wfc.metrics.OperationCompleted(time.Since(startTime).Seconds())
	if woc.wf.Status.Fulfilled() {
		// Send all completed pods to gcPods channel to delete it later depend on the PodGCStrategy.
		var doPodGC bool
		if woc.execWf.Spec.PodGC != nil {
			switch woc.execWf.Spec.PodGC.Strategy {
			case wfv1.PodGCOnWorkflowCompletion:
				doPodGC = true
			case wfv1.PodGCOnWorkflowSuccess:
				if woc.wf.Status.Successful() {
					doPodGC = true
				}
			}
		}
		if doPodGC {
			for podName := range woc.completedPods {
				delay := woc.controller.Config.GetPodGCDeleteDelayDuration()
				woc.controller.queuePodForCleanupAfter(woc.wf.Namespace, podName, deletePod, delay)
			}
		}
	}

	// TODO: operate should return error if it was unable to operate properly
	// so we can requeue the work for a later time
	// See: https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
	// c.handleErr(err, key)
	return true
}

func (wfc *WorkflowController) podWorker() {
	for wfc.processNextPodItem() {
	}
}

// processNextPodItem is the worker logic for handling pod updates.
// For pods updates, this simply means to "wake up" the workflow by
// adding the corresponding workflow key into the workflow workqueue.
func (wfc *WorkflowController) processNextPodItem() bool {
	key, quit := wfc.podQueue.Get()
	if quit {
		return false
	}
	defer wfc.podQueue.Done(key)

	obj, exists, err := wfc.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get pod from informer index")
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}

	err = wfc.enqueueWfFromPodLabel(obj)
	if err != nil {
		log.WithError(err).Warnf("Failed to enqueue the workflow for %s", key)
	}
	return true
}

// enqueueWfFromPodLabel will extract the workflow name from pod label and
// enqueue workflow for processing
func (wfc *WorkflowController) enqueueWfFromPodLabel(obj interface{}) error {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		return fmt.Errorf("Key in index is not a pod")
	}
	if pod.Labels == nil {
		return fmt.Errorf("Pod did not have labels")
	}
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		return fmt.Errorf("Watch returned pod unrelated to any workflow")
	}
	wfc.wfQueue.AddRateLimited(pod.ObjectMeta.Namespace + "/" + workflowName)
	return nil
}

func (wfc *WorkflowController) tweakListOptions(options *metav1.ListOptions) {
	labelSelector := labels.NewSelector().
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))
	options.LabelSelector = labelSelector.String()
}

func getWfPriority(obj interface{}) (int32, time.Time) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return 0, time.Now()
	}
	priority, hasPriority, err := unstructured.NestedInt64(un.Object, "spec", "priority")
	if err != nil {
		return 0, un.GetCreationTimestamp().Time
	}
	if !hasPriority {
		priority = 0
	}

	return int32(priority), un.GetCreationTimestamp().Time
}

func (wfc *WorkflowController) addWorkflowInformerHandlers(ctx context.Context) {
	wfc.wfInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				return !common.UnstructuredHasCompletedLabel(obj)
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						// for a new workflow, we do not want to rate limit its execution using AddRateLimited
						wfc.wfQueue.AddAfter(key, wfc.Config.InitialDelay.Duration)
						priority, creation := getWfPriority(obj)
						wfc.throttler.Add(key, priority, creation)
					}
				},
				UpdateFunc: func(old, new interface{}) {
					oldWf, newWf := old.(*unstructured.Unstructured), new.(*unstructured.Unstructured)
					// this check is very important to prevent doing many reconciliations we do not need to do
					if oldWf.GetResourceVersion() == newWf.GetResourceVersion() {
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						wfc.wfQueue.AddRateLimited(key)
						priority, creation := getWfPriority(new)
						wfc.throttler.Add(key, priority, creation)
					}
				},
				DeleteFunc: func(obj interface{}) {
					// IndexerInformer uses a delta queue, therefore for deletes we have to use this
					// key function.
					key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
					if err == nil {
						wfc.releaseAllWorkflowLocks(obj)
						// no need to add to the queue - this workflow is done
						wfc.throttler.Remove(key)
					}
				},
			},
		},
	)
	wfc.wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			// no need to check the `common.LabelKeyCompleted` as we already know it must be complete
			return ok && un.GetLabels()[common.LabelKeyWorkflowArchivingStatus] == "Pending"
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				wfc.archiveWorkflow(ctx, obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				wfc.archiveWorkflow(ctx, obj)
			},
		},
	},
	)
	wfc.wfInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			wf, ok := obj.(*unstructured.Unstructured)
			if ok { // maybe cache.DeletedFinalStateUnknown
				wfc.metrics.StopRealtimeMetricsForKey(string(wf.GetUID()))
			}
		},
	})
}

func (wfc *WorkflowController) archiveWorkflow(ctx context.Context, obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed to get key for object")
		return
	}
	wfc.workflowKeyLock.Lock(key)
	defer wfc.workflowKeyLock.Unlock(key)
	err = wfc.archiveWorkflowAux(ctx, obj)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("failed to archive workflow")
	}
}

func (wfc *WorkflowController) archiveWorkflowAux(ctx context.Context, obj interface{}) error {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return fmt.Errorf("failed to convert to workflow from unstructured: %w", err)
	}
	err = wfc.hydrator.Hydrate(wf)
	if err != nil {
		return fmt.Errorf("failed to hydrate workflow: %w", err)
	}
	log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name, "uid": wf.UID}).Info("archiving workflow")
	err = wfc.wfArchive.ArchiveWorkflow(wf)
	if err != nil {
		return fmt.Errorf("failed to archive workflow: %w", err)
	}
	data, err := json.Marshal(map[string]interface{}{
		"metadata": metav1.ObjectMeta{
			Labels: map[string]string{
				common.LabelKeyWorkflowArchivingStatus: "Archived",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	_, err = wfc.wfclientset.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(
		ctx,
		un.GetName(),
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	if err != nil {
		// from this point on we have successfully archived the workflow, and it is possible for the workflow to have actually
		// been deleted, so it's not a problem to get a `IsNotFound` error
		if apierr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to archive workflow: %w", err)
	}
	return nil
}

func (wfc *WorkflowController) newWorkflowPodWatch(ctx context.Context) *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().Pods(wfc.GetManagedNamespace())
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		return c.List(ctx, options)
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = labelSelector.String()
		return c.Watch(ctx, options)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newPodInformer(ctx context.Context) cache.SharedIndexInformer {
	source := wfc.newWorkflowPodWatch(ctx)
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{
		indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
		indexes.PodPhaseIndex: indexes.PodPhaseIndexFunc,
	})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					return
				}
				wfc.podQueue.Add(key)
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err != nil {
					return
				}
				oldPod, newPod := old.(*apiv1.Pod), new.(*apiv1.Pod)
				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				if !pod.SignificantPodChange(oldPod, newPod) {
					log.WithField("key", key).Info("insignificant pod change")
					diff.LogChanges(oldPod, newPod)
					return
				}
				wfc.podQueue.Add(key)
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.

				// Enqueue the workflow for deleted pod
				_ = wfc.enqueueWfFromPodLabel(obj)
			},
		},
	)
	return informer
}

// call this func whenever the configuration changes, or when the workflow informer changes
func (wfc *WorkflowController) updateEstimatorFactory() {
	wfc.estimatorFactory = estimation.NewEstimatorFactory(wfc.wfInformer, wfc.hydrator, wfc.wfArchive)
}

// setWorkflowDefaults sets values in the workflow.Spec with defaults from the
// workflowController. Values in the workflow will be given the upper hand over the defaults.
// The defaults for the workflow controller are set in the workflow-controller config map
func (wfc *WorkflowController) setWorkflowDefaults(wf *wfv1.Workflow) error {
	if wfc.Config.WorkflowDefaults != nil {
		err := util.MergeTo(wfc.Config.WorkflowDefaults, wf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) GetManagedNamespace() string {
	if wfc.managedNamespace != "" {
		return wfc.managedNamespace
	}
	return wfc.Config.Namespace
}

func (wfc *WorkflowController) GetContainerRuntimeExecutor(labels labels.Labels) string {
	if wfc.containerRuntimeExecutor != "" {
		return wfc.containerRuntimeExecutor
	}
	executor, err := wfc.Config.GetContainerRuntimeExecutor(labels)
	if err != nil {
		log.WithError(err).Info("failed to determine container runtime executor")
	}
	return executor
}

func (wfc *WorkflowController) getMetricsServerConfig() (metrics.ServerConfig, metrics.ServerConfig) {
	// Metrics config
	path := wfc.Config.MetricsConfig.Path
	if path == "" {
		path = metrics.DefaultMetricsServerPath
	}
	port := wfc.Config.MetricsConfig.Port
	if port == 0 {
		port = metrics.DefaultMetricsServerPort
	}
	metricsConfig := metrics.ServerConfig{
		Enabled:      wfc.Config.MetricsConfig.Enabled == nil || *wfc.Config.MetricsConfig.Enabled,
		Path:         path,
		Port:         port,
		TTL:          time.Duration(wfc.Config.MetricsConfig.MetricsTTL),
		IgnoreErrors: wfc.Config.MetricsConfig.IgnoreErrors,
	}

	// Telemetry config
	path = metricsConfig.Path
	if wfc.Config.TelemetryConfig.Path != "" {
		path = wfc.Config.TelemetryConfig.Path
	}

	port = metricsConfig.Port
	if wfc.Config.TelemetryConfig.Port > 0 {
		port = wfc.Config.TelemetryConfig.Port
	}
	telemetryConfig := metrics.ServerConfig{
		Enabled:      wfc.Config.TelemetryConfig.Enabled == nil || *wfc.Config.TelemetryConfig.Enabled,
		Path:         path,
		Port:         port,
		IgnoreErrors: wfc.Config.TelemetryConfig.IgnoreErrors,
	}
	return metricsConfig, telemetryConfig
}

func (wfc *WorkflowController) releaseAllWorkflowLocks(obj interface{}) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.WithFields(log.Fields{"key": obj}).Warn("Key in index is not an unstructured")
		return
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.WithFields(log.Fields{"key": obj}).Warn("Invalid workflow object")
		return
	}
	if wf.Status.Synchronization != nil {
		wfc.syncManager.ReleaseAll(wf)
	}
}

func (wfc *WorkflowController) isArchivable(wf *wfv1.Workflow) bool {
	return wfc.archiveLabelSelector.Matches(labels.Set(wf.Labels))
}

func (wfc *WorkflowController) syncWorkflowPhaseMetrics() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	for _, phase := range []wfv1.NodePhase{wfv1.NodePending, wfv1.NodeRunning, wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError} {
		keys, err := wfc.wfInformer.GetIndexer().IndexKeys(indexes.WorkflowPhaseIndex, string(phase))
		errors.CheckError(err)
		wfc.metrics.SetWorkflowPhaseGauge(phase, len(keys))
	}
	for _, x := range []wfv1.Condition{
		{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionTrue},
		{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse},
	} {
		keys, err := wfc.wfInformer.GetIndexer().IndexKeys(indexes.ConditionsIndex, indexes.ConditionValue(x))
		errors.CheckError(err)
		metrics.WorkflowConditionMetric.WithLabelValues(string(x.Type), string(x.Status)).Set(float64(len(keys)))
	}
}

func (wfc *WorkflowController) syncPodPhaseMetrics() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	for _, phase := range []apiv1.PodPhase{apiv1.PodRunning, apiv1.PodPending} {
		objs, err := wfc.podInformer.GetIndexer().IndexKeys(indexes.PodPhaseIndex, string(phase))
		if err != nil {
			log.WithError(err).Error("failed to list active pods")
			return
		}
		wfc.metrics.SetPodPhaseGauge(phase, len(objs))
	}
}
