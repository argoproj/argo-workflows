package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	wfsync "github.com/argoproj/argo-workflows/v4/workflow/sync"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// prepareNode returns the node named nodeName, if it exists, with the
// display name of processedTmpl's workflows.argoproj.io/display-name
// annotation applied (or marked Error if that name is invalid).
func (woc *wfOperationCtx) prepareNode(ctx context.Context, nodeName string, tmplCtx *templateresolution.TemplateContext, processedTmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, nodeFlag *wfv1.NodeFlag) (*wfv1.NodeStatus, error) {
	// A missing node will be initialized via woc.initializeNodeOrMarkError
	node, _ := woc.wf.GetNodeByName(nodeName)

	if displayName := processedTmpl.GetDisplayName(); node != nil && displayName != "" {
		if !displayNameRegex.MatchString(displayName) {
			return woc.initializeNodeOrMarkError(ctx, node, nodeName, tmplCtx.GetTemplateScope(), orgTmpl, boundaryID, nodeFlag, fmt.Errorf("displayName must match the regex %s", displayNameRegex.String())), fmt.Errorf("displayName must match the regex %s", displayNameRegex.String())
		}

		woc.log.WithFields(logging.Fields{"nodeName": nodeName, "displayName": displayName}).Debug(ctx, "Updating node display name")
		woc.setNodeDisplayName(ctx, node, displayName)
	}
	return node, nil
}

// checkConstraints checks deadline and parallelism.
func (woc *wfOperationCtx) checkConstraints(ctx context.Context, nodeName string, node *wfv1.NodeStatus, processedTmpl *wfv1.Template, boundaryID string) error {
	if woc.deadlineExceeded() {
		woc.log.Warn(ctx, "Deadline exceeded")
		woc.requeue()
		return ErrDeadlineExceeded
	}

	// Check the timeout and pendingTimeout deadlines for Pending nodes.
	// This also covers the resource-forbidden and synchronization scenarios,
	// where only the node exists, in a pending state, with no pod created.
	deadline, pendingDeadline, err := woc.checkTemplateTimeouts(processedTmpl, node, time.Now().UTC())
	if err != nil {
		woc.log.WithField("template", processedTmpl.Name).Warn(ctx, "Template exceeded its deadline")
		if node != nil && node.Type == wfv1.NodeTypePod {
			// delete the timed-out pod so the resources it was waiting for are freed.
			// Deletion is by UID so a pod recreated by a retry cannot be affected.
			if pod, exists, podErr := woc.podExists(node.ID); podErr != nil {
				woc.log.WithError(podErr).Warn(ctx, "failed to check pod existence while cleaning up timed-out node")
			} else if exists {
				woc.controller.PodController.DeletePodByUID(ctx, pod.Namespace, pod.Name, string(pod.UID))
			}
		}
		phase := wfv1.NodeFailed
		if node != nil && node.NodeFlag != nil && node.NodeFlag.Hooked {
			// A timed-out hook errored rather than failed: like a
			// hook that could not run, it ends its task's boundary Error.
			phase = wfv1.NodeError
		}
		_ = woc.markNodePhase(ctx, nodeName, phase, err.Error())
		return err
	}
	// Ensure that we will check again soon after the earliest deadline
	if deadline == nil || (pendingDeadline != nil && pendingDeadline.Before(*deadline)) {
		deadline = pendingDeadline
	}
	if deadline != nil && time.Now().Before(*deadline) {
		woc.requeueAfter(time.Until(*deadline))
	}

	if err := woc.checkParallelism(ctx, processedTmpl, node, boundaryID); err != nil {
		return err
	}
	return nil
}

