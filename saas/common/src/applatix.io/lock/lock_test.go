package lock_test

import (
	"time"

	"applatix.io/common"
	"applatix.io/lock"
	"gopkg.in/check.v1"
)

func (s *S) TestTryLock(c *check.C) {
	common.InitLoggers("LOCK")
	group := lock.LockGroup{}
	group.Name = "TestTryLock"
	group.Init()
	group.Lock("a")
	c.Assert(false, check.Equals, group.TryLock("a", time.Second))

	group.Unlock("a")
	c.Assert(true, check.Equals, group.TryLock("a", time.Second))
}

func (s *S) TestLock(c *check.C) {
	common.InitLoggers("LOCK")
	group := lock.LockGroup{}
	group.Name = "TestLock"
	group.Init()
	group.Lock("a")

	go func() {
		group.Lock("a")
		c.FailNow()
	}()

	time.Sleep(time.Second)
}

func (s *S) TestLockTTL(c *check.C) {
	common.InitLoggers("LOCK")
	group := lock.LockGroup{}
	group.Name = "TestLockTTL"
	group.TtlSeconds = 2
	group.Init()
	group.Lock("a")
	group.Lock("b")
	c.Assert(2, check.Equals, group.Size())
	time.Sleep(time.Second * 4)
	c.Assert(0, check.Equals, group.Size())
}
