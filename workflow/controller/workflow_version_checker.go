package controller

import (
	"fmt"
	"time"

	"github.com/TwiN/gocache/v2"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// workflowVersionChecker is a cache used to check if current workflow status is outdated.
// There are 2 kinds of information stored in the cache:
// 1. The latest version of the workflow, use the namespace/workflow as the key, value is the latest version
// 2. The outdated versions of workflows, use the namespace/workflow:version as the key, value is nil
// Set the ttl as the max delay of the informer you anticipate.
type workflowVersionChecker struct {
	cache *gocache.Cache
}

const NoExpiration = gocache.NoExpiration

func NewWorkflowVersionChecker(ttl time.Duration) *workflowVersionChecker {
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

func (c *workflowVersionChecker) UpdateLatestVersion(wf metav1.Object) {
	key, _ := cache.MetaNamespaceKeyFunc(wf)
	currentVersion := wf.GetResourceVersion()
	previousVersion, exist := c.cache.Get(key)
	if exist {
		previousVersionStr, _ := previousVersion.(string)
		if previousVersionStr == currentVersion {
			return
		}
		// add the previous version to the cache as it's outdated
		c.cache.Set(toProcessedVersionKey(wf, previousVersionStr), nil)
	}
	// if the workflow has not been marked for expiration, keep the latest version
	c.cache.SetWithTTL(key, currentVersion, NoExpiration)
}

func toProcessedVersionKey(wf metav1.Object, version string) string {
	key, _ := cache.MetaNamespaceKeyFunc(wf)
	return fmt.Sprintf("%s:%s", key, version)
}

func (c *workflowVersionChecker) MarkForExpiration(wf interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(wf)
	if err != nil {
		log.WithError(err).Error("Failed to get key for workflow version checker")
		return
	}
	currentVersion, exist := c.cache.Get(key)
	if exist {
		// set with default ttl so it could be removed later
		c.cache.Set(key, currentVersion)
	}
}
