// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package configuration_test

import (
	"applatix.io/axops/configuration"
	"gopkg.in/check.v1"
	"strings"
	"time"
)

func (s *S) TestCreateConfiguration(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql1",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}

	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)
}

func (s *S) TestGetConfiguration(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql2",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}

	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)

	configs, err := configuration.GetConfigurationsByUserName("admin@internal", "mysql2")
	c.Assert(err, check.IsNil)

	c.Assert(len(configs), check.Equals, 1)
	c.Assert(configs[0].ConfigurationName, check.Equals, "mysql2")
}

func (s *S) TestUpdateConfiguration(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql3",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}

	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)

	config2 := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql3",
		ConfigurationValue:       map[string]string{"username": "admin1", "password": "password2"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
	}

	err = configuration.UpdateConfiguration(config2)
	c.Assert(err, check.IsNil)

	configs, err := configuration.GetConfigurationsByUserName("admin@internal", "mysql3")
	c.Assert(err, check.IsNil)

	c.Assert(len(configs), check.Equals, 1)
	value_map := configs[0].ConfigurationValue
	c.Assert(value_map["username"], check.Equals, "admin1")
}

func (s *S) TestDeleteConfiguration(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql4",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}

	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)

	err = configuration.DeleteConfiguration(config)
	c.Assert(err, check.IsNil)

	configs, err := configuration.GetConfigurationsByUserName("admin@internal", "mysql4")
	c.Assert(err, check.IsNil)

	c.Assert(len(configs), check.Equals, 0)
}

func (s *S) TestConfigurationValidationGood(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql5",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}
	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)

	res, err := configuration.ProcessConfigurationStr("%%config.admin@internal.mysql5.username%%")
	c.Assert(err, check.IsNil)
	c.Assert(res, check.NotNil)
	c.Assert(res, check.Equals, "admin")
	err = configuration.DeleteConfiguration(config)
	c.Assert(err, check.IsNil)
}

func (s *S) TestConfigurationValidationBad(c *check.C) {
	config := "%%config.admin@internal.mysql5%%"
	_, err := configuration.ProcessConfigurationStr(config)
	c.Assert(err, check.NotNil)
	c.Assert(strings.Contains(err.Message, "is an invalid configuration variable expression"), check.Equals, true)
}

func (s *S) TestConfigurationValidationBad2(c *check.C) {
	config := "%%config.admin@internal.test.sql%%"
	_, err := configuration.ProcessConfigurationStr(config)
	c.Assert(err, check.NotNil)
	c.Assert(strings.Contains(err.Message, "Configuration does not exist with user admin@internal, name test and key sql"), check.Equals, true)
}

func (s *S) TestConfigurationValidationBad3(c *check.C) {
	config := &configuration.ConfigurationData{
		ConfigurationUser:        "admin@internal",
		ConfigurationName:        "mysql6",
		ConfigurationValue:       map[string]string{"username": "admin", "password": "password"},
		ConfigurationDesc:        "test configuration",
		ConfigurationLastUpdated: time.Now().Unix(),
		ConfigurationDateCreated: time.Now().Unix(),
	}
	err := configuration.CreateConfiguration(config)
	c.Assert(err, check.IsNil)

	configstr := "%%config.admin@internal.mysql6.wrongkey%%"
	_, err = configuration.ProcessConfigurationStr(configstr)
	c.Assert(err, check.NotNil)
	c.Assert(strings.Contains(err.Message, "Configuration exists under user admin@internal, name mysql6 but does not have key wrongkey"), check.Equals, true)
}
