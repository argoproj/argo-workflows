package deployment_test

import (
	"applatix.io/axamm/deployment"
	"applatix.io/axops/service"
	"applatix.io/common"
	"applatix.io/template"
	"gopkg.in/check.v1"
)

func createDeployment(cpu, mem float64, scale, maxSurge, maxUnavailable int, useRollingUpdate bool) *deployment.Deployment {
	ctn1 := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeContainer,
		Name:    "test-ctn1",
		Id:      common.GenerateUUIDv1(),
		Container: &service.TemplateContainer{
			ImageURL: "%%image1%%",
			Command:  "%%cmd1%%",
			Resources: &service.TemplateResource{
				CPUCores: cpu,
				MemMiB:   mem,
			},
		},
		Inputs: &service.TemplateInput{
			Parameters: map[string]*service.TemplateParameterInput{
				"image1": nil,
				"cmd1":   nil,
			},
		},
	}

	t := &service.ServiceTemplate{
		Type:    service.DocTypeServiceTemplate,
		Subtype: service.TemplateTypeDeployment,
		Name:    "test",
		Id:      common.GenerateUUIDv1(),
		Scale: &template.Scale{
			Min: scale,
		},
		Containers: []service.ServiceMap{
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
				},
			},
			map[string]*service.Service{
				"ctn1": &service.Service{
					Template: ctn1,
				},
			},
		},
	}
	if useRollingUpdate {
		t.Strategy = &service.TemplateStrategy{
			Type: service.StrategyRollingUpdate,
			RollingUpdate: &service.RollingUpdateStrategy{
				MaxUnavailable: maxUnavailable,
				MaxSurge:       maxSurge,
			},
		}
	} else {
		t.Strategy = &service.TemplateStrategy{
			Type: service.StrategyRecreate,
		}
	}

	return &deployment.Deployment{deployment.Base{Id: common.GenerateUUIDv1(), Name: "test-deployment", Template: t, Parameters: map[string]interface{}{"image1": "image1", "image2": "image2", "cmd1": "cmd1", "cmd2": "cmd2"}}, "", "", "", "", "", nil, "", nil, nil, nil}

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
