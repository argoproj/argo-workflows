package deployment

import (
	"applatix.io/axamm/utils"
	"sync"
	"time"
)

var InstancesCache map[string]*DeploymentStatus = map[string]*DeploymentStatus{}
var cacheLock sync.Mutex
var cacheTTL int64 = 120 // 2 minutes

func GetPodCacheById(id string) *DeploymentStatus {
	return InstancesCache[id]
}

func DeletePodCache(id string) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	delete(InstancesCache, id)
}

func UpdatePodCache(d *Deployment, st *DeploymentStatus) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	InstancesCache[d.Id] = st
}

func UpdatePodCacheDelta(d *Deployment, hb *PodHeartBeat) (*DeploymentStatus, string) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	new := &DeploymentStatus{
		Name:            d.Name,
		DesiredReplicas: d.Template.Scale.Min,
		Pods:            []*Pod{},
	}

	var reason string

	podMap := map[string]*Pod{}
	if InstancesCache[d.Id] != nil {
		for _, pod := range InstancesCache[d.Id].Pods {
			copy := pod
			// Pod Status is expired in 2 minutes without updating
			if time.Now().Unix()-copy.Mtime < cacheTTL {
				podMap[copy.Name] = copy
			} else {
				utils.InfoLog.Printf("[HB] Pod information is expired: %v.\n", *pod)
			}
		}
	}

	pod := hb.Data.PodStatus
	pod.Mtime = hb.Date

	oldPod := podMap[pod.Name]
	if oldPod == nil || (oldPod != nil && pod.Mtime > oldPod.Mtime) {
		switch hb.Data.Type {
		case TypeBirthCry, TypeHeartBeat, TypeArtifactLoadStart:
			podMap[pod.Name] = pod
		case TypeTombStone:
			delete(podMap, pod.Name)
		case TypeArtifactLoadFailed:
			podMap[pod.Name] = pod
		default:
			utils.ErrorLog.Printf("[HB] Unsupported heart beat type: %v.\n", hb.Data.Type)
		}

		reason = hb.Data.Type
	}

	var available int
	for _, pod := range podMap {
		copy := pod
		new.Pods = append(new.Pods, copy)
		if copy.Ready() {
			available++
		}
	}

	new.AvailableReplicas = available
	InstancesCache[d.Id] = new
	return new, reason
}
