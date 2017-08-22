package deployment

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/axops/host"
	"applatix.io/common"
	"applatix.io/template"
)

func (d *Deployment) getMaxResources() (cpu float64, mem float64) {

	cpu, mem = d.getInstanceResources()

	// take scale factor into consideration
	if d.Template.Scale != nil && d.Template.Scale.Min != 0 {
		cpu = float64(d.Template.Scale.Min) * cpu
		mem = float64(d.Template.Scale.Min) * mem
	}

	d.CPU = cpu
	d.Mem = mem

	return
}

func calculateMaxResource(r1, r2 float64, scale1, scale2, ms, mu int) float64 {

	// simulate how k8s will bring nodes up and down

	max := math.Max(r1*float64(scale1), r2*float64(scale2))
	// old and new instances
	i1, i2 := scale1, 0

	if mu > scale1 {
		mu = scale1
	}
	if ms > scale2 {
		ms = scale2
	}

	i1 -= mu
	i2 += ms + mu
	if i2 > scale2 {
		i2 = scale2
	}
	max = math.Max(max, (float64(i1)*r1 + float64(i2)*r2))

	for i1 > 0 && i2 < scale2 {
		i1--
		i2++
		max = math.Max(max, (float64(i1)*r1 + float64(i2)*r2))
		utils.DebugLog.Println(i1, i2, max)
	}
	return max

}

func (d *Deployment) GetMaxResourcesForUpgrade(n *Deployment) (cpu float64, mem float64) {

	if d.Template.Strategy != nil && d.Template.Strategy.Type == template.StrategyRollingUpdate {
		// per instance resource and scale for old
		cpu1, mem1 := d.getInstanceResources()
		scale1 := 1
		if d.Template.Scale != nil && d.Template.Scale.Min != 0 {
			scale1 = d.Template.Scale.Min
		}
		// per instance resource and scale for new
		cpu2, mem2 := n.getInstanceResources()
		scale2 := 1
		if n.Template.Scale != nil && n.Template.Scale.Min != 0 {
			scale2 = n.Template.Scale.Min
		}
		// rolling strategy
		up, dn := 1, 1
		if d.Template.Strategy.RollingUpdate != nil {
			parsedUp, err := strconv.Atoi(d.Template.Strategy.RollingUpdate.MaxSurge)
			if err == nil {
				up = parsedUp
			}
			parsedDn, err := strconv.Atoi(d.Template.Strategy.RollingUpdate.MaxUnavailable)
			if err == nil {
				dn = parsedDn
			}
		}
		cpu = calculateMaxResource(cpu1, cpu2, scale1, scale2, up, dn)
		mem = calculateMaxResource(mem1, mem2, scale1, scale2, up, dn)
	} else {
		// return max of the 2 for recreate strategy
		cpu1, mem1 := d.getMaxResources()
		cpu2, mem2 := n.getMaxResources()
		cpu = math.Max(cpu1, cpu2)
		mem = math.Max(mem1, mem2)
	}
	return
}

var DockerEnabledKey = "ax_ea_docker_enable"

type DockerConfig struct {
	GraphStorageName string  `json:"graph_storage_name,omitempty"`
	GraphStorageSize string  `json:"graph_storage_size,omitempty"`
	CpuCores         float64 `json:"cpu_cores,omitempty"`
	MemMib           int64   `json:"mem_mib,omitempty"`
}

var ResourceScaleFactor float64 = 1.0
var ResourceCpuOverhead float64 = 0.0
var ResourceMemOverhead float64 = 0.0

