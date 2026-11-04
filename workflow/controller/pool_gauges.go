package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// taskSetClaim returns the cached claim on a pooled TaskSet, if a result
// claims it.
func taskSetClaim(results cache.Store, namespace string, ref dispatch.TaskSetRef) (dispatch.Claim, bool) {
	key := dispatch.Key{Namespace: namespace, NodeID: ref.NodeID}
	obj, exists, err := results.GetByKey(namespace + "/" + key.ResultName())
	if err != nil || !exists {
		return dispatch.Claim{}, false
	}
	result, err := meta.Accessor(obj)
	if err != nil {
		return dispatch.Claim{}, false
	}
	claim, ok := dispatch.ClaimFromResult(result)
	if !ok || claim.TaskSetUID != ref.UID {
		return dispatch.Claim{}, false
	}
	return claim, true
}
