package deployment

import (
	"encoding/json"
	"fmt"
	"time"

	"applatix.io/axamm/adc"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/restcl"
)

var TEST_ENABLED bool = false

func EnableTest() {
	TEST_ENABLED = true
}

var MaxRetryDuration time.Duration = 10 * time.Minute

var retryConfig *restcl.RetryConfig = &restcl.RetryConfig{
	Timeout: MaxRetryDuration,
}

func createKubeDeployment(d *Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	if axErr, code := d.reserveAdcResources(); axErr != nil {
		return axErr, code
	}

	utils.DebugLog.Printf("[AXMON] Creating kube deployment for: %v", d)
	return utils.AxmonCl.PostWithTimeRetry(fmt.Sprintf("axmon/application/%v/deployment", d.ApplicationName), nil, d, nil, retryConfig)
}

func upgradeKubeDeployment(d *Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	// upgrade is just a post to axmon with the new deployment spec
	utils.DebugLog.Printf("[AXMON] Upgrading kube deployment for: %v", d)
	return utils.AxmonCl.PostWithTimeRetry(fmt.Sprintf("axmon/application/%v/deployment", d.ApplicationName), nil, d, nil, retryConfig)

}

func deleteKubeDeployment(d *Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	if axErr, code := adc.Release(d.DeploymentID); axErr != nil {
		return axErr, code
	}
	utils.DebugLog.Printf("[AXMON] Deleting kube deployment for: %v", d)
	return utils.AxmonCl.DeleteWithTimeRetry(fmt.Sprintf("axmon/application/%v/deployment/%s", d.ApplicationName, d.Name), nil, nil, nil, retryConfig)
}

func getKubeDeploymentStatus(d *Deployment) (*DeploymentStatus, *axerror.AXError) {

	if TEST_ENABLED {
		return &DeploymentStatus{}, nil
	}

	var status DeploymentResult

	axErr, _ := utils.AxmonCl.GetWithTimeRetry(fmt.Sprintf("axmon/application/%v/deployment/%s", d.ApplicationName, d.Name), nil, &status, retryConfig)
	if axErr != nil {
		return nil, axErr
	}

	bytes, _ := json.MarshalIndent(status.Result, "", "    ")
	utils.DebugLog.Println(d.ApplicationName, d.Name, string(bytes))

	if status.Result != nil && status.Result.Pods != nil {
		for _, pod := range status.Result.Pods {
			pod.Mtime = time.Now().Unix()
		}
	}

	return status.Result, axErr
}

func scaleKubeDeployment(d *Deployment, replicas int) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	detail := map[string]string{
		"name": d.Name,
		"app":  d.ApplicationName,
	}

	cpu, mem := d.getInstanceResources()
	if axErr, code := adc.Reserve(d.DeploymentID, "deployment", cpu*float64(replicas), mem*float64(replicas), adc.AdcDefaultTtl, detail); axErr != nil {
		return axErr, code
	}

	payload := map[string]interface{}{
		"replicas": replicas,
	}
	utils.DebugLog.Printf("[AXMON] Scaling kube deployment for: %v", d)
	return utils.AxmonCl.PutWithTimeRetry(fmt.Sprintf("axmon/application/%v/deployment/%s/scale", d.ApplicationName, d.Name), nil, payload, nil, retryConfig)
}