// handleSynchronization attempts to acquire the lock.
func (woc *wfOperationCtx) handleSynchronization(ctx context.Context, nodeName string, node *wfv1.NodeStatus, processedTmpl *wfv1.Template, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, opts *executeTemplateOpts) (bool, *wfv1.NodeStatus, error) {
	if processedTmpl.Synchronization == nil {
		return false, node, nil
	}

	lockNodeID := woc.wf.ResolveNodeID(nodeName)
	lockCtx, lockSpan := woc.controller.tracing.StartTryAcquireLock(ctx, lockNodeID, false)
	lockAcquired, wfUpdated, msg, failedLockName, err := woc.controller.syncManager.TryAcquire(lockCtx, woc.wf, lockNodeID, processedTmpl.Synchronization)
	lockSpan.SetAttributes(attribute.Bool("LockAcquired", lockAcquired))
	lockSpan.End()
	if err != nil {
		if wfsync.IsRetryableSyncError(err) || errorsutil.IsTransientErr(ctx, err) {
			// Transient failure in the lock backend: leave the node pending
			// and try again on a later reconcile instead of erroring it.
			woc.requeue()
			if node == nil {
				_, node = woc.initializeExecutableNode(ctx, nodeName, wfutil.GetNodeType(processedTmpl), templateScope, processedTmpl, orgTmpl, boundaryID, wfv1.NodePending, opts.nodeFlag, false, err.Error())
			}
			return false, node, nil
		}
		return false, woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, boundaryID, opts.nodeFlag, err), err
	}
	woc.updated = woc.updated || wfUpdated

	if !lockAcquired {
		if node == nil {
			_, node = woc.initializeExecutableNode(ctx, nodeName, wfutil.GetNodeType(processedTmpl), templateScope, processedTmpl, orgTmpl, boundaryID, wfv1.NodePending, opts.nodeFlag, false, msg)
		}
		woc.log.WithField("lockName", failedLockName).Info(ctx, "Could not acquire lock")
		n, lockErr := woc.markNodeWaitingForLock(ctx, node.Name, failedLockName, msg)
		return false, n, lockErr
	}

	woc.log.WithField("nodeName", nodeName).Info(ctx, "Node acquired synchronization lock")
	if node != nil {
		node, err = woc.markNodeWaitingForLock(ctx, node.Name, "", "")
		if err != nil {
			woc.log.WithField("node.Name", node.Name).WithField("lockName", "").Error(ctx, "markNodeWaitingForLock returned err")
			return true, nil, err
		}
	}
	return true, node, nil
}

// handleMemoization checks the cache.
func (woc *wfOperationCtx) handleMemoization(ctx context.Context, nodeName string, node *wfv1.NodeStatus, processedTmpl *wfv1.Template, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, nodeFlag *wfv1.NodeFlag, unlockedNode bool) (bool, *wfv1.NodeStatus, error) {
	if processedTmpl.Memoize == nil {
		return false, node, nil
	}

	if node == nil || unlockedNode {
		memoizationCache := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, processedTmpl.Memoize.Cache.ConfigMap.Name)
		if memoizationCache == nil {
			err := fmt.Errorf("cache could not be found or created")
			woc.log.WithFields(logging.Fields{"cacheName": processedTmpl.Memoize.Cache.ConfigMap.Name}).WithError(err).Error(ctx, "memoization cache could not be found or created")
			return true, woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, boundaryID, nodeFlag, err), err
		}

		entry, err := memoizationCache.Load(ctx, processedTmpl.Memoize.Key)
		if err != nil {
			return true, woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, boundaryID, nodeFlag, err), err
		}

		hit := entry.Hit()
		var outputs *wfv1.Outputs
		if processedTmpl.Memoize.MaxAge != "" {
			maxAge, err := time.ParseDuration(processedTmpl.Memoize.MaxAge)
			if err != nil {
				err = fmt.Errorf("invalid maxAge: %w", err)
				return true, woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, boundaryID, nodeFlag, err), err
			}
			maxAgeOutputs, ok := entry.GetOutputsWithMaxAge(maxAge)
			if !ok {
				hit = false
			}
			outputs = maxAgeOutputs
		} else {
			outputs = entry.GetOutputs()
		}

		memoizationStatus := &wfv1.MemoizationStatus{
			Hit:       hit,
			Key:       processedTmpl.Memoize.Key,
			CacheName: processedTmpl.Memoize.Cache.ConfigMap.Name,
		}
		if hit {
			if node == nil {
				_, node = woc.initializeCacheHitNode(ctx, nodeName, processedTmpl, templateScope, orgTmpl, boundaryID, outputs, memoizationStatus, nodeFlag)
			} else {
				woc.log.WithField("nodeName", nodeName).Info(ctx, "Node is using mutex with memoize. Cache is hit.")
				woc.updateAsCacheHitNode(ctx, node, outputs, memoizationStatus)
			}
		} else {
			if node == nil {
				_, node = woc.initializeCacheNode(ctx, nodeName, processedTmpl, templateScope, orgTmpl, boundaryID, memoizationStatus, nodeFlag)
			} else {
				woc.log.WithField("nodeName", nodeName).Info(ctx, "Node is using mutex with memoize. Cache is NOT hit")
				woc.updateAsCacheNode(ctx, node, memoizationStatus)
			}
		}
		woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
		woc.updated = true
	}
	return false, node, nil
}