func (d *Deployment) getInstanceResources() (cpu float64, mem float64) {

	if d.Template == nil {
		return 0.0, 0.0
	}

	maxStepCpu := 0.0
	maxStepMem := 0.0

	stepCpu := 0.0
	stepMem := 0.0
	for _, ctr := range d.Template.Containers {
		c, m := ctr.GetMaxResources()
		stepCpu += c
		stepMem += m
		if stepCpu > maxStepCpu {
			maxStepCpu = stepCpu
		}
		if stepMem > maxStepMem {
			maxStepMem = stepMem
		}
	}

	cpu = maxStepCpu
	mem = maxStepMem

	if d.Annotations != nil {
		for key, value := range d.Annotations {
			if key == DockerEnabledKey {
				var docker DockerConfig
				err := json.Unmarshal([]byte(value), &docker)
				if err == nil {
					utils.DebugLog.Println("[ADC] dind ", docker)
					if math.Abs(docker.CpuCores-0.0) <= 0.000001 {
						cpu = cpu * 2
					} else {
						cpu += float64(docker.CpuCores)
					}

					if docker.MemMib == 0 {
						mem = mem * 2
					} else {
						mem += float64(docker.MemMib)
					}

					//if dindCpuCores, err := strconv.ParseFloat(docker.CpuCores, 64); err == nil {
					//	cpu = cpu + dindCpuCores
					//} else {
					//	cpu = cpu * 2
					//}
					//
					//if dindMemMib, err := strconv.ParseFloat(docker.MemMib, 64); err == nil {
					//	mem = mem + dindMemMib
					//} else {
					//	mem = mem * 2
					//}
				}
			}
		}
	}

	if d.Labels != nil {
		for key, value := range d.Labels {
			if key == DockerEnabledKey {
				var docker DockerConfig
				err := json.Unmarshal([]byte(value), &docker)
				if err == nil {
					utils.DebugLog.Println("[ADC] dind ", docker)
					if math.Abs(docker.CpuCores-0.0) <= 0.000001 {
						cpu = cpu * 2
					} else {
						cpu += float64(docker.CpuCores)
					}

					if docker.MemMib == 0 {
						mem = mem * 2
					} else {
						mem += float64(docker.MemMib)
					}

					//if dindCpuCores, err := strconv.ParseFloat(docker.CpuCores, 64); err == nil {
					//	cpu = cpu + dindCpuCores
					//} else {
					//	cpu = cpu * 2
					//}
					//
					//if dindMemMib, err := strconv.ParseFloat(docker.MemMib, 64); err == nil {
					//	mem = mem + dindMemMib
					//} else {
					//	mem = mem * 2
					//}
				}
			}
		}
	}

	// Add the artifacts resource
	cpu += ResourceCpuOverhead
	mem += ResourceMemOverhead

	// Scale the CPU request
	cpu = cpu * ResourceScaleFactor

	return
}

func ExtendDeploymentResource() *axerror.AXError {
	params := map[string]interface{}{}
	params[DeploymentAppName] = utils.APPLICATION_NAME

	common.DebugLog.Printf("[ADC] Deployment resource extending starting\n")

	deployments, axErr := GetLatestDeployments(params, false)
	if axErr != nil {
		return axErr
	}

	common.DebugLog.Printf("[ADC] Find %v deployments\n", len(deployments))

	for i, _ := range deployments {
		extendDeploymentResource(deployments[i])
	}

	common.DebugLog.Printf("[ADC] Deployment resource extending finished\n")
	return nil
}

func extendDeploymentResource(d *Deployment) {

	//locked := DeployLockGroup.TryLock(d.Key(), time.Duration(-1))
	//if locked {
	//	defer DeployLockGroup.Unlock(d.Key())
	//} else {
	//	return
	//}

	DeployLockGroup.Lock(d.Key())
	defer DeployLockGroup.Unlock(d.Key())

	latest, axErr := GetDeploymentByID(d.Id, false)
	if axErr != nil {
		common.ErrorLog.Printf("Error fetch deployment %v/%v: %v", d.ApplicationName, d.Name, axErr)
	}

	if latest == nil {
		return
	}

	d = latest

	common.DebugLog.Printf("[ADC] Procoess deployment %v/%v/%v starting\n", d.ApplicationName, d.Name, d.Status)
	switch d.Status {
	case DeployStateWaiting, DeployStateActive, DeployStateError:
		if axErr, _ := d.reserveAdcResources(); axErr != nil {
			common.ErrorLog.Printf("[ADC] Error extend deployment %v/%v resource reservation: %v", d.ApplicationName, d.Name, axErr)
		}
	case DeployStateUpgrading:
		old, _ := GetHistoryDeploymentByID(d.PreviousDeploymentId, false)
		if old != nil {
			if axErr, _ := d.reserveAdcResourcesForUpgrade(old); axErr != nil {
				common.ErrorLog.Printf("[ADC] Error extend deployment %v/%v resource reservation: %v", d.ApplicationName, d.Name, axErr)
			}

		} else {
			if axErr, _ := d.reserveAdcResources(); axErr != nil {
				common.ErrorLog.Printf("[ADC] Error extend deployment %v/%v resource reservation: %v", d.ApplicationName, d.Name, axErr)
			}
		}

	}
	common.DebugLog.Printf("[ADC] Procoess deployment %v/%v/%v finished\n", d.ApplicationName, d.Name, d.Status)
}

func ScheduleDeploymentResourceExtender() {
	ticker := time.NewTicker(time.Minute * 20)
	go func() {
		for _ = range ticker.C {
			ExtendDeploymentResource()
			// Refresh the average price periodically
			host.GetAveragePrice(true)
		}
	}()
}

func (d *Deployment) updateCost() (*axerror.AXError, int) {
	if axErr, code := GetDeploymentSpending(d); axErr != nil {
		return axErr, code
	}

	if _, axErr := d.updateObject(); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}
