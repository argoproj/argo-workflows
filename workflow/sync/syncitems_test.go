package sync

import (
	"fmt"
	"reflect"
	"testing"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		items []*syncItem
	}{
		{
			name: "single",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
		{
			name: "two",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "beta",
					},
				},
			},
		},
		{
			name: "different namespace mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name:      "alpha",
						Namespace: "foo",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name:      "alpha",
						Namespace: "bar",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDuplicates(test.items)
			assert.NoError(t, err)
		})
	}
}

func TestExpectDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		items []*syncItem
	}{
		{
			name: "simple duplicate mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
		{
			name: "simple duplicate semaphore",
			items: []*syncItem{
				{
					semaphore: &v1alpha1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "foo",
							},
							Key: "alpha",
						},
					},
				},
				{
					semaphore: &v1alpha1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "foo",
							},
							Key: "alpha",
						},
					},
				},
			},
		},
		{
			name: "another duplicate mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "beta",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDuplicates(test.items)
			assert.Error(t, err)
		})
	}
}

// checkDuplicatesPairwise is the pairwise DeepEqual check checkDuplicates
// must agree with.
func checkDuplicatesPairwise(items []*syncItem) bool {
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if reflect.DeepEqual(*items[i], *items[j]) {
				return true
			}
		}
	}
	return false
}

// syncItemVariants returns items that differ from each other in at most one
// field, including nil against empty pointers.
func syncItemVariants() []*syncItem {
	t, f := true, false
	dbs := []*v1alpha1.SyncDatabaseRef{nil, {}, {Key: "a"}}
	items := []*syncItem{{}}
	for _, ns := range []string{"", "ns"} {
		for _, db := range dbs {
			items = append(items, &syncItem{semaphore: &v1alpha1.SemaphoreRef{Namespace: ns, Database: db}})
		}
		for _, name := range []string{"", "a"} {
			items = append(items,
				&syncItem{mutex: &v1alpha1.Mutex{Name: name, Namespace: ns}},
				&syncItem{mutex: &v1alpha1.Mutex{Name: name, Namespace: ns, Database: true}},
			)
			for _, optional := range []*bool{nil, &t, &f} {
				for _, db := range dbs {
					items = append(items, &syncItem{semaphore: &v1alpha1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{Name: name},
							Key:                  name,
							Optional:             optional,
						},
						Namespace: ns,
						Database:  db,
					}})
				}
			}
		}
	}
	return items
}

func TestCheckDuplicatesMatchesPairwise(t *testing.T) {
	variants := syncItemVariants()
	require.False(t, checkDuplicatesPairwise(variants), "variants must be distinct")
	require.NoError(t, checkDuplicates(variants))
	for i := range variants {
		for j := range variants {
			// a fresh copy of j, so equal items are not the same pointer
			copied := *variants[j]
			items := []*syncItem{variants[i], &copied}
			assert.Equal(t, checkDuplicatesPairwise(items), checkDuplicates(items) != nil, "items %d and %d", i, j)
		}
	}
}

func manyMutexes(n int) []*syncItem {
	items := make([]*syncItem, n)
	for i := range items {
		items[i] = &syncItem{mutex: &v1alpha1.Mutex{Name: fmt.Sprintf("mutex-%d", i)}}
	}
	return items
}

func TestCheckDuplicatesManyItems(t *testing.T) {
	items := manyMutexes(20000)
	require.NoError(t, checkDuplicates(items))
	items = append(items, &syncItem{mutex: &v1alpha1.Mutex{Name: "mutex-0"}})
	require.Error(t, checkDuplicates(items))
}

func BenchmarkCheckDuplicates(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		items := manyMutexes(n)
		b.Run(fmt.Sprintf("items=%d", n), func(b *testing.B) {
			for b.Loop() {
				if err := checkDuplicates(items); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
