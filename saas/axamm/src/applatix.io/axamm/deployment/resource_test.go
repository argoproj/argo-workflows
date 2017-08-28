package deployment_test

import (
	"fmt"
	"strconv"

	"applatix.io/axamm/deployment"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/template"
	"gopkg.in/check.v1"
)

func createDeployment(cpu, mem float64, scale, maxSurge, maxUnavailable int, useRollingUpdate bool) *deployment.Deployment {
	ctn1 := &service.EmbeddedContainerTemplate{
		ContainerTemplate: &template.ContainerTemplate{},
	}
	ctn1.Type = template.TemplateTypeContainer
	ctn1.Name = "test-ctn1"
	ctn1.ID = common.GenerateUUIDv1()
	ctn1.Image = "%%image1%%"
	ctn1.Command = []string{"%%cmd1%%"}
	ctn1.Resources = &template.ContainerResources{
		CPUCores: template.NumberOrString(fmt.Sprintf("%f", cpu)),
		MemMiB:   template.NumberOrString(fmt.Sprintf("%f", mem)),
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
	t.Scale = &template.Scale{
		Min: scale,
	}
	t.Containers = map[string]*service.Service{
		"ctn1": &service.Service{
			Template: ctn1,
		},
	}
	if useRollingUpdate {
		t.Strategy = &template.Strategy{
			Type: template.StrategyRollingUpdate,
			RollingUpdate: &template.RollingUpdateStrategy{
				MaxUnavailable: strconv.Itoa(maxUnavailable),
				MaxSurge:       strconv.Itoa(maxSurge),
			},
		}
	} else {
		t.Strategy = &template.Strategy{
			Type: template.StrategyRecreate,
		}
	}

	dep := deployment.Deployment{
		deployment.Base{
			Id:       common.GenerateUUIDv1(),
			Name:     "test-deployment",
			Template: t,
			Arguments: map[string]*string{
				"image1": utils.NewString("image1"),
				"image2": utils.NewString("image2"),
				"cmd1":   utils.NewString("cmd1"),
				"cmd2":   utils.NewString("cmd2"),
			},
		}, "", "", "", "", "", nil, "", nil, nil, nil,
	}
	return &dep

}

func (s *S) TestGetMaxResourcesForUpgrade(c *check.C) {

	// for recreate
	d1 := createDeployment(1, 200, 5, 2, 0, false)
	d2 := createDeployment(2, 100, 5, 2, 0, false)
	cpu, mem := d1.GetMaxResourcesForUpgrade(d2)
	c.Assert(cpu, check.Equals, float64(10))
	c.Assert(mem, check.Equals, float64(1000))

	// for rollingupdate
	d1 = createDeployment(1, 100, 5, 1, 1, true)
	d2 = createDeployment(1, 100, 5, 1, 1, true)
	cpu, mem = d1.GetMaxResourcesForUpgrade(d2)
	c.Assert(cpu, check.Equals, float64(6))
	c.Assert(mem, check.Equals, float64(600))

	d1 = createDeployment(1, 100, 5, 2, 0, true)
	d2 = createDeployment(1, 100, 5, 2, 0, true)
	cpu, mem = d1.GetMaxResourcesForUpgrade(d2)
	c.Assert(cpu, check.Equals, float64(7))
	c.Assert(mem, check.Equals, float64(700))

	d1 = createDeployment(2, 1, 5, 2, 1, true)
	d2 = createDeployment(1, 2, 6, 3, 1, true) // target's strategy shouldn't matter
	cpu, mem = d1.GetMaxResourcesForUpgrade(d2)
	c.Assert(cpu, check.Equals, float64(11))
	c.Assert(mem, check.Equals, float64(13))

}
