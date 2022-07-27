package controller

import "math"

// getResourceAllowance determines how many resources can be scheduled by this user given the following state variables:
//
// T: total possible slots available for this type of resource (e.g. licenses, compute instances, ..)
// A: total active/pending pods for this type of resource
// R: number of unique users requesting this resource.
// C: current active/pending pods for this particular user
func getResourceAllowance(T, A, R, C int) int {
	// "in theory, we should be able to schedule this many new pods"
	slotsPerUser := math.Ceil(float64(T) / float64(R))
	// "this user is already using this many"
	slotsUsedAlready := float64(C)
	// "there are this many total slots available to run this resource"
	slotsAvailable := float64(T) - float64(A)
	// "we'd normally run slotsPerUser-slotsUsedAlready, but may not be able to"
	return int(math.Min(slotsPerUser-slotsUsedAlready, slotsAvailable))
}
