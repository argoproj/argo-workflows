package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// Main presumes the supervisor dead after supervisorHeartbeatTimeout without
// a heartbeat; that must stay several heartbeats long, or a live but slow
// supervisor is killed.
func TestSupervisorHeartbeatTimeoutSpansSeveralHeartbeats(t *testing.T) {
	assert.GreaterOrEqual(t, supervisorHeartbeatTimeout, 3*wfexecutor.SupervisorHeartbeatInterval)
}