// executeTemplateFunc is a function type for executing a template (used in retries)
type executeTemplateFunc func(ctx context.Context, nodeName string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error)

// handleRetries handles retry logic.
func (woc *wfOperationCtx) handleRetries(ctx context.Context, node *wfv1.NodeStatus, nodeName string, processedTmpl *wfv1.Template, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts, next executeTemplateFunc) (*wfv1.NodeStatus, error) {
	if woc.retryStrategy(processedTmpl) == nil {
		return next(ctx, nodeName, processedTmpl, orgTmpl, opts)
	}

	retryNodeName := nodeName
	retryParentNode := node
	if retryParentNode == nil {
		woc.log.WithField("nodeName", retryNodeName).Debug(ctx, "Inject a retry node")
		_, retryParentNode = woc.initializeExecutableNode(ctx, retryNodeName, wfv1.NodeTypeRetry, templateScope, processedTmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning, opts.nodeFlag, true)
	}
	if opts.nodeFlag == nil {
		opts.nodeFlag = &wfv1.NodeFlag{}
	}
	opts.nodeFlag.Retried = true
	processedRetryParentNode, continueExecution, err := woc.processNodeRetries(ctx, retryParentNode, *woc.retryStrategy(processedTmpl), opts)
	if err != nil {
		return woc.markNodeError(ctx, retryNodeName, err), err
	}
	// An attempt the retry will follow with another is finished here, so
	// its metrics count it with its own status and exit code (#8207,
	// #10463); the final attempt is counted by the Retry node below. A pod
	// attempt is fulfilled by pod reconciliation and is never re-entered, so
	// this is the only place that finishes it. A nested Steps/DAG attempt
	// finished itself in its own dispatch, and handleNodeFulfilled does not
	// finish a node twice. This runs before a backoff returns, so an attempt
	// is finished in the operation that sees it fail.
	if _, lastChildNode := getChildNodeIdsAndLastRetriedNode(processedRetryParentNode, woc.wf.Status.Nodes); lastChildNode != nil &&
		!processedRetryParentNode.Fulfilled() && lastChildNode.Phase.Fulfilled(lastChildNode.TaskResultSynced) {
		woc.handleNodeFulfilled(ctx, lastChildNode, processedTmpl)
	}
	if !continueExecution {
		return retryParentNode, nil
	}
	retryParentNode = processedRetryParentNode
	childNodeIDs, lastChildNode := getChildNodeIdsAndLastRetriedNode(retryParentNode, woc.wf.Status.Nodes)

	// A Retry node the retry policy has finished is done once its last
	// attempt is: it goes on only to re-enter an attempt still running, never
	// to start another. Its other children (its hooks, the next StepGroup
	// hung off it) do not hold it back.
	if retryParentNode.Fulfilled() && (lastChildNode == nil || lastChildNode.Fulfilled() || (retryParentNode.IsDaemoned() && retryParentNode.FailedOrError())) {
		if lastChildNode != nil {
			retryParentNode.Outputs = lastChildNode.Outputs.DeepCopy()
			woc.wf.Status.Nodes.Set(ctx, retryParentNode.ID, *retryParentNode)
		}
		woc.handleNodeFulfilled(ctx, retryParentNode, processedTmpl)
		return retryParentNode, nil
	}
	var retryNum int
	if lastChildNode != nil && !lastChildNode.Phase.Fulfilled(lastChildNode.TaskResultSynced) {
		nodeName = lastChildNode.Name
		node = lastChildNode
		retryNum = len(childNodeIDs) - 1
	} else {
		retryNum = len(childNodeIDs)
		nodeName = fmt.Sprintf("%s(%d)", retryNodeName, retryNum)
		// We need to check if the node already exists in case we are re-processing
		node, _ = woc.wf.GetNodeByName(nodeName)
	}

	localParams := make(map[string]string)
	if processedTmpl.IsPodType() {
		localParams[varkeys.PodName.Template()] = woc.getPodName(nodeName, processedTmpl.Name)
	}
	localParams[varkeys.Retries.Template()] = strconv.Itoa(retryNum)

	// The first attempt has no previous child to read lastRetry variables
	// from; exitCode and duration must default to "0" so numeric expressions
	// (e.g. asInt(lastRetry.exitCode)) resolve (#10364, #14450).
	exitCode := "0"
	status := ""
	duration := "0"
	message := ""

	if lastChildNode != nil {
		if lastChildNode.Outputs != nil && lastChildNode.Outputs.ExitCode != nil {
			exitCode = *lastChildNode.Outputs.ExitCode
		}
		status = string(lastChildNode.Phase)
		duration = fmt.Sprint(lastChildNode.GetDuration().Seconds())
		message = lastChildNode.Message
	}

	localParams[varkeys.RetriesLastExitCode.Template()] = exitCode
	localParams[varkeys.RetriesLastStatus.Template()] = status
	localParams[varkeys.RetriesLastDuration.Template()] = duration
	localParams[varkeys.RetriesLastMessage.Template()] = message

	// Save the unsubstituted template for potential recursive retry calls.
	// SubstituteParams replaces {{retries}} etc. in-place, so the recursive call
	// needs the original template to correctly substitute the next retry's values.
	unsubstitutedTmpl := processedTmpl.DeepCopy()

	// Always substitute retry params (matching main branch behavior).
	// This is needed even when re-executing an existing Pending child (e.g. exceeded quota)
	// because {{retries}} and {{pod.name}} must be resolved before pod creation.
	processedTmpl, err = common.SubstituteParams(ctx, processedTmpl, woc.globalParams(), localParams)
	if errorsutil.IsTransientErr(ctx, err) {
		return node, err
	}
	if err != nil {
		errNode := woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, opts.boundaryID, opts.nodeFlag, err)
		if node == nil {
			// the attempt node was just created; link it or the next
			// reconcile re-derives the same attempt name and panics
			woc.addChildNode(ctx, retryNodeName, nodeName)
		}
		return errNode, err
	}

	// Link a new retry attempt only now that nothing can return before the
	// dispatch below creates its node: an edge persisted for a node that is
	// never created (the parameter substitution above can return on transient
	// errors) can later be claimed by a colliding name (#16376). It still
	// precedes dispatch so FindRetryNode (used by scheduleOnDifferentHost for
	// nodeAntiAffinity) can locate the retry parent during pod creation.
	if node == nil {
		woc.addChildNode(ctx, retryNodeName, nodeName)
	}

	childNode, err := next(ctx, nodeName, processedTmpl, orgTmpl, opts)
	if err != nil {
		return woc.markNodeError(ctx, retryParentNode.Name, err), err
	}

	woc.addChildNode(ctx, retryParentNode.Name, childNode.Name)

	if !childNode.Phase.Fulfilled(childNode.TaskResultSynced) && childNode.IsDaemoned() {
		retryParentNode = woc.markRetryNodeDaemoned(ctx, retryParentNode.Name, childNode.Phase)
	}

	// Re-fetch the child node since dispatch (next) may have updated it in-place
	// (e.g., a Steps/DAG child that completed during executeSteps).
	if retrieved, err := woc.wf.GetNodeByName(childNode.Name); err == nil {
		childNode = retrieved
	}

	// If the child became fulfilled during this dispatch, re-enter the retry handler.
	// This matches main branch behavior where executeTemplate recursively re-enters
	// itself when a retry child completes, allowing retries to progress within a
	// single operate cycle. This covers both cases: re-executing an existing child
	// that transitions to a terminal phase, and new children that complete instantly.
	if childNode.Phase.Fulfilled(childNode.TaskResultSynced) {
		retryParentNode, _ = woc.wf.GetNodeByName(retryParentNode.Name)
		if retryParentNode != nil && !retryParentNode.Phase.Fulfilled(retryParentNode.TaskResultSynced) {
			return woc.handleRetries(ctx, retryParentNode, retryNodeName, unsubstitutedTmpl, templateScope, orgTmpl, opts, next)
		}
	}

	return retryParentNode, nil
}

