package lock_test

import (
	"testing"

	"gopkg.in/check.v1"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})
