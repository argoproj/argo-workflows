// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

//import (
//	"applatix.io/axops/commit"
//	"applatix.io/test"
//	"gopkg.in/check.v1"
//	"time"
//)
//
//func (s *S) TestCommitGetList(c *check.C) {
//
//
//	randStr := "rand" + test.RandStr()
//	b1 := &commit.Commit{
//		Revision:    randStr,
//		Repo:        randStr,
//		Branch:      []string{randStr},
//		Author:      randStr,
//		Committer:   randStr,
//		Description: randStr,
//		Date:        time.Now().Unix() * 1e6,
//	}
//
//	randStr = "rand" + test.RandStr()
//	b2 := &commit.Commit{
//		Revision:    randStr,
//		Repo:        randStr,
//		Branch:      []string{randStr},
//		Author:      randStr,
//		Committer:   randStr,
//		Description: randStr,
//		Date:        time.Now().Unix() * 1e6,
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
//	for _, field := range []string{commit.CommitRepo, commit.CommitRevision, commit.CommitBranch, commit.CommitCommitter, commit.CommitAuthor, commit.CommitDescription} {
//		params := map[string]interface{}{
//			field: b1.Description,
//		}
//
//		// Filter by name
//		data := &GeneralGetResult{}
//		err = axopsClient.Get("commits", params, data)
//		c.Assert(err, check.IsNil)
//		c.Assert(data, check.NotNil)
//		c.Assert(len(data.Data), check.Equals, 1)
//
//		// Search by name to get one
//		params = map[string]interface{}{
//			field: "~" + b1.Description,
//		}
//		err = axopsClient.Get("commits", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data), check.Equals, 1)
//
//		// Search by name to get two
//		params = map[string]interface{}{
//			field: "~rand",
//		}
//		err = axopsClient.Get("commits", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) >= 2, check.Equals, true)
//
//		// Search by name to get two with limit one
//		params = map[string]interface{}{
//			field:   "~rand",
//			"limit": 1,
//		}
//		err = axopsClient.Get("commits", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) == 1, check.Equals, true)
//
//		// Search to get one
//		params = map[string]interface{}{
//			"search": "~" + b1.Description,
//		}
//		err = axopsClient.Get("commits", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) == 1, check.Equals, true)
//
//		// Search to get more
//		params = map[string]interface{}{
//			"search": "~rand",
//		}
//		err = axopsClient.Get("commits", params, &data)
//		c.Assert(err, check.IsNil)
//		c.Assert(len(data.Data) >= 2, check.Equals, true)
//	}
//}