// postExecutionHandling handles error checking, sync release, and metrics.
func (woc *wfOperationCtx) postExecutionHandling(ctx context.Context, node *wfv1.NodeStatus, nodeName string, processedTmpl *wfv1.Template, err error) (*wfv1.NodeStatus, error) {
	if err != nil {
		node = woc.markNodeError(ctx, nodeName, err)

		retryStrategy := woc.retryStrategy(processedTmpl)
		release := false
		if retryStrategy == nil {
			release = true
		} else {
			retryPolicy := retryStrategy.RetryPolicyActual()
			if retryPolicy != wfv1.RetryPolicyAlways &&
				retryPolicy != wfv1.RetryPolicyOnError &&
				retryPolicy != wfv1.RetryPolicyOnTransientError {
				release = true
			}
		}
		if release {
			woc.controller.syncManager.Release(ctx, woc.wf, node.ID, processedTmpl.Synchronization)
			return node, err
		}
	}

	// Task-result placeholder nodes have empty Type AND empty Phase — they are
	// pre-synced outputs whose real node was never initialized (e.g. a workflow
	// labelled completed while still Running, #12615). This error is fatal by
	// design: the reconciler's recordTaskError records it as an Error node in
	// the placeholder's slot, so the task fails rather than reconciling an
	// uninitializable node forever (see TestWorkflowRunningButLabelCompleted).
	// We only return the error — the callers, not this function, record it.
	// Note: we check both Type=="" and Phase=="" to distinguish placeholders from
	// legitimately initialized nodes that happen to have empty Type (e.g., nodes
	// from FormulateResubmitWorkflow where the YAML didn't specify Type).
	if node.Type == "" && node.Phase == "" {
		return node, fmt.Errorf("task result placeholder node has empty type")
	}

	retrieveNode, err := woc.wf.GetNodeByName(node.Name)
	if err != nil {
		err := fmt.Errorf("no Node found by the name of %s;  wf.Status.Nodes=%+v", node.Name, woc.wf.Status.Nodes)
		woc.log.Error(ctx, err.Error())
		woc.markWorkflowError(ctx, err)
		return node, err
	}
	node = retrieveNode

	// A node this dispatch completed (a nested template, a suspend whose
	// duration passed) is finished here. Realtime metric registration moved
	// out of here to emitNodeMetrics, on the node executeProcessedTemplate
	// finally returns: for a retried template that is always the Retry
	// node, not each attempt this dispatch may be handling.
	woc.handleNodeFulfilled(ctx, node, processedTmpl)
	return node, nil
}
