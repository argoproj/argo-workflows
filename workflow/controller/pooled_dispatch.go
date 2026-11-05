package controller

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/propagation"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4"
	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

// executePooled is the dispatch seam of a pooled node: a container or script
// node whose template, with the templateDefaults merged in, has a pool
// (pool.Pooled). Instead of creating a pod it enqueues the node's task as a
// WorkflowTaskSet for a hot worker pool and records the node's ledger
// (NodeStatus.PooledTask). Everything above it (retries, constraints,
// synchronization, memoization) treats the node as a pod's, and the guards
// are a pod's: the shutdown strategy, the workflow deadline and the resource
// rate limiter. A node that can't be pooled fails; it never falls back to a
// pod, which would surprise its user. Once the ledger is recorded,
// reconciling the node makes no API calls, as reconciling a node whose pod
// exists makes none, until its queued task is past a timeout
// (checkPooledTimeouts), or another controller version enqueued it
// (replaceOtherVersionTask).
func (woc *wfOperationCtx) executePooled(ctx context.Context, node *wfv1.NodeStatus, templateScope string, mainCtr apiv1.Container, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	replaced := false
	if node.PooledTask != nil {
		replaced = woc.replaceOtherVersionTask(ctx, node)
		var err error
		if node, err = woc.wf.Status.Nodes.Get(node.ID); err != nil {
			return nil, err
		}
		if !replaced {
			return woc.checkPooledTimeouts(ctx, node, tmpl), nil
		}
	}
	ctx = woc.controller.tracing.ContextWithNode(ctx, woc.wf.Name, woc.wf.Namespace, node.ID)

	if !woc.GetShutdownStrategy().ShouldExecute(opts.onExitTemplate) {
		woc.markNodeFailedOnShutdown(ctx, node.Name, notStartedOnShutdownMessage(woc.GetShutdownStrategy()))
		return node, nil
	}
	// As the controller started: like the pool controller, dispatch takes a
	// restart to follow a change to the flag.
	if !woc.controller.hotWorkersEnabled() {
		return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "template '%s' has a pool, but hot workers are disabled: set hotWorkers.enabled in the workflow controller configmap and restart the controller", tmpl.Name)
	}
	// Validation at submission sees the template as written; only now are its
	// pool name, volume mounts and artifact paths substituted, and is the
	// execution workflow's spec known.
	if err := validate.Poolable(tmpl, &woc.execWf.Spec); err != nil {
		return nil, err
	}

	includeScriptOutput, err := woc.includeScriptOutput(ctx, node.Name, opts.boundaryID)
	if err != nil {
		return node, err
	}
	pb := woc.newPodBuilder(ctx, node.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{
		includeScriptOutput: includeScriptOutput,
		onExitPod:           opts.onExitTemplate,
		executionDeadline:   opts.executionDeadline,
	})
	task, err := pb.buildPooledTask(ctx, woc.poolWorkflowTemplateName(orgTmpl, templateScope))
	if err != nil {
		return woc.requeueIfTransientErr(ctx, err, node.Name)
	}
	if task == nil {
		// As for a pod: the workflow deadline has passed, so the task isn't
		// started, and failNodesWithoutCreatedPodsAfterDeadlineOrShutdown ends it.
		return node, nil
	}
	// As for a pod that exists, a TaskSet Enqueue would adopt, as after a lost
	// status write, takes no slot: it creates nothing.
	if !woc.pooledTaskSetAdoptable(node.ID) {
		if err = woc.reserveRateLimiter(ctx); err != nil {
			return woc.requeueIfTransientErr(ctx, err, node.Name)
		}
	}

	enqueued, err := woc.controller.dispatcher.Enqueue(ctx, *task)
	switch {
	case errors.Is(err, dispatch.ErrFieldsDropped):
		// The TaskSet exists, so the ledger records it to be deleted with the
		// node's others; no worker can run it.
		woc.recordPooledTask(ctx, node.ID, task, enqueued.TaskSet)
		return nil, err
	case errors.Is(err, dispatch.ErrNotFound), errors.Is(err, dispatch.ErrStaleTaskSet):
		// A TaskSet of that name was deleted between the create that found it
		// and the read that would have adopted it, or one left by an earlier
		// workflow of the same name is still being deleted: enqueue again.
		woc.requeue()
		return woc.markNodePending(ctx, node.Name, err), nil
	case err != nil:
		// Including ErrNameTaken: the TaskSet isn't a pooled task, so the
		// node fails without a ledger and the TaskSet is never touched.
		return woc.requeueIfTransientErr(ctx, err, node.Name)
	}
	node = woc.recordPooledTask(ctx, node.ID, task, enqueued.TaskSet)
	woc.log.WithFields(logging.Fields{
		"nodeID":     node.ID,
		"taskSet":    enqueued.TaskSet.Name,
		"taskSetUID": enqueued.TaskSet.UID,
		"pool":       node.PooledTask.PoolName,
		"poolHash":   node.PooledTask.PoolHash,
		"adopted":    enqueued.Adopted,
	}).Info(ctx, "Enqueued pooled task")
	// Counted whether created or adopted: this operation counted the active
	// pods from its nodes as they were when it started, when this node had
	// no ledger, so an adopted TaskSet, whose ledger a lost write dropped,
	// wasn't counted either. A replaced task's node had its ledger then, and
	// was counted.
	if !replaced {
		woc.incrementActivePods()
	}
	return node, nil
}

