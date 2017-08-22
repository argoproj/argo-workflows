// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

//import (
//	"applatix.io/axops/branch"
//	"applatix.io/test"
//	"gopkg.in/check.v1"
//	"time"
//)
//
//func (s *S) TestBranchGetList(c *check.C) {
//
//	randStr := "rand" + test.RandStr()
//	b1 := &branch.Branch{
//		ID:      randStr,
//		Name:    randStr,
//		Repo:    randStr,
//		Project: randStr,
//	}
//
//	randStr = "rand" + test.RandStr()
//	b2 := &branch.Branch{
//		ID:      randStr,
//		Name:    randStr,
//		Repo:    randStr,
//		Project: randStr,
//	}
//
//	err := b1.Update()
//	c.Check(err, check.IsNil)
//
//	err = b2.Update()
//	c.Check(err, check.IsNil)
//
//	time.Sleep(time.Second)
//
//	for _, field := range []string{branch.BranchName, branch.BranchRepo, branch.BranchProject} {
//		params := map[string]interface{}{
//			field: b1.Name,
//		}
//
//		// Filter by name
//		data := &GeneralGetResult{}
//		err = axopsClient.Get("branches", params, data)
//		c.Assert(err, check.IsNil)
//		c.Assert(data, check.NotNil)
//		c.Assert(len(data.Data), check.Equals, 1)
//
//		// Search by name to get one
//		params = map[string]interface{}{
//			field: "~" + b1.Name,
//		}
//		err = axopsClient.Get("branches", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data), check.Equals, 1)
//
//		// Search by name to get two
//		params = map[string]interface{}{
//			field: "~rand",
//		}
//		err = axopsClient.Get("branches", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) >= 2, check.Equals, true)
//
//		// Search by name to get two with limit one
//		params = map[string]interface{}{
//			field:   "~rand",
//			"limit": 1,
//		}
//		err = axopsClient.Get("branches", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) == 1, check.Equals, true)
//
//		// Search to get one
//		params = map[string]interface{}{
//			"search": "~" + b1.Name,
//		}
//		err = axopsClient.Get("branches", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) == 1, check.Equals, true)
//
//		// Search to get more
//		params = map[string]interface{}{
//			"search": "~rand",
//		}
//		err = axopsClient.Get("branches", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) >= 2, check.Equals, true)
//	}
//}
