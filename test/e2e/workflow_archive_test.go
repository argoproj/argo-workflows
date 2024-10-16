//go:build functional

package e2e

import (
	"testing"

	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

func BenchmarkWorkflowArchive(b *testing.B) {
	// Workaround for https://github.com/stretchr/testify/issues/811
	suite := fixtures.E2ESuite{}
	suite.SetT(&testing.T{})
	suite.SetupSuite()
	b.ResetTimer()

	b.Run("ListWorkflows", func(b *testing.B) {
		for range b.N {
			wfs, err := suite.Persistence.WorkflowArchive.ListWorkflows(sutils.ListOptions{
				Limit: 100,
			})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("Found %d workflows", wfs.Len())
		}
	})

	b.Run("CountWorkflows", func(b *testing.B) {
		for range b.N {
			wfCount, err := suite.Persistence.WorkflowArchive.CountWorkflows(sutils.ListOptions{})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("Found %d workflows", wfCount)
		}
	})

	suite.TearDownSuite()
}
