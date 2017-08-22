package rediscl_test

import (
	"testing"

	"applatix.io/rediscl"
	"gopkg.in/check.v1"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

var client *rediscl.RedisClient

func (s *S) SetUpSuite(c *check.C) {
	//c.Skip("Need a redis instance to run the test")
	client = rediscl.NewRedisClient("127.0.0.1:6379", "", 10)
}
