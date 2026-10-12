package parameterization

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestHistoryContinuesBeyondThousandWithoutRewalkingOrphans(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	var err error
	s.HistoryPager, err = NewHistoryPager(key)
	require.NoError(t, err)
	for i := range 1107 {
		_, err = s.Save(ctx, "long", SaveRequest{Template: "typed", Values: map[string]any{"count": i + 1}})
		require.NoError(t, err)
	}
	_, err = s.resource("parameterrevisions").Create(ctx, object("ParameterRevision", "orphan", "lab", Revision{Set: "long", Values: map[string]any{"count": -1}}), metav1.CreateOptions{})
	require.NoError(t, err)
	c := s.Client.(*dynamicfake.FakeDynamicClient)
	seen := []int{}
	token := ""
	pages := 0
	for {
		c.ClearActions()
		page, pageErr := s.History(ctx, "long", 25, token)
		require.NoError(t, pageErr)
		require.LessOrEqual(t, len(c.Actions()), 27, "deep pages read only set, anchor and this page")
		for _, obj := range page.Items {
			var rev Revision
			require.NoError(t, decode(&obj, &rev))
			seen = append(seen, int(rev.Values.(map[string]any)["count"].(float64)))
		}
		pages++
		if pages == 1 {
			_, err = s.Save(ctx, "long", SaveRequest{Template: "typed", Values: map[string]any{"count": 99999}})
			require.NoError(t, err)
			// Another server with the same configured key can continue the query.
			s.HistoryPager, err = NewHistoryPager(key)
			require.NoError(t, err)
		}
		token = page.GetContinue()
		if token == "" {
			break
		}
	}
	require.Equal(t, 45, pages)
	require.Len(t, seen, 1107)
	for i, v := range seen {
		require.Equal(t, 1107-i, v)
	}
	fresh, err := s.History(ctx, "long", 1, "")
	require.NoError(t, err)
	var newest Revision
	require.NoError(t, decode(&fresh.Items[0], &newest))
	require.InDelta(t, float64(99999), newest.Values.(map[string]any)["count"], 0)
}

func TestHistoryCursorRejectsReplayTamperExpirationAndReplacement(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	p, err := NewHistoryPager(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	s.HistoryPager = p
	for i := 1; i <= 3; i++ {
		_, err = s.Save(ctx, "set", SaveRequest{Template: "typed", Values: map[string]any{"count": i}})
		require.NoError(t, err)
	}
	page, err := s.History(ctx, "set", 1, "")
	require.NoError(t, err)
	token := page.GetContinue()
	require.NotEmpty(t, token)
	_, err = s.History(ctx, "set", 1, token+"tampered")
	require.True(t, apierrors.IsResourceExpired(err))
	_, err = s.Save(ctx, "other", SaveRequest{Template: "typed", Values: map[string]any{"count": 1}})
	require.NoError(t, err)
	_, err = s.History(ctx, "other", 1, token)
	require.True(t, apierrors.IsResourceExpired(err))
	p.now = func() time.Time { return time.Now().Add(16 * time.Minute) }
	_, err = s.History(ctx, "set", 1, token)
	require.True(t, apierrors.IsResourceExpired(err))
	p.now = time.Now
	otherKey, _ := NewHistoryPager(bytes.Repeat([]byte{2}, 32))
	s.HistoryPager = otherKey
	_, err = s.History(ctx, "set", 1, token)
	require.True(t, apierrors.IsResourceExpired(err))
	s.HistoryPager = p
	anchor := page.Items[0]
	anchor.SetUID("replacement")
	_, err = s.resource("parameterrevisions").Update(ctx, &anchor, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.History(ctx, "set", 1, token)
	require.True(t, apierrors.IsResourceExpired(err))
	_, err = NewHistoryPager([]byte{})
	require.Error(t, err)
	_, err = NewHistoryPager(nil)
	require.NoError(t, err)
}

func TestHistoryPagesRecheckRBACAndRejectCyclicChains(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	s.HistoryPager, _ = NewHistoryPager(nil)
	for i := 1; i <= 4; i++ {
		_, err := s.Save(ctx, "set", SaveRequest{Template: "typed", Values: map[string]any{"count": i}})
		require.NoError(t, err)
	}
	first, err := s.History(ctx, "set", 1, "")
	require.NoError(t, err)
	s.Client.(*dynamicfake.FakeDynamicClient).PrependReactor("get", "parameterrevisions", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "parameterrevisions"}, a.(clienttesting.GetAction).GetName(), nil)
	})
	_, err = s.History(ctx, "set", 1, first.GetContinue())
	require.True(t, apierrors.IsForbidden(err))
	// Deliberately malformed imported records; the detector spans page boundaries.
	s = testStore()
	s.HistoryPager, _ = NewHistoryPager(nil)
	for i := range 30 {
		_, err = s.resource("parameterrevisions").Create(ctx, object("ParameterRevision", fmt.Sprintf("r-%d", i), "lab", Revision{Set: "cycle", Parent: fmt.Sprintf("r-%d", (i+1)%30)}), metav1.CreateOptions{})
		require.NoError(t, err)
	}
	_, err = s.resource("parametersets").Create(ctx, object("ParameterSet", "cycle", "lab", map[string]any{"revision": "r-0"}), metav1.CreateOptions{})
	require.NoError(t, err)
	token := ""
	for range 10 {
		page, e := s.History(ctx, "cycle", 10, token)
		if e != nil {
			require.ErrorContains(t, e, "revision cycle")
			return
		}
		token = page.GetContinue()
	}
	t.Fatal("cyclic history should have been rejected")
}
