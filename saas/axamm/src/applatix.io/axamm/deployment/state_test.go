package deployment_test

/*
import (
	"applatix.io/axamm/deployment"
	"applatix.io/axamm/utils"
	"applatix.io/axops/service"
	"applatix.io/common"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestStates(c *check.C) {

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

	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateInit)

	err, _ = d.MarkWaiting(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateWaiting)

	err, _ = d.MarkActive(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateActive)

	err, _ = d.MarkError(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateError)

	err, _ = d.MarkStopping(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateStopping)

	err, _ = d.MarkStopped(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateStopped)

	err, _ = d.MarkTerminating(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateTerminating)

	err, _ = d.MarkTerminated(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateTerminated)

	err, _ = d.MarkUpgrading(nil)
	c.Assert(err, check.IsNil)
	d, err = deployment.GetDeploymentByID(d.Id, true)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.NotNil)
	c.Assert(d.Status, check.Equals, deployment.DeployStateUpgrading)

}
*/
