package pool

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StopAt is when a claimed task's worker stops it: the earlier of its
// absolute deadline and its activeDeadlineSeconds after claimedAt, its
// claim's creation on the API server's clock, or zero if it has neither.
// active says it is the activeDeadlineSeconds' end, which wins a tie, as the
// kubelet's message is what the controller reports for a pod that both
// stop. The controller's lease on the task ends a slack after it.
func StopAt(deadline *metav1.Time, activeDeadlineSeconds *int64, claimedAt time.Time) (at time.Time, active bool) {
	if deadline != nil {
		at = deadline.Time
	}
	if activeDeadlineSeconds != nil {
		end := claimedAt.Add(time.Duration(*activeDeadlineSeconds) * time.Second)
		if at.IsZero() || !end.After(at) {
			return end, true
		}
	}
	return at, false
}
