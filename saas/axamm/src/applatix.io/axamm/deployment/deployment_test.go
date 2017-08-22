package deployment_test

/*
import (
	"fmt"

	"math"

	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axops/service"
	"applatix.io/common"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateDeleteDeployment(c *check.C) {

	appName := utils.APPLICATION_NAME
	deployName := TEST_PREFIX + "-" + "deployment-" + test.RandStr()

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
	}

	d := &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: deployName, Template: t}, "", "", "", "", "", nil, "", nil, nil, nil}
	d.ApplicationName = appName
	d.ApplicationID = common.GenerateUUIDv5(appName)
	d.ApplicationGeneration = common.GenerateUUIDv1()
	d.DeploymentID = common.GenerateUUIDv5(d.ApplicationName + "$$" + d.Name)
	d.Status = deployment.DeployStateInit

	err := d.CreateObject(nil)
	c.Assert(err, check.IsNil)

	err, _ = d.Create()
	c.Assert(err, check.IsNil)

	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)

	err, _ = d.Delete(nil)
	c.Assert(err, check.IsNil)

	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateTerminated)
}

func (s *S) TestParamSubstitutionWithoutFixture(c *check.C) {

	ctn1 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn1",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image1%%",
			Command:  "%%cmd1%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image1": nil,
				"cmd1":   nil,
			},
		},
	}

	ctn2 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn2",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image2%%",
			Command:  "%%cmd2%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image2": nil,
				"cmd2":   nil,
			},
		},
	}

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
		Containers: []service.ServiceMap{
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
				},
				"ctn2": &service.Service{
					Template: ctn2,
				},
			},
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
				},
			},
		},
	}

	d := &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: "test-deployment", Template: t, Parameters: map[string]interface{}{"image1": "image1", "image2": "image2", "cmd1": "cmd1", "cmd2": "cmd2"}}, "", "", "", "", "", nil, "", nil, nil, nil}

	err := d.PreProcess()
	c.Assert(err, check.IsNil)
	d, err = d.Substitute()
	c.Assert(err, check.IsNil)

	c.Assert(d.Template.Containers[0]["ctn1"].Template.Container.ImageURL, check.Equals, "image1")
	c.Assert(d.Template.Containers[0]["ctn1"].Template.Container.Command, check.Equals, "cmd1")
	c.Assert(d.Template.Containers[0]["ctn2"].Template.Container.ImageURL, check.Equals, "image2")
	c.Assert(d.Template.Containers[0]["ctn2"].Template.Container.Command, check.Equals, "cmd2")
	c.Assert(d.Template.Containers[1]["ctn1"].Template.Container.ImageURL, check.Equals, "image1")
	c.Assert(d.Template.Containers[1]["ctn1"].Template.Container.Command, check.Equals, "cmd1")

}

func (s *S) TestParamSubstitutionWithFixture(c *check.C) {

	ctn1 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn1",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image1%%",
			Command:  "%%cmd1%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image1": nil,
				"cmd1":   nil,
			},
		},
	}

	ctn2 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn2",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image2%%",
			Command:  "%%cmd2%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image2": nil,
				"cmd2":   nil,
			},
		},
	}

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
		Containers: []service.ServiceMap{
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
					Parameters: map[string]interface{}{
						"image1": "%%fixtures.fix1.key1%%",
						"cmd1":   "%%fixtures.fix1.key2%%",
					},
				},
				"ctn2": &service.Service{
					Template: ctn2,
					Parameters: map[string]interface{}{
						"image2": "%%fixtures.fix2.key1%%",
						"cmd2":   "%%fixtures.fix2.key2%%",
					},
				},
			},
			map[string]*service.Service{
				"ctn3": &service.Service{
					Template: ctn1,
				},
			},
		},
	}

	d := &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: "test-deployment", Template: t, Parameters: map[string]interface{}{"image1": "image3", "cmd1": "cmd3"}}, "", "", "", "", "", nil, "", nil, nil, nil}

	d.Fixtures = map[string]map[string]interface{}{
		"fix1": map[string]interface{}{
			"key1": "image1",
			"key2": "cmd1",
		},
		"fix2": map[string]interface{}{
			"key1": "image2",
			"key2": "cmd2",
		},
	}

	err := d.PreProcess()
	c.Assert(err, check.IsNil)
	d, err = d.Substitute()
	c.Assert(err, check.IsNil)

	c.Assert(d.Template.Containers[0]["ctn1"].Template.Container.ImageURL, check.Equals, "image1")
	c.Assert(d.Template.Containers[0]["ctn1"].Template.Container.Command, check.Equals, "cmd1")
	c.Assert(d.Template.Containers[0]["ctn2"].Template.Container.ImageURL, check.Equals, "image2")
	c.Assert(d.Template.Containers[0]["ctn2"].Template.Container.Command, check.Equals, "cmd2")
	c.Assert(d.Template.Containers[1]["ctn3"].Template.Container.ImageURL, check.Equals, "image3")
	c.Assert(d.Template.Containers[1]["ctn3"].Template.Container.Command, check.Equals, "cmd3")

}

func (s *S) TestCIDRvalidation(c *check.C) {
	c.Assert(common.ValidateCIDR("192.168.0.1/32"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1/"), check.Equals, false)
	c.Assert(common.ValidateCIDR("192.168.0.1/0"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1/1"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1/10"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1/32"), check.Equals, true)
	c.Assert(common.ValidateCIDR("192.168.0.1/33"), check.Equals, false)
	c.Assert(common.ValidateCIDR("192.168.33"), check.Equals, false)
	c.Assert(common.ValidateCIDR("192.33"), check.Equals, false)
	c.Assert(common.ValidateCIDR("33"), check.Equals, false)
	c.Assert(common.ValidateCIDR(".33"), check.Equals, false)
}

func (s *S) TestStartStopScaleDeployment(c *check.C) {

	appName := utils.APPLICATION_NAME
	deployName := TEST_PREFIX + "-" + "deployment-" + test.RandStr()

	ctn1 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn1",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image1%%",
			Command:  "%%cmd1%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image1": nil,
				"cmd1":   nil,
			},
		},
	}

	ctn2 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn2",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image2%%",
			Command:  "%%cmd2%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image2": nil,
				"cmd2":   nil,
			},
		},
	}

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
		Containers: []service.ServiceMap{
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
					Parameters: map[string]interface{}{
						"image1": "%%fixtures.fix1.key1%%",
						"cmd1":   "%%fixtures.fix1.key2%%",
					},
				},
				"ctn2": &service.Service{
					Template: ctn2,
					Parameters: map[string]interface{}{
						"image2": "%%fixtures.fix2.key1%%",
						"cmd2":   "%%fixtures.fix2.key2%%",
					},
				},
			},
			map[string]*service.Service{
				"ctn3": &service.Service{
					Template: ctn1,
				},
			},
		},
		Scale: &template.Scale{
			Min: 1,
		},
		Annotations: map[string]string{
			deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi", "cpu_cores": 100, "mem_mib": 10000}`,
		},
	}

	d := &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: deployName, Template: t}, "", "", "", "", "", nil, "", nil, nil, nil}
	d.Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi", "cpu_cores": 100, "mem_mib": 10000}`,
	}
	d.ApplicationName = appName
	d.ApplicationID = common.GenerateUUIDv5(appName)
	d.ApplicationGeneration = common.GenerateUUIDv1()
	d.DeploymentID = common.GenerateUUIDv5(d.ApplicationName + "$$" + d.Name)
	d.Status = deployment.DeployStateInit

	err := d.CreateObject(nil)
	c.Assert(err, check.IsNil)

	// Init -> Stop: No
	err, _ = d.Stop()
	c.Assert(err, check.NotNil)

	// Init -> Scale: No
	err, _ = d.Scale(&template.Scale{Min: 100})
	c.Assert(err, check.NotNil)

	err, _ = d.Create()
	c.Assert(err, check.IsNil)

	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)

	// Waiting -> Scale: Yes
	err, _ = d.Scale(&template.Scale{Min: 10})
	c.Assert(err, check.IsNil)
	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)
	c.Assert(d.Template.Scale.Min, check.Equals, 10)
	c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

	// Waiting -> Stop: Yes
	err, _ = d.Stop()
	c.Assert(err, check.IsNil)
	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateStopped)
	// Scale and resource setting won't be changed
	c.Assert(d.Template.Scale.Min, check.Equals, 10)
	c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

	// Stopped -> Scale: No
	err, _ = d.Scale(&template.Scale{Min: 100})
	c.Assert(err, check.NotNil)

	// Stopped -> Start: Yes
	err, _ = d.Start()
	c.Assert(err, check.IsNil)
	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)
	c.Assert(d.Template.Scale.Min, check.Equals, 10)
	c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

	// Stop
	err, _ = d.Stop()
	c.Assert(err, check.IsNil)

	// Stop -> Terminated: Yes
	err, _ = d.Delete(nil)
	c.Assert(err, check.IsNil)

	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateTerminated)
}

func (s *S) TestDinDDeploymentResource(c *check.C) {

	appName := utils.APPLICATION_NAME
	deployName := TEST_PREFIX + "-" + "deployment-" + test.RandStr()

	ctn1 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn1",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image1%%",
			Command:  "%%cmd1%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image1": nil,
				"cmd1":   nil,
			},
		},
	}

	ctn2 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn2",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image2%%",
			Command:  "%%cmd2%%",
			Resources: &service.TemplateResource{
				CPUCores: 0.1,
				MemMiB:   100,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image2": nil,
				"cmd2":   nil,
			},
		},
	}

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
		Containers: []service.ServiceMap{
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
					Parameters: map[string]interface{}{
						"image1": "%%fixtures.fix1.key1%%",
						"cmd1":   "%%fixtures.fix1.key2%%",
					},
				},
				"ctn2": &service.Service{
					Template: ctn2,
					Parameters: map[string]interface{}{
						"image2": "%%fixtures.fix2.key1%%",
						"cmd2":   "%%fixtures.fix2.key2%%",
					},
				},
			},
			map[string]*service.Service{
				"ctn3": &service.Service{
					Template: ctn1,
				},
			},
		},
		Scale: &template.Scale{
			Min: 1,
		},
		Annotations: map[string]string{
			deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi"}`,
		},
	}

	d := &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: deployName, Template: t}, "", "", "", "", "", nil, "", nil, nil, nil}
	d.Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi"}`,
	}
	d.ApplicationName = appName
	d.ApplicationID = common.GenerateUUIDv5(appName)
	d.ApplicationGeneration = common.GenerateUUIDv1()
	d.DeploymentID = common.GenerateUUIDv5(d.ApplicationName + "$$" + d.Name)
	d.Status = deployment.DeployStateInit

	err := d.CreateObject(nil)
	c.Assert(err, check.IsNil)

	err, _ = d.Create()
	c.Assert(err, check.IsNil)

	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)

	// Waiting -> Scale: Yes
	err, _ = d.Scale(&template.Scale{Min: 10})
	c.Assert(err, check.IsNil)
	d, err = deployment.GetLatestDeploymentByID(d.Id, false)
	fmt.Println(*d)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)
	c.Assert(d.Template.Scale.Min, check.Equals, 10)
	c.Assert(math.Abs(d.CPU-10.0*(2*0.1*2+deployment.ResourceCpuOverhead)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	c.Assert(math.Abs(d.Mem-10.0*(2*100*2+deployment.ResourceMemOverhead)) < 0.01, check.Equals, true)
}
*/
