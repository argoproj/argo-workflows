package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

var cases = []struct {
	UID             types.UID
	ResourceVersion string
}{
	{
		UID:             "1",
		ResourceVersion: "1",
	},
	{
		UID:             "2",
		ResourceVersion: "2",
	},
	{
		UID:             "2",
		ResourceVersion: "3",
	},
	{
		UID:             "1",
		ResourceVersion: "4",
	},
	{
		UID:             "1",
		ResourceVersion: "2",
	},
	{
		UID:             "2",
		ResourceVersion: "5",
	},
}

func TestLatchUpdate(t *testing.T) {
	latch := NewResourceVersionLatch()

	latch.Set("1", "0")
	latch.Set("2", "0")

	for _, cs := range cases {
		latch.Update(cs.UID, cs.ResourceVersion)
	}

	assert.Equal(t, latch.Get("1"), "4")
	assert.Equal(t, latch.Get("2"), "5")
}
