package application_test

import (
	"applatix.io/common"
	"gopkg.in/check.v1"
)

func (s *S) TestValidateApplicationName(c *check.C) {
	c.Assert(common.ValidateKubeObjName(""), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("-"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("a-"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("-a"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("a&a"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("a?a"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("a?A"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("AAA"), check.Equals, false)
	c.Assert(common.ValidateKubeObjName("aAa"), check.Equals, false)

	c.Assert(common.ValidateKubeObjName("a"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("aa"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("aaa"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("a-a"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("a--a"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("1--a"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("0--a"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("0--0"), check.Equals, true)
	c.Assert(common.ValidateKubeObjName("applatix"), check.Equals, true)
}
