package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStopAt(t *testing.T) {
	claimed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hour := &metav1.Time{Time: claimed.Add(time.Hour)}
	for _, tt := range []struct {
		name       string
		deadline   *metav1.Time
		ads        *int64
		want       time.Time
		wantActive bool
	}{
		{name: "neither"},
		{name: "absolute deadline", deadline: hour, want: hour.Time},
		{name: "activeDeadlineSeconds from the claim", ads: new(int64(60)), want: claimed.Add(time.Minute), wantActive: true},
		{name: "activeDeadlineSeconds earlier", deadline: hour, ads: new(int64(60)), want: claimed.Add(time.Minute), wantActive: true},
		{name: "absolute deadline earlier", deadline: hour, ads: new(int64(7200)), want: hour.Time},
		{name: "a tie is activeDeadlineSeconds'", deadline: hour, ads: new(int64(3600)), want: hour.Time, wantActive: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, active := StopAt(tt.deadline, tt.ads, claimed)
			assert.True(t, tt.want.Equal(got), "got %v, want %v", got, tt.want)
			assert.Equal(t, tt.wantActive, active)
		})
	}
}
