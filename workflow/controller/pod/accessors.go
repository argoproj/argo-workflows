// Package pod implements pod life cycle management
package pod

import (
	"fmt"
	"log"
	"runtime/debug"

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
func (c *Controller) GetPodsByIndex(index, key string) ([]interface{}, error) {
	return c.podInformer.GetIndexer().ByIndex(index, key)
}

func (c *Controller) TerminateContainers(namespace, name string) {
	log.Println("[debug]")
	log.Println(string(debug.Stack()))
	c.queuePodForCleanup(namespace, name, terminateContainers)
}

func (c *Controller) DeletePod(namespace, name string) {
	c.queuePodForCleanup(namespace, name, deletePod)
}

func (c *Controller) RemoveFinalizer(namespace, name string) {
	c.queuePodForCleanup(namespace, name, removeFinalizer)
}
