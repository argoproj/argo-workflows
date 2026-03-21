// Package pod implements pod life cycle management
package pod

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
)

func (c *Controller) GetPod(namespace string, podName string) (*apiv1.Pod, error) {
	obj, exists, err := c.podInformer.GetStore().GetByKey(namespace + "/" + podName)
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

// TODO - return []*apiv1.Pod instead, save on duplicating this
func (c *Controller) GetPodsByIndex(index, key string) ([]any, error) {
	return c.podInformer.GetIndexer().ByIndex(index, key)
}

func (c *Controller) TerminateContainers(ctx context.Context, namespace, name string) {
	c.queuePodForCleanup(ctx, namespace, name, terminateContainers)
}

func (c *Controller) DeletePod(ctx context.Context, namespace, name string) {
	c.queuePodForCleanup(ctx, namespace, name, deletePod)
}

func (c *Controller) DeletePodByUID(ctx context.Context, namespace, name, uid string) {
	c.queuePodForCleanupByUID(ctx, namespace, name, uid)
}

func (c *Controller) RemoveFinalizer(ctx context.Context, namespace, name string) {
	c.queuePodForCleanup(ctx, namespace, name, removeFinalizer)
}
