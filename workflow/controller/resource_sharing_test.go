package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetResourceAllowance ensures that getResourceAllowance returns the correct values
func TestGetResourceAllowance(t *testing.T) {
	totalAvailable := 10

	// at first, 1 user tries to use 10 available resources, and has access to all of them
	totalActive := 0
	totalRequesting := 1
	totalActiveUser := 0
	assert.Equal(t, 10, getResourceAllowance(totalAvailable, totalActive, totalRequesting, totalActiveUser))

	// immediately after, a second user tries to submit a job that uses this resource
	// because all 10 slots are taken, this user cannot run any pods atm
	totalActive = 10
	totalRequesting = 1
	totalActiveUser = 0
	assert.Equal(t, 0, getResourceAllowance(totalAvailable, totalActive, totalRequesting, totalActiveUser))

	// then, 7 of the original pods finish, and we're able to allocate 7 resources to the second user.
	totalActive = 3
	totalRequesting = 1
	totalActiveUser = 0
	assert.Equal(t, 7, getResourceAllowance(totalAvailable, totalActive, totalRequesting, totalActiveUser))

	// suppose all pods from the first user's job have now finished
	// we have 0 pods running from first user and 7 from the second. first user
	// first user submits their job again, but cannot schedule all templates immediately
	totalActive = 7
	totalRequesting = 2
	totalActiveUser = 0
	assert.Equal(t, 3, getResourceAllowance(totalAvailable, totalActive, totalRequesting, totalActiveUser))
}