// replaceOtherVersionTask drops the queued task of the unended pooled node,
// and with it the node's ledger, if another controller version enqueued it,
// so that the node is enqueued again with this version's spec.pool: a
// controller never builds another version's pool, whose pod template might
// not suit its executor image. It reports whether it did. Only a task that
// no worker has claimed, as the caches show it, is dropped, with hot workers
// enabled and the workflow not shutting down, which would end the task
// instead; a task a worker has released stays on its own version's pool,
// which it has already run on. Its TaskSet is deleted, then its claim read
// live (Dispatcher.Drop); a claim that raced the drop is settled by
// settleRacedClaim, and the task is enqueued again only if it never ran.
// Dropping the ledger outright, rather than through its setters, starts the
// task's hand-off afresh, as a retry's reset does.
func (woc *wfOperationCtx) replaceOtherVersionTask(ctx context.Context, node *wfv1.NodeStatus) bool {
	if !waitingForClaim(node) || !woc.controller.hotWorkersEnabled() || woc.GetShutdownStrategy().Enabled() {
		return false
	}
	ref := pooledTaskSetRef(node)
	key := woc.pooledKey(node)
	taskSet, ok := woc.cachedPooledTaskSet(ref)
	version := argo.GetVersion().Version
	if !ok || taskSet.GetAnnotations()[common.AnnotationKeyPoolControllerVersion] == version {
		return false
	}
	if _, claimed := taskSetClaim(woc.controller.taskResultInformer.GetStore(), woc.wf.Namespace, ref); claimed {
		return false
	}
	log := woc.log.WithFields(logging.Fields{
		"nodeID":                   node.ID,
		"taskSetUID":               ref.UID,
		"taskSetControllerVersion": taskSet.GetAnnotations()[common.AnnotationKeyPoolControllerVersion],
		"controllerVersion":        version,
	})
	found, err := woc.controller.dispatcher.Drop(ctx, key, ref.UID)
	if err != nil {
		log.WithError(err).Warn(ctx, "Failed to drop a pooled task another controller version enqueued")
		woc.requeue()
		return false
	}
	if found != nil && !woc.settleRacedClaim(ctx, node, *found, log) {
		return false
	}
	woc.controller.pooledTaskSetsGone.record(woc.wf.UID, ref)
	updated := node.DeepCopy()
	updated.PooledTask = nil
	woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
	woc.updated = true
	log.Info(ctx, "Enqueueing again a pooled task another controller version enqueued")
	return true
}

// racedClaimMessage ends a node whose task a worker claimed as the task was
// being moved to this controller version's pool.
const racedClaimMessage = "a worker claimed the task as it was moved to this controller version's pool"

