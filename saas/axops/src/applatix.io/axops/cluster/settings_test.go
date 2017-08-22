package cluster_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/cluster"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestSystemSettingCreate(c *check.C) {
	setting := &cluster.ClusterSetting{
		Key:   "key-" + test.RandStr(),
		Value: "value-" + test.RandStr(),
	}
	setting, err, code := setting.Create()
	c.Assert(err, check.IsNil)
	c.Assert(setting, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(setting.Ctime, check.Not(check.Equals), 0)
	c.Assert(setting.Mtime, check.Not(check.Equals), 0)

	cp, err := cluster.GetClusterSetting(setting.Key)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.Key, check.Equals, setting.Key)
	c.Assert(cp.Value, check.Equals, setting.Value)
}

func (s *S) TestCustomViewUpdate(c *check.C) {
	setting := &cluster.ClusterSetting{
		Key:   "key-" + test.RandStr(),
		Value: "value-" + test.RandStr(),
	}
	setting, err, code := setting.Create()
	c.Assert(err, check.IsNil)
	c.Assert(setting, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(setting.Ctime, check.Not(check.Equals), 0)
	c.Assert(setting.Mtime, check.Not(check.Equals), 0)

	cp, err := cluster.GetClusterSetting(setting.Key)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.Key, check.Equals, setting.Key)
	c.Assert(cp.Value, check.Equals, setting.Value)

	setting.Value = "value-" + test.RandStr()
	setting, err, code = setting.Update()
	c.Assert(err, check.IsNil)
	c.Assert(setting, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_STATUS_OK)

	cp, err = cluster.GetClusterSetting(setting.Key)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.Key, check.Equals, setting.Key)
	c.Assert(cp.Value, check.Equals, setting.Value)
}

func (s *S) TestCustomViewDelete(c *check.C) {
	setting := &cluster.ClusterSetting{
		Key:   "key-" + test.RandStr(),
		Value: "value-" + test.RandStr(),
	}
	setting, err, code := setting.Create()
	c.Assert(err, check.IsNil)
	c.Assert(setting, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(setting.Ctime, check.Not(check.Equals), 0)
	c.Assert(setting.Mtime, check.Not(check.Equals), 0)

	cp, err := cluster.GetClusterSetting(setting.Key)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.NotNil)
	c.Assert(cp.Key, check.Equals, setting.Key)
	c.Assert(cp.Value, check.Equals, setting.Value)

	err, code = setting.Delete()
	c.Assert(err, check.IsNil)
	c.Assert(code, check.Equals, axerror.REST_STATUS_OK)

	cp, err = cluster.GetClusterSetting(setting.Key)
	c.Assert(err, check.IsNil)
	c.Assert(cp, check.IsNil)
}
