package lcs

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestLCS(t *testing.T) {
	cases := []struct {
		left       []interface{}
		right      []interface{}
		indexPairs []IndexPair
		values     []interface{}
		length     int
	}{
		{
			left:       []interface{}{1, 2, 3},
			right:      []interface{}{2, 3},
			indexPairs: []IndexPair{{1, 0}, {2, 1}},
			values:     []interface{}{2, 3},
			length:     2,
		},
		{
			left:       []interface{}{2, 3},
			right:      []interface{}{1, 2, 3},
			indexPairs: []IndexPair{{0, 1}, {1, 2}},
			values:     []interface{}{2, 3},
			length:     2,
		},
		{
			left:       []interface{}{2, 3},
			right:      []interface{}{2, 5, 3},
			indexPairs: []IndexPair{{0, 0}, {1, 2}},
			values:     []interface{}{2, 3},
			length:     2,
		},
		{
			left:       []interface{}{2, 3, 3},
			right:      []interface{}{2, 5, 3},
			indexPairs: []IndexPair{{0, 0}, {2, 2}},
			values:     []interface{}{2, 3},
			length:     2,
		},
		{
			left:       []interface{}{1, 2, 5, 3, 1, 1, 5, 8, 3},
			right:      []interface{}{1, 2, 3, 3, 4, 4, 5, 1, 6},
			indexPairs: []IndexPair{{0, 0}, {1, 1}, {2, 6}, {4, 7}},
			values:     []interface{}{1, 2, 5, 1},
			length:     4,
		},
		{
			left:       []interface{}{},
			right:      []interface{}{2, 5, 3},
			indexPairs: []IndexPair{},
			values:     []interface{}{},
			length:     0,
		},
		{
			left:       []interface{}{3, 4},
			right:      []interface{}{},
			indexPairs: []IndexPair{},
			values:     []interface{}{},
			length:     0,
		},
		{
			left:       []interface{}{"foo"},
			right:      []interface{}{"baz", "foo"},
			indexPairs: []IndexPair{{0, 1}},
			values:     []interface{}{"foo"},
			length:     1,
		},
		{
			left:       []interface{}{byte('T'), byte('G'), byte('A'), byte('G'), byte('T'), byte('A')},
			right:      []interface{}{byte('G'), byte('A'), byte('T'), byte('A')},
			indexPairs: []IndexPair{{1, 0}, {2, 1}, {4, 2}, {5, 3}},
			values:     []interface{}{byte('G'), byte('A'), byte('T'), byte('A')},
			length:     4,
		},
	}

	for i, c := range cases {
		lcs := New(c.left, c.right)

		actualPairs := lcs.IndexPairs()
		if !reflect.DeepEqual(actualPairs, c.indexPairs) {
			t.Errorf("test case %d failed at index pair, actual: %#v, expected: %#v", i, actualPairs, c.indexPairs)
		}

		actualValues := lcs.Values()
		if !reflect.DeepEqual(actualValues, c.values) {
			t.Errorf("test case %d failed at values, actual: %#v, expected: %#v", i, actualValues, c.values)
		}

		actualLength := lcs.Length()
		if actualLength != c.length {
			t.Errorf("test case %d failed at length, actual: %d, expected: %d", i, actualLength, c.length)
		}
	}
}

func TestContextCancel(t *testing.T) {
	left := make([]interface{}, 100000) // takes over 1 sec
	right := make([]interface{}, 100000)
	right[0] = 1
	right[len(right)-1] = 1
	lcs := New(left, right)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

	_, err := lcs.LengthContext(ctx)
	if err != context.Canceled {
		t.Fatalf("unexpected err: %s", err)
	}
}