// settleRacedClaim settles found, a claim a worker made on the node's queued
// task as replaceOtherVersionTask dropped it, as a drop for a shutdown does
// (applyPooledEnding), and reports whether the task never ran, so can be
// enqueued again. A completed claim is the node's ending. An incomplete one is
// interrupted: if it is already gone, its worker found the TaskSet deleted
// and withdrew without running the task; otherwise the task may have
// started, so the node ends Error once it has stopped, for a retry to run it
// afresh. If the interrupt fails, the claim is recorded, and the task runs on
// to its end on the other version's worker, as any claimed task does.
func (woc *wfOperationCtx) settleRacedClaim(ctx context.Context, node *wfv1.NodeStatus, found dispatch.FoundClaim, log logging.Logger) bool {
	claim := found.Claim
	log = log.WithField("claimUID", claim.UID)
	if claim.Completed {
		log.Info(ctx, "A worker completed a pooled task as it was dropped for another controller version")
		woc.applyFoundCompletedClaim(ctx, node, found)
		woc.updated = true
		return false
	}
	err := woc.controller.dispatcher.Interrupt(ctx, woc.pooledKey(node), node.PooledTask.TaskSetUID, claim.UID)
	switch {
	case errors.Is(err, dispatch.ErrClaimLost):
		log.Info(ctx, "A worker claimed a pooled task as it was dropped for another controller version, and withdrew")
		return true
	case err != nil:
		log.WithError(err).Warn(ctx, "Failed to interrupt the claim on a pooled task dropped for another controller version; leaving it to run")
		woc.recordFoundClaim(ctx, node, claim)
		woc.updated = true
		return false
	}
	log.Info(ctx, "Interrupted a pooled task a worker claimed as it was dropped for another controller version")
	woc.endInterruptedNode(ctx, woc.recordFoundClaim(ctx, node, claim), false, wfv1.NodeError, racedClaimMessage)
	return false
}

// poolWorkflowTemplateName is the WorkflowTemplate or ClusterWorkflowTemplate
// the template of a pooled node comes from, which its default pool name
// starts with: orgTmpl's templateRef's, or else that of the template
// scope orgTmpl was resolved in, or else the workflow's workflowTemplateRef's.
// It is empty for a template of the workflow's own.
func (woc *wfOperationCtx) poolWorkflowTemplateName(orgTmpl wfv1.TemplateReferenceHolder, templateScope string) string {
	if ref := orgTmpl.GetTemplateRef(); ref != nil {
		return ref.Name
	}
	scope, name := wfv1.NodeStatus{TemplateScope: templateScope}.GetTemplateScope()
	if scope == wfv1.ResourceScopeNamespaced || scope == wfv1.ResourceScopeCluster {
		return name
	}
	if ref := woc.execWf.Spec.WorkflowTemplateRef; ref != nil {
		return ref.Name
	}
	return ""
}

// recordPooledTask records the ledger of the node with nodeID for the TaskSet
// its task was enqueued as. Pool, task name and deadlines are the TaskSet's,
// which for an adopted TaskSet were resolved by the reconcile that created
// it; task gives them only where the TaskSet was stored without them. It
// clears the node's message, such as an earlier transient error: a pod's
// would be replaced by its status, but a pooled node can wait long for a
// worker.
func (woc *wfOperationCtx) recordPooledTask(ctx context.Context, nodeID string, task *dispatch.Task, taskSet *wfv1.WorkflowTaskSet) *wfv1.NodeStatus {
	node, err := woc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		woc.log.WithField("nodeID", nodeID).WithPanic().Error(ctx, "was unable to obtain node for nodeID")
	}
	d := &task.Dispatch
	if taskSet.Spec.Dispatch != nil {
		d = taskSet.Spec.Dispatch
	}
	ledger := &wfv1.PooledTaskStatus{
		PoolName:              taskSet.Labels[common.LabelKeyPoolName],
		PoolHash:              taskSet.Labels[common.LabelKeyPoolHash],
		TaskName:              d.TaskName,
		TaskSetUID:            taskSet.UID,
		Deadline:              d.Deadline.DeepCopy(),
		ActiveDeadlineSeconds: d.ActiveDeadlineSeconds,
	}
	// The controller's clock, as the node's own times are.
	ledger.Enqueue(metav1.Now())
	node.PooledTask = ledger
	node.Message = ""
	woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
	woc.updated = true
	return node
}

