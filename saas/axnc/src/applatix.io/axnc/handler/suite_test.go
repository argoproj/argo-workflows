package handler

import (
	"gopkg.in/check.v1"

	"applatix.io/common"
	"testing"
)

type S struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&S{})

func (s *S) SetUpSuite(c *check.C) {
	common.InitLoggers("axnc")
}

func (s *S) TearDownSuite(c *check.C) {}
