package deployment_test

import (
	"fmt"

	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axops/service"
	axoputils "applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestCreateDeleteDeployment(c *check.C) {

	appName := utils.APPLICATION_NAME
	deployName := TEST_PREFIX + "-" + "deployment-" + test.RandStr()

	t := &service.EmbeddedDeploymentTemplate{
		DeploymentTemplate: &template.DeploymentTemplate{},
	}
	t.Type = template.TemplateTypeDeployment
	t.Name = "test"
	t.ID = common.GenerateUUIDv1()

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

func newDeployment() *deployment.Deployment {
	ctn1 := &service.EmbeddedContainerTemplate{
		ContainerTemplate: &template.ContainerTemplate{},
	}
	ctn1.Type = template.TemplateTypeContainer
	ctn1.Name = "test-ctn1"
	ctn1.ID = common.GenerateUUIDv1()
	ctn1.Image = "%%inputs.parameters.image1%%"
	ctn1.Command = []string{"%%inputs.parameters.cmd1%%"}
	ctn1.Resources = &template.ContainerResources{
		CPUCores: "0.1",
		MemMiB:   "100",
	}
	ctn1.Inputs = &template.Inputs{
		Parameters: map[string]*template.InputParameter{
			"image1": nil,
			"cmd1":   nil,
		},
	}

	t := &service.EmbeddedDeploymentTemplate{
		DeploymentTemplate: &template.DeploymentTemplate{},
	}
	t.Type = template.TemplateTypeDeployment
	t.Name = "test"
	t.ID = common.GenerateUUIDv1()
	t.Inputs = &template.Inputs{
		Parameters: map[string]*template.InputParameter{
			"image1": nil,
			"cmd1":   nil,
		},
	}
	t.Containers = map[string]*service.Service{
		"ctn1": &service.Service{
			Template: ctn1,
			Arguments: map[string]*string{
				"parameters.image1": axoputils.NewString("%%inputs.parameters.image1%%"),
				"parameters.cmd1":   axoputils.NewString("%%inputs.parameters.cmd1%%"),
			},
		},
	}

	d := deployment.Deployment{
		deployment.Base{
			Id:       common.GenerateUUIDv1(),
			Name:     "test-deployment",
			Template: t,
			Arguments: map[string]*string{
				"parameters.image1": axoputils.NewString("image1"),
				"parameters.cmd1":   axoputils.NewString("cmd1"),
			},
		}, "", "", "", "", "", nil, "", nil, nil, nil,
	}
	return &d
}

func (s *S) TestParamSubstitutionWithoutFixture(c *check.C) {
	d := newDeployment()
	err := d.PreProcess()
	c.Assert(err, check.IsNil)
	d, err = d.Substitute()
	c.Assert(err, check.IsNil)

	ctrTmpl := d.Template.Containers["ctn1"].Template.(*service.EmbeddedContainerTemplate)
	c.Assert(ctrTmpl.Image, check.Equals, "image1")
	c.Assert(ctrTmpl.Command[0], check.Equals, "cmd1")
}

func (s *S) TestParamSubstitutionWithFixture(c *check.C) {
	d := newDeployment()
	d.Template.Containers["ctn1"].Arguments = map[string]*string{
		"parameters.image1": axoputils.NewString("%%fixtures.fix1.key1%%"),
		"parameters.cmd1":   axoputils.NewString("%%fixtures.fix2.key2%%"),
	}

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

	ctrTmpl := d.Template.Containers["ctn1"].Template.(*service.EmbeddedContainerTemplate)
	c.Assert(ctrTmpl.Image, check.Equals, "image1")
	c.Assert(ctrTmpl.Command[0], check.Equals, "cmd2")

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

	d := newDeployment()
	d.Template.Scale = &template.Scale{
		Min: 1,
	}
	d.Template.Containers["ctn1"].Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi", "cpu_cores": 100, "mem_mib": 10000}`,
	}

	d.Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi", "cpu_cores": 100, "mem_mib": 10000}`,
	}
	d.Name = deployName
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
	//c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	//c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

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
	//c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	//c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

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
	//c.Assert(math.Abs(d.CPU-10.0*(2*0.1+deployment.ResourceCpuOverhead+100)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	//c.Assert(math.Abs(d.Mem-10.0*(2*100+deployment.ResourceMemOverhead+10000)) < 0.01, check.Equals, true)

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

	d := newDeployment()
	d.Template.Scale = &template.Scale{
		Min: 1,
	}
	d.Template.Containers["ctn1"].Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi"}`,
	}

	d.Annotations = map[string]string{
		deployment.DockerEnabledKey: `{ "graph-storage-name": "deploymentstorage", "graph-storage-size": "30Gi"}`,
	}

	d.Name = deployName
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
	//c.Assert(math.Abs(d.CPU-10.0*(2*0.1*2+deployment.ResourceCpuOverhead)*deployment.ResourceScaleFactor) < 0.01, check.Equals, true)
	//c.Assert(math.Abs(d.Mem-10.0*(2*100*2+deployment.ResourceMemOverhead)) < 0.01, check.Equals, true)
}