// buildPooledTask builds what a pooled node's task is enqueued with from the
// pod builder's snapshot, as build builds a pod from it: the template as its
// pod's executor would get it, spec.pool and its hash (buildPoolSpec), and
// spec.dispatch, the per-task settings a pod gets from its spec and
// environment. workflowTemplateName is where the template comes from, for its
// default pool name. It returns nil when the workflow deadline has passed, so
// the task must not start. Like build, it creates nothing; it may look up the
// image's entrypoint and the executor's token Secret.
func (pb *podBuilder) buildPooledTask(ctx context.Context, workflowTemplateName string) (*dispatch.Task, error) {
	in := &pb.in
	opts := in.opts
	ctx = logging.WithLogger(ctx, in.log.WithFields(logging.Fields{"nodeName": in.nodeName, "nodeID": in.nodeID}))
	tmpl := in.tmpl.DeepCopy()
	wfSpec := in.execWfSpec

	// The workflow deadline is part of the absolute deadline; the worker
	// counts the template's activeDeadlineSeconds from the claim, as a pod's is
	// counted from its start.
	if _, start, err := taskActiveDeadlineSeconds(tmpl, in.computedWorkflowDeadline, opts.onExitPod, in.now); err != nil || !start {
		return nil, err
	}
	activeDeadlineSeconds, err := intstr.Int64(tmpl.ActiveDeadlineSeconds)
	if err != nil {
		return nil, err
	}
	mainCtrs := make([]apiv1.Container, len(in.mainCtrs))
	for i := range in.mainCtrs {
		in.mainCtrs[i].DeepCopyInto(&mainCtrs[i])
	}
	if err = applyMainContainerDefaults(mainCtrs, tmpl, in.mainContainerDefaults); err != nil {
		return nil, err
	}
	if debugPauseRequested(mainCtrs) {
		activeDeadlineSeconds = nil
	}
	node, err := pb.deps.getNodeByName(in.nodeName)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).Warn(ctx, "couldn't retrieve node, will get nil templateDeadline")
	}
	timeoutDeadline, _, err := templateTimeoutDeadlines(tmpl, node, in.now)
	if err != nil {
		return nil, err
	}

	command, err := pb.pooledCommand(ctx, mainCtrs[0], tmpl)
	if err != nil {
		return nil, err
	}

	// The template carries its archive location, with whether to archive its
	// logs, as a pod's executor gets it.
	addArchiveLocation(ctx, tmpl, in.artifactRepository, isArchiveLogs(in.artifactRepository, wfSpec, tmpl))
	populateAllPluginArtifactTimeouts(tmpl, in.config)
	taskName := util.GeneratePodName(in.wfName, in.nodeName, tmpl.Name, in.nodeID, in.podNameVersion)
	tmpl, err = substituteTaskParams(ctx, tmpl, in.globalParams, taskName)
	if err != nil {
		return nil, err
	}
	// As build checks the archive location, whose key format comes from the
	// repository and is resolved only now.
	if err = validate.VerifyResolvedVariables(tmpl.ArchiveLocation); err != nil {
		return nil, err
	}

	x := poolSpecExtras{
		executorLogArgs:      pb.deps.getExecutorLogOpts(ctx),
		workflowTemplateName: workflowTemplateName,
		controllerVersion:    argo.GetVersion().Version,
	}
	if name := executorServiceAccountName(in.tmpl, wfSpec); name != "" {
		if x.executorTokenSecretName, err = pb.deps.getServiceAccountTokenName(ctx, name); err != nil {
			return nil, err
		}
	}
	spec, err := buildPoolSpec(ctx, in, x)
	if err != nil {
		return nil, err
	}
	if err = pb.lookupPoolPluginCommands(ctx, spec); err != nil {
		return nil, err
	}
	hash, err := poolHash(spec)
	if err != nil {
		return nil, err
	}

	d := wfv1.TaskDispatch{
		TaskName:              taskName,
		IncludeScriptOutput:   opts.includeScriptOutput,
		InputArtifactPlugins:  tmpl.Inputs.Artifacts.GetPluginNames(ctx, in.artifactRepository, wfv1.ExcludeLogs, tmpl.ArchiveLocation),
		Command:               command,
		Deadline:              pooledTaskDeadline(taskDeadline(in.computedWorkflowDeadline, opts.executionDeadline, opts.onExitPod), timeoutDeadline),
		ActiveDeadlineSeconds: activeDeadlineSeconds,
		TraceContext:          traceContext(ctx),
	}
	// As for a pod, progress is reported only when both ticks are set.
	if in.progressPatchTickDuration != 0 && in.progressFileTickDuration != 0 {
		d.ProgressPatchTickDuration = &metav1.Duration{Duration: in.progressPatchTickDuration}
		d.ProgressFileTickDuration = &metav1.Duration{Duration: in.progressFileTickDuration}
	}
	return &dispatch.Task{
		Key:          dispatch.Key{Namespace: in.namespace, NodeID: in.nodeID},
		WorkflowName: in.wfName,
		WorkflowUID:  in.execCfg.workflowUID,
		OnExit:       opts.onExitPod,
		PoolHash:     hash,
		Template:     *tmpl,
		Pool:         *spec,
		Dispatch:     d,
	}, nil
}

