package axam

import (
	"fmt"
	"time"

	"applatix.io/axamm/deployment"
	"applatix.io/axerror"
	"applatix.io/restcl"
	"applatix.io/template"
)

var TEST_ENABLED bool = false

func EnableTest() {
	TEST_ENABLED = true
}

var MaxRetryDuration time.Duration = 10 * time.Minute
var retryConfig *restcl.RetryConfig = &restcl.RetryConfig{
	Timeout: MaxRetryDuration,
}

func PingAM(app string) *axerror.AXError {

	if TEST_ENABLED {
		return nil
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", app), time.Second*3)
	data := map[string]interface{}{}
	axErr, _ := amClient.GetWithTimeRetry("ping", nil, &data, retryConfig)
	return axErr
}

func PostAmDeployment(d *deployment.Deployment, r *deployment.Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	return amClient.PostWithTimeRetry("deployments", nil, d, r, retryConfig)
}

func DeleteAmDeployment(d *deployment.Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	return amClient.DeleteWithTimeRetry("deployments/"+d.Id, nil, d, nil, retryConfig)
}

func ScaleAmDeployment(d *deployment.Deployment, scale *template.Scale) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	return amClient.PostWithTimeRetry("deployments/"+d.Id+"/scale", nil, scale, nil, retryConfig)
}

func UpdateAmDeployment(d *deployment.Deployment) (*deployment.Deployment, *axerror.AXError, int) {

	if TEST_ENABLED {
		return d, nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	var deploy deployment.Deployment
	if axErr, code := amClient.PutWithTimeRetry("deployments/"+d.Id, nil, d, &deploy, retryConfig); axErr != nil {
		return nil, axErr, code
	} else {
		return &deploy, axErr, code
	}
}

func StopAmDeployment(d *deployment.Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	return amClient.PostWithTimeRetry("deployments/"+d.Id+"/stop", nil, nil, nil, retryConfig)
}

func StartAmDeployment(d *deployment.Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	amClient := restcl.NewRestClientWithTimeout(fmt.Sprintf("http://axam.%v:8968/v1", d.ApplicationName), time.Minute*5)
	return amClient.PostWithTimeRetry("deployments/"+d.Id+"/start", nil, nil, nil, retryConfig)
}
