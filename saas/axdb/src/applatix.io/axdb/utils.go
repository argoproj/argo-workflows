// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axdb

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func SerializeOrderedMap(data map[string]interface{}) string {
	var keys []string
	for key, _ := range data {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var buf bytes.Buffer
	for _, key := range keys {
		val := data[key]
		buf.WriteString(fmt.Sprintf("%s=%v;", key, val))
	}

	return buf.String()
}

func DeserializeOrderedMap(data string) map[string]interface{} {
	orderedMap := make(map[string]interface{})
	kvs := strings.Split(data, ";")
	kvs = kvs[0 : len(kvs)-1]
	for _, kv := range kvs {
		pair := strings.Split(kv, "=")
		orderedMap[pair[0]] = pair[1]
	}
	return orderedMap
}

func EscapedString(str string) string {
	return strings.Replace(str, "'", "''", -1)
}

// Lucene query for doing search and sorting
type LuceneFilterBase struct {
	Type   string             `json:"$$ax$$type$$ax$$,omitempty"`
	Should []LuceneFilterBase `json:"$$ax$$should$$ax$$,omitempty"`
	Must   []LuceneFilterBase `json:"$$ax$$must$$ax$$,omitempty"`
	Not    []LuceneFilterBase `json:"$$ax$$not$$ax$$,omitempty"`
	Field  string             `json:"$$ax$$field$$ax$$,omitempty"`
	Value  interface{}        `json:"$$ax$$value$$ax$$,omitempty"`
	Values interface{}        `json:"$$ax$$values$$ax$$,omitempty"`

	Lower        int64 `json:"$$ax$$lower$$ax$$,omitempty"`
	Upper        int64 `json:"$$ax$$upper$$ax$$,omitempty"`
	IncludeLower bool  `json:"$$ax$$include_lower$$ax$$,omitempty"`
	IncludeUpper bool  `json:"$$ax$$include_upper$$ax$$,omitempty"`
}

func NewLuceneBooleanFilterBase() *LuceneFilterBase {
	return &LuceneFilterBase{
		Type: "boolean",
	}
}

func NewLuceneWildCardFilterBase(field string, value interface{}) LuceneFilterBase {
	return LuceneFilterBase{
		Type:  "wildcard",
		Field: field,
		Value: value,
	}
}

func NewLuceneRegexpFilterBase(field string, value interface{}) LuceneFilterBase {
	return LuceneFilterBase{
		Type:  "regexp",
		Field: field,
		Value: value,
	}
}

func NewLuceneRangeFilterBase(field string, lower, upper int64) LuceneFilterBase {
	return LuceneFilterBase{
		Type:         "range",
		Field:        field,
		Lower:        lower,
		Upper:        upper,
		IncludeLower: false,
		IncludeUpper: true,
	}
}

func NewLuceneContainsFilterBase(field string, values interface{}) LuceneFilterBase {
	return LuceneFilterBase{
		Type:   "contains",
		Field:  field,
		Values: values,
	}
}

type LuceneSorterBase struct {
	Type    string `json:"$$ax$$type$$ax$$"`
	Field   string `json:"$$ax$$field$$ax$$"`
	Reverse bool   `json:"$$ax$$reverse$$ax$$"`
}

func NewLuceneSorterBase(field string, reverse bool) LuceneSorterBase {
	return LuceneSorterBase{
		Type:    "simple",
		Field:   field,
		Reverse: reverse,
	}
}

type LuceneFilter struct {
	Type   string             `json:"$$ax$$type$$ax$$,omitempty"`
	Should []LuceneFilterBase `json:"$$ax$$should$$ax$$,omitempty"`
	Must   []LuceneFilterBase `json:"$$ax$$must$$ax$$,omitempty"`
	Not    []LuceneFilterBase `json:"$$ax$$not$$ax$$,omitempty"`
}

func (filter *LuceneFilter) addShould(should LuceneFilterBase) {
	// we support 2 levels of nesting for filter
	// top level is assumed to use Must(AND). Everything in Should(OR) and Not is nested inside the Must
	// ex: a AND (b or c)
	var inner *LuceneFilterBase
	for i, v := range filter.Must {
		if len(v.Should) > 0 {
			inner = &filter.Must[i]
			inner.Should = append(inner.Should, should)
			break
		}
	}
	if inner == nil {
		inner = NewLuceneBooleanFilterBase()
		inner.Should = append(inner.Should, should)
		filter.Must = append(filter.Must, *inner)
	}
}

func (filter *LuceneFilter) addMust(must LuceneFilterBase) {
	filter.Must = append(filter.Must, must)
}

func (filter *LuceneFilter) addNot(not LuceneFilterBase) {
	// we support 2 levels of nesting for filter
	// top level is assumed to use Must(AND). Should(OR) and Not is nested inside the Must
	// ex: a AND (b or c)
	var inner *LuceneFilterBase
	for i, v := range filter.Must {
		if len(v.Should) > 0 {
			inner = &filter.Must[i]
			inner.Not = append(inner.Not, not)
			break
		}
	}
	if inner == nil {
		inner = NewLuceneBooleanFilterBase()
		inner.Not = append(inner.Not, not)
		filter.Must = append(filter.Must, *inner)
	}
}

func NewLuceneFilter() *LuceneFilter {
	return &LuceneFilter{
		Type: "boolean",
	}
}

type LuceneSorter struct {
	Fields []LuceneSorterBase `json:"$$ax$$fields$$ax$$,omitempty"`
}

func (sorter *LuceneSorter) addSorter(sorterBase LuceneSorterBase) {
	sorter.Fields = append(sorter.Fields, sorterBase)
}

func NewLuceneSorter() *LuceneSorter {
	return &LuceneSorter{}
}

type LuceneSearch struct {
	Query  *LuceneFilter `json:"$$ax$$query$$ax$$,omitempty"`
	Filter *LuceneFilter `json:"$$ax$$filter$$ax$$,omitempty"`
	Sort   *LuceneSorter `json:"$$ax$$sort$$ax$$,omitempty"`
}

func (search *LuceneSearch) AddQueryShould(should LuceneFilterBase) {
	if search.Query == nil {
		search.Query = NewLuceneFilter()
	}
	search.Query.addShould(should)
}

func (search *LuceneSearch) AddQueryMust(must LuceneFilterBase) {
	if search.Query == nil {
		search.Query = NewLuceneFilter()
	}
	search.Query.addMust(must)
}

func (search *LuceneSearch) AddQueryNot(not LuceneFilterBase) {
	if search.Query == nil {
		search.Query = NewLuceneFilter()
	}
	search.Query.addNot(not)
}

func (search *LuceneSearch) AddFilterMust(must LuceneFilterBase) {
	if search.Filter == nil {
		search.Filter = NewLuceneFilter()
	}
	search.Filter.addMust(must)
}

func (search *LuceneSearch) AddFilterShould(should LuceneFilterBase) {
	if search.Filter == nil {
		search.Filter = NewLuceneFilter()
	}
	search.Filter.addShould(should)
}

func (search *LuceneSearch) AddSorter(sorter LuceneSorterBase) {
	if search.Sort == nil {
		search.Sort = NewLuceneSorter()
	}
	search.Sort.addSorter(sorter)
}

func (search *LuceneSearch) IsValid() bool {
	return search.Filter != nil || search.Query != nil || search.Sort != nil
}

func (search *LuceneSearch) HasSort() bool {
	if search.Sort == nil || len(search.Sort.Fields) == 0 {
		return false
	} else {
		return true
	}
}

func NewLuceneSearch() *LuceneSearch {
	return &LuceneSearch{}
}
