package controller

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// prepareNode initializes or updates the node status and sets the display name.
func (woc *wfOperationCtx) prepareNode(ctx context.Context, nodeName string, tmplCtx *templateresolution.TemplateContext, processedTmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, nodeFlag *wfv1.NodeFlag) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.Warn(ctx, "Node was nil, will be initialized as type Skipped")
	}

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
	if time.Now().UTC().After(woc.deadline) {
		woc.log.Warn(ctx, "Deadline exceeded")
		woc.requeue()
		return ErrDeadlineExceeded
	}

	_, err := woc.checkTemplateTimeout(processedTmpl, node)
	if err != nil {
		woc.log.WithField("template", processedTmpl.Name).Warn(ctx, "Template exceeded its deadline")
		_ = woc.markNodePhase(ctx, nodeName, wfv1.NodeFailed, err.Error())
		return err
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

	lockCtx, lockSpan := woc.controller.tracing.StartTryAcquireLock(ctx, woc.wf.NodeID(nodeName), false)
	lockAcquired, wfUpdated, msg, failedLockName, err := woc.controller.syncManager.TryAcquire(lockCtx, woc.wf, woc.wf.NodeID(nodeName), processedTmpl.Synchronization)
	lockSpan.SetAttributes(attribute.Bool("LockAcquired", lockAcquired))
	lockSpan.End()
	if err != nil {
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
	} else if !continueExecution {
		return retryParentNode, nil
	}
	retryParentNode = processedRetryParentNode
	childNodeIDs, lastChildNode := getChildNodeIdsAndLastRetriedNode(retryParentNode, woc.wf.Status.Nodes)

	if retryParentNode.Fulfilled() && (woc.childrenFulfilled(retryParentNode) || (retryParentNode.IsDaemoned() && retryParentNode.FailedOrError())) {
		if lastChildNode != nil {
			retryParentNode.Outputs = lastChildNode.Outputs.DeepCopy()
			woc.wf.Status.Nodes.Set(ctx, retryParentNode.ID, *retryParentNode)
		}
		if processedTmpl.Metrics != nil {
			if prevNodeStatus, ok := woc.preExecutionNodeStatuses[retryParentNode.ID]; (!ok || !prevNodeStatus.Fulfilled()) && retryParentNode.Fulfilled() {
				localScope, realTimeScope := woc.prepareMetricScope(processedRetryParentNode)
				woc.computeMetrics(ctx, processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
			}
		}
		if processedTmpl.Synchronization != nil {
			woc.controller.syncManager.Release(ctx, woc.wf, retryParentNode.ID, processedTmpl.Synchronization)
		}
		_, lastChildNode = getChildNodeIdsAndLastRetriedNode(retryParentNode, woc.wf.Status.Nodes)
		if lastChildNode != nil {
			retryParentNode.Outputs = lastChildNode.Outputs.DeepCopy()
			woc.wf.Status.Nodes.Set(ctx, retryParentNode.ID, *retryParentNode)
		}
		return retryParentNode, nil
	} else if lastChildNode != nil && lastChildNode.Fulfilled() && processedTmpl.Metrics != nil {
		localScope, realTimeScope := woc.prepareMetricScope(lastChildNode)
		woc.computeMetrics(ctx, processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
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
		// Register the child with the retry parent BEFORE dispatching so that
		// FindRetryNode (used by scheduleOnDifferentHost for nodeAntiAffinity)
		// can locate the retry parent during pod creation.
		woc.addChildNode(ctx, retryNodeName, nodeName)
	}

	localParams := make(map[string]string)
	if opts.scope != nil {
		maps.Copy(localParams, opts.scope.getParameters())
	}
	if processedTmpl.IsPodType() {
		localParams[varkeys.PodName.Template()] = woc.getPodName(nodeName, processedTmpl.Name)
	}
	localParams[varkeys.Retries.Template()] = strconv.Itoa(retryNum)

	exitCode := ""
	status := ""
	duration := ""
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
	//
	// allowUnresolved=true: late-resolved tags like {{pod.name}} (for non-pod
	// retry-decorated templates) and {{tasks.X.outputs.*}} are substituted by
	// later passes. Matches origin/main behavior; the previous opts.onExitTemplate
	// value was a bool meaning "is this an onExit handler call?" and was
	// semantically unrelated to allowUnresolved — for normal (non-exit) retries
	// it evaluated to false and broke any retry-decorated template body with
	// late-resolved tags.
	processedTmpl, err = common.SubstituteParams(ctx, processedTmpl, woc.globalParams(), localParams, true)
	if errorsutil.IsTransientErr(ctx, err) {
		return node, err
	}
	if err != nil {
		return woc.initializeNodeOrMarkError(ctx, node, nodeName, templateScope, orgTmpl, opts.boundaryID, opts.nodeFlag, err), err
	}

	childNode, err := next(ctx, nodeName, processedTmpl, orgTmpl, opts)
	if err != nil {
		return woc.markNodeError(ctx, retryParentNode.Name, err), err
	}

	woc.addChildNode(ctx, retryParentNode.Name, childNode.Name)

	if !childNode.Phase.Fulfilled(childNode.TaskResultSynced) && childNode.IsDaemoned() {
		retryParentNode = woc.markNodePhase(ctx, retryParentNode.Name, childNode.Phase)
		if childNode.IsDaemoned() {
			retryParentNode.Daemoned = new(true)
		}
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

	if node.Fulfilled() {
		woc.controller.syncManager.Release(ctx, woc.wf, node.ID, processedTmpl.Synchronization)
	}

	// Task-result placeholder nodes have empty Type AND empty Phase — they are
	// pre-synced outputs awaiting real node initialization. Return an error so
	// the execution machinery (executeSteps/executeDAG) can assess the phase
	// correctly. Do NOT call markWorkflowError here: when task results are still
	// in progress, the workflow must stay Running; the callers will mark the
	// appropriate phase.
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

	if processedTmpl.Metrics != nil {
		if _, ok := woc.preExecutionNodeStatuses[node.ID]; !ok {
			localScope, realTimeScope := woc.prepareMetricScope(node)
			woc.computeMetrics(ctx, processedTmpl.Metrics.Prometheus, localScope, realTimeScope, true)
		}
		if prevNodeStatus, ok := woc.preExecutionNodeStatuses[node.ID]; (!ok || !prevNodeStatus.Fulfilled()) && node.Fulfilled() {
			localScope, realTimeScope := woc.prepareMetricScope(node)
			woc.computeMetrics(ctx, processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
		}
	}
	return node, nil
}
