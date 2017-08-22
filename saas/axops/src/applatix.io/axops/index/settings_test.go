package index_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestSearchIndexCreate(c *check.C) {
	idx := &index.SearchIndex{
		Type:  "type-" + test.RandStr(),
		Key:   "key-" + test.RandStr(),
		Value: "value-" + test.RandStr(),
	}
	idx, err, code := idx.Create()
	c.Assert(err, check.IsNil)
	c.Assert(idx, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(idx.Ctime, check.Not(check.Equals), 0)
	c.Assert(idx.Mtime, check.Not(check.Equals), 0)

	indexes, err := index.GetSearchIndexesByType(idx.Type)
	c.Assert(err, check.IsNil)
	c.Assert(len(indexes), check.Equals, 1)
}

func (s *S) TestSearchIndexDelete(c *check.C) {
	idx := &index.SearchIndex{
		Type:  "type-" + test.RandStr(),
		Key:   "key-" + test.RandStr(),
		Value: "value-" + test.RandStr(),
	}

	idx, err, code := idx.Create()
	c.Assert(err, check.IsNil)
	c.Assert(idx, check.NotNil)
	c.Assert(code, check.Equals, axerror.REST_CREATE_OK)
	c.Assert(idx.Ctime, check.Not(check.Equals), 0)
	c.Assert(idx.Mtime, check.Not(check.Equals), 0)

	indexes, err := index.GetSearchIndexesByType(idx.Type)
	c.Assert(err, check.IsNil)
	c.Assert(len(indexes), check.Equals, 1)

	err, code = idx.Delete()
	c.Assert(err, check.IsNil)
	c.Assert(code, check.Equals, axerror.REST_STATUS_OK)

	indexes, err = index.GetSearchIndexesByType(idx.Type)
	c.Assert(err, check.IsNil)
	c.Assert(len(indexes), check.Equals, 0)
}