// pooledCommand is the command a pooled task's main container c, with the
// controller's main container defaults merged in, runs, resolved as build
// resolves a pod's: nil when c has a command; otherwise the image's
// entrypoint, followed by the image's cmd when the task has no arguments of
// its own, as a pod gets its cmd as its args. A script with a source has its
// script as its argument, which the worker supplies.
func (pb *podBuilder) pooledCommand(ctx context.Context, c apiv1.Container, tmpl *wfv1.Template) ([]string, error) {
	if len(c.Command) > 0 {
		return nil, nil
	}
	image, err := lookupEntrypoint(ctx, pb.deps.lookupImage, c.Image, pb.in.namespace, pb.in.execWfSpec)
	if err != nil {
		return nil, err
	}
	command := append([]string{}, image.Entrypoint...)
	if c.Args == nil && (tmpl.Script == nil || tmpl.Script.Source == "") {
		command = append(command, image.Cmd...)
	}
	return command, nil
}

// lookupPoolPluginCommands sets the Command of each of spec's artifact plugins
// to its image's entrypoint, looked up as a pod's plugin containers look it up
// (see artifactContainer), so spec.pool holds everything the pool's
// Deployment is built from.
func (pb *podBuilder) lookupPoolPluginCommands(ctx context.Context, spec *wfv1.PoolSpec) error {
	for i := range spec.ArtifactPlugins {
		p := &spec.ArtifactPlugins[i]
		image, err := lookupEntrypoint(ctx, pb.deps.lookupImage, p.Image, pb.in.namespace, pb.in.execWfSpec)
		if err != nil {
			return err
		}
		p.Command = image.Entrypoint
	}
	return nil
}

// substituteTaskParams substitutes a pooled task's template as
// substitutePodParams substitutes a pod, with taskName as its task name. A
// pooled task has no pod of its own, so its template is substituted without
// pod.name, and an expression that needs it fails rather than reaching the
// worker unresolved. Its archive location, whose key format comes from the
// artifact repository, resolves pod.name as the task name, as a pod's does.
// The template is the executor's (common.ExecutorTemplate), as a pod's
// ARGO_TEMPLATE is: its inputs.parameters, already substituted into it, are
// data, and a value that holds a {{pod...}} tag must not fail the task.
func substituteTaskParams(ctx context.Context, tmpl *wfv1.Template, globalParams common.Parameters, taskName string) (*wfv1.Template, error) {
	params := taskParams(globalParams, tmpl, taskName)
	var archiveLocation *wfv1.ArtifactLocation
	if tmpl.ArchiveLocation != nil {
		var err error
		if archiveLocation, err = substituteTaskObject(ctx, tmpl.ArchiveLocation, params, nil); err != nil {
			return nil, err
		}
	}
	withoutPod := params.DeepCopy()
	delete(withoutPod, varkeys.PodName.Template())
	rest := common.ExecutorTemplate(tmpl)
	rest.ArchiveLocation = nil
	substituted, err := substituteTaskObject(ctx, rest, withoutPod, []string{"pod"})
	if err != nil {
		return nil, err
	}
	substituted.ArchiveLocation = archiveLocation
	return substituted, nil
}

// pooledTaskDeadline is a pooled task's absolute deadline: the earlier of
// the deadline a pod's executor enforces itself (taskDeadline, zero for none)
// and the template timeout's, which a pod gets through its
// activeDeadlineSeconds. Nil means none.
func pooledTaskDeadline(executorDeadline time.Time, timeoutDeadline *time.Time) *metav1.Time {
	var executor *time.Time
	if !executorDeadline.IsZero() {
		executor = &executorDeadline
	}
	if deadline := earliest(executor, timeoutDeadline); deadline != nil {
		return &metav1.Time{Time: *deadline}
	}
	return nil
}

// traceContext is the W3C trace context of ctx, which a pod gets as
// TRACEPARENT and TRACESTATE, or nil when ctx has none.
func traceContext(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}
	return carrier
}
