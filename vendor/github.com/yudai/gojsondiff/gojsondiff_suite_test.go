package gojsondiff_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGojsondiff(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gojsondiff Suite")
}
