package util

import "testing"

func TestCleanMetadata(t *testing.T) {
	CleanMetadata(nil)
}

func TestRemoveManagedFields(t *testing.T) {
	RemoveManagedFields(nil)
}

func TestRemoveSelfLink(t *testing.T) {
	RemoveSelfLink(nil)
}
