// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package core

import (
	"applatix.io/axdb"
	"gopkg.in/check.v1"
	"strings"
	"testing"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

func (s *S) SetUpSuite(c *check.C) {
	InitLoggers()
}

func (s *S) TestGetOrderByClaus(c *check.C) {
	tableName := "TestGetOrderByClaus"
	columns := make(map[string]axdb.Column)
	columns["prim"] = axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition}
	columns["clus1"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
	columns["clus2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
	columns["clus3"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
	columns["clus4"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexClustering}
	columns["val1"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	columns["val2"] = axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone}
	axdbT := axdb.Table{AppName: "test", Name: tableName, Type: axdb.TableTypeKeyValue, Columns: columns}
	tb := &KeyValueTable{Table{axdbT, nil, "", nil, nil, 0, 0}}

	params := map[string]interface{}{
		axdb.AXDBQueryOrderByASC: []interface{}{"clus1"},
	}
	orderstr := tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 ASC")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByDESC: []interface{}{"clus1"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 DESC")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC: []interface{}{"clus1", "clus2"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 ASC, clus2 ASC")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByDESC: []interface{}{"clus1", "clus2"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 DESC, clus2 DESC")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC:  []interface{}{"clus1"},
		axdb.AXDBQueryOrderByDESC: []interface{}{"clus2"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 ASC, clus2 DESC")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC:  []interface{}{"clus1"},
		axdb.AXDBQueryOrderByDESC: []interface{}{"clus2", "clus3"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 ASC, clus2 DESC, clus3 DESC")

	params = map[string]interface{}{}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC:  []interface{}{"val1"},
		axdb.AXDBQueryOrderByDESC: []interface{}{"val2"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "")

	params = map[string]interface{}{
		axdb.AXDBQueryOrderByASC:  []interface{}{"val1", "clus1"},
		axdb.AXDBQueryOrderByDESC: []interface{}{"val2", "clus2"},
	}
	orderstr = tb.getOrderByClause(params)
	verifyString(c, orderstr, "ORDER BY clus1 ASC, clus2 DESC")
}

func verifyString(c *check.C, actual, expected string) {
	c.Assert(strings.TrimSpace(actual), check.Equals, strings.TrimSpace(expected))
}
