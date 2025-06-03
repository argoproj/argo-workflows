package controller

import (
	"fmt"
	"time"

	"github.com/TwiN/gocache/v2"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// workflowVersionChecker is a cache used to check if a workflow status is outdated.
// It stores outdated versions of workflows, use the namespace/workflow:version as the key, value is nil.
// Set the ttl as the max delay of the informer you anticipate.
type workflowVersionChecker struct {
	cache *gocache.Cache
}

const NoExpiration = gocache.NoExpiration

func NewWorkflowVersionChecker(ttl time.Duration) *workflowVersionChecker {
	log.WithFields(log.Fields{"ttl": ttl}).Info("Starting workflow version checker")
	cache := gocache.NewCache().WithDefaultTTL(ttl).WithMaxSize(gocache.NoMaxSize).WithMaxMemoryUsage(gocache.NoMaxMemoryUsage).WithEvictionPolicy(gocache.LeastRecentlyUsed)
	if err := cache.StartJanitor(); err != nil {
		log.WithError(err).Warn("Failed to start cache janitor, TTL functionality will be disabled")
	}
	return &workflowVersionChecker{cache: cache}
}

func (c *workflowVersionChecker) IsOutdated(wf metav1.Object) bool {
	_, exist := c.cache.Get(toProcessedVersionKey(wf, wf.GetResourceVersion()))
	return exist
}

func (c *workflowVersionChecker) UpdateOutdatedVersion(wf metav1.Object) {
	c.cache.Set(toProcessedVersionKey(wf, wf.GetResourceVersion()), nil)
	log.WithFields(log.Fields{"workflow": wf.GetName(), "version": wf.GetResourceVersion()}).Info("Workflow version checker: logged outdated version")
}

func toProcessedVersionKey(wf metav1.Object, version string) string {
	key, _ := cache.MetaNamespaceKeyFunc(wf)
	return fmt.Sprintf("%s:%s", key, version)
}
