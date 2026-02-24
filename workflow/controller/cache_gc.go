package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

var gcAfterNotHitDuration = env.LookupEnvDurationOr(logging.InitLoggerInContext(), "CACHE_GC_AFTER_NOT_HIT_DURATION", 30*time.Second)

func init() {
	logging.InitLogger().WithField("gcAfterNotHitDuration", gcAfterNotHitDuration).Info(context.Background(), "Memoization caches will be garbage-collected if they have not been hit after")
}

// syncAllCacheForGC syncs all cache for GC
func (wfc *WorkflowController) syncAllCacheForGC(ctx context.Context) {
	logger := logging.RequireLoggerFromContext(ctx)
	configMaps, err := wfc.configMapInformer.GetIndexer().ByIndex(indexes.ConfigMapLabelsIndex, common.LabelValueTypeConfigMapCache)
	if err != nil {
		logger.WithError(err).Error(ctx, "Failed to get configmaps from informer")
		return
	}

	for _, obj := range configMaps {
		cm, ok := obj.(*apiv1.ConfigMap)
		if !ok {
			logger.Error(ctx, "Unable to convert object to configmap when syncing ConfigMaps")
			continue
		}
		if err := wfc.cleanupUnusedCache(ctx, cm); err != nil {
			logger.WithField("configMap", cm.Name).WithError(err).Error(ctx, "Unable to sync ConfigMap")
		}
	}
}

func (wfc *WorkflowController) cleanupUnusedCache(ctx context.Context, cm *apiv1.ConfigMap) error {
	logger := logging.RequireLoggerFromContext(ctx)
	var modified bool
	for key, rawEntry := range cm.Data {
		var entry controllercache.Entry
		if err := json.Unmarshal([]byte(rawEntry), &entry); err != nil {
			return fmt.Errorf("malformed cache entry: could not unmarshal JSON; unable to parse: %w", err)
		}
		if time.Since(entry.LastHitTimestamp.Time) > gcAfterNotHitDuration {
			logger.WithFields(logging.Fields{"key": key, "configMap": cm.Name, "gcAfterNotHitDuration": gcAfterNotHitDuration}).Info(ctx, "Deleting entry in ConfigMap since it's not been hit")
			delete(cm.Data, key)
			modified = true
		}
	}
	if len(cm.Data) == 0 {
		err := wfc.kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
		if err != nil {
			if apierr.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to delete ConfigMap %s: %w", cm.Name, err)
		}
	} else {
		if modified {
			_, err := wfc.kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update ConfigMap %s: %w", cm.Name, err)
			}
		}
	}

	return nil
}
