package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/env"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SyncAllCacheForGC syncs all cache for GC
func (wfc *WorkflowController) SyncAllCacheForGC(ctx context.Context) {
	gcAfterNotHitDuration := env.LookupEnvDurationOr("CACHE_GC_AFTER_NOT_HIT_DURATION", 30*time.Second)
	configMaps, err := wfc.configMapInformer.GetIndexer().ByIndex(indexes.ConfigMapLabelsIndex, common.LabelValueCacheTypeConfigMap)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Errorln("Failed to get configmaps from informer")
	}

	for _, obj := range configMaps {
		cm, ok := obj.(*apiv1.ConfigMap)
		if !ok {
			log.WithField("configMap", cm.Name).Errorln("Unable to convert object to configmap when syncing ConfigMaps")
			continue
		}
		if err := cleanupUnusedCache(ctx, wfc.kubeclientset, cm, gcAfterNotHitDuration); err != nil {
			log.WithFields(log.Fields{"configMap": cm.Name, "error": err}).Errorln("Unable to sync ConfigMap")
			continue
		}
	}
}

func cleanupUnusedCache(ctx context.Context, kubeclientset kubernetes.Interface, cm *apiv1.ConfigMap, gcAfterNotHitDuration time.Duration) error {
	var modified bool
	for key, rawEntry := range cm.Data {
		var entry controllercache.Entry
		if err := json.Unmarshal([]byte(rawEntry), &entry); err != nil {
			return fmt.Errorf("malformed cache entry: could not unmarshal JSON; unable to parse: %w", err)
		}
		if time.Since(entry.LastHitTimestamp.Time) > gcAfterNotHitDuration {
			log.WithFields(log.Fields{"key": key, "configMap": cm.Name, "gcAfterNotHitDuration": gcAfterNotHitDuration}).Infoln("Deleting entry in ConfigMap since it's not been hit")
			delete(cm.Data, key)
			modified = true
		}
	}
	if len(cm.Data) == 0 {
		log.WithField("configMap", cm.Name).Debugln("Deleting ConfigMap since it doesn't contain any cache entries")
		err := kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete ConfigMap %s: %w", cm.Name, err)
		}
	} else {
		if modified {
			log.WithField("configMap", cm.Name).Debugln("Updated ConfigMap")
			_, err := kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update ConfigMap %s: %w", cm.Name, err)
			}
		}
	}

	return nil
}
