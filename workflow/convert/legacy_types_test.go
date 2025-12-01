package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestLegacySynchronization_ToCurrent(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var ls *LegacySynchronization
		assert.Nil(t, ls.ToCurrent())
	})

	t.Run("neither", func(t *testing.T) {
		ls := &LegacySynchronization{}
		result := ls.ToCurrent()
		assert.Empty(t, result.Semaphores)
		assert.Empty(t, result.Mutexes)
	})

	t.Run("only singular", func(t *testing.T) {
		ls := &LegacySynchronization{
			Semaphore: &wfv1.SemaphoreRef{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "sem"},
					Key:                  "key",
				},
			},
			Mutex: &wfv1.Mutex{Name: "mutex"},
		}
		result := ls.ToCurrent()
		assert.Len(t, result.Semaphores, 1)
		assert.Len(t, result.Mutexes, 1)
	})

	t.Run("only plural", func(t *testing.T) {
		ls := &LegacySynchronization{
			Semaphores: []*wfv1.SemaphoreRef{
				{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "sem1"},
					Key:                  "key",
				}},
				{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "sem2"},
					Key:                  "key",
				}},
			},
			Mutexes: []*wfv1.Mutex{{Name: "mutex1"}, {Name: "mutex2"}},
		}
		result := ls.ToCurrent()
		assert.Len(t, result.Semaphores, 2)
		assert.Len(t, result.Mutexes, 2)
	})

	t.Run("appends singular to plural without aliasing", func(t *testing.T) {
		ls := &LegacySynchronization{
			Semaphore: &wfv1.SemaphoreRef{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "singular-sem"},
					Key:                  "key",
				},
			},
			Mutex: &wfv1.Mutex{Name: "singular-mutex"},
			Semaphores: []*wfv1.SemaphoreRef{
				{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "plural-sem"},
					Key:                  "key",
				}},
			},
			Mutexes: []*wfv1.Mutex{{Name: "plural-mutex"}},
		}
		result := ls.ToCurrent()

		assert.Len(t, result.Semaphores, 2)
		assert.Equal(t, "plural-sem", result.Semaphores[0].ConfigMapKeyRef.Name)
		assert.Equal(t, "singular-sem", result.Semaphores[1].ConfigMapKeyRef.Name)

		assert.Len(t, result.Mutexes, 2)
		assert.Equal(t, "plural-mutex", result.Mutexes[0].Name)
		assert.Equal(t, "singular-mutex", result.Mutexes[1].Name)

		// Verify no aliasing
		assert.Len(t, ls.Semaphores, 1)
		assert.Len(t, ls.Mutexes, 1)
	})
}

func TestLegacyCronWorkflowSpec_ToCurrent(t *testing.T) {
	t.Run("neither", func(t *testing.T) {
		lcs := &LegacyCronWorkflowSpec{}
		result := lcs.ToCurrent()
		assert.Empty(t, result.Schedules)
	})

	t.Run("only singular", func(t *testing.T) {
		lcs := &LegacyCronWorkflowSpec{
			Schedule: "0 * * * *",
		}
		result := lcs.ToCurrent()
		assert.Equal(t, []string{"0 * * * *"}, result.Schedules)
	})

	t.Run("only plural", func(t *testing.T) {
		lcs := &LegacyCronWorkflowSpec{
			Schedules: []string{"0 * * * *", "30 * * * *"},
		}
		result := lcs.ToCurrent()
		assert.Equal(t, []string{"0 * * * *", "30 * * * *"}, result.Schedules)
	})

	t.Run("appends singular to plural without aliasing", func(t *testing.T) {
		lcs := &LegacyCronWorkflowSpec{
			Schedule:  "0 0 * * *",
			Schedules: []string{"0 * * * *", "30 * * * *"},
		}
		result := lcs.ToCurrent()

		assert.Equal(t, []string{"0 * * * *", "30 * * * *", "0 0 * * *"}, result.Schedules)

		// Verify no aliasing
		assert.Len(t, lcs.Schedules, 2)
	})
}
