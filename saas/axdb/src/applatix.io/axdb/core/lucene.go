// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package core

import (
	"applatix.io/axdb"
)

const LuceneIndexTemplate = `
CREATE CUSTOM INDEX %s%s ON %s()
USING 'com.stratio.cassandra.lucene.Index'
WITH OPTIONS = {
    'refresh_seconds' : '1',
    'schema' : '%s'
};
`

// Lucene schema for creating lucene index
type LuceneIndexSchema struct {
	Fields map[string]LuceneIndexField `json:"$$ax$$fields$$ax$$,omitempty"`
}

func NewLuceneIndexSchema() *LuceneIndexSchema {
	return &LuceneIndexSchema{
		Fields: make(map[string]LuceneIndexField),
	}
}

func (schema *LuceneIndexSchema) addFields(cols map[string]axdb.Column, excluded map[string]bool) {
	for colName, col := range cols {
		excl := true
		if excluded == nil {
			excl = false
		} else if _, ok := excluded[colName]; !ok {
			excl = false
		}

		if !excl {
			schema.addField(colName, adaptToLuceneIndexType(col.Type))
		}
	}
}

func (schema *LuceneIndexSchema) addField(colName string, luceneType int) {
	if luceneType == -1 {
		return
	}

	var caseSensitive *bool = nil
	if luceneType == axdb.LuceneTypeString {
		caseSensitive = newFalse()
	} else {
		caseSensitive = nil
	}

	schema.Fields["$$ax$$"+colName+"$$ax$$"] = LuceneIndexField{
		Type:          axdbLuceneIndexTypeNames[luceneType],
		Validated:     true,
		CaseSensitive: caseSensitive,
	}
}

func (schema *LuceneIndexSchema) isEmpty() bool {
	return len(schema.Fields) == 0
}

type LuceneIndexField struct {
	Type          string `json:"$$ax$$type$$ax$$,omitempty"`
	Validated     bool   `json:"$$ax$$validated$$ax$$,omitempty"`
	CaseSensitive *bool  `json:"$$ax$$case_sensitive$$ax$$,omitempty"`
}

func newFalse() *bool {
	b := false
	return &b
}
