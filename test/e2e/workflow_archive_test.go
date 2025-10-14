//go:build functional

package e2e

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"k8s.io/apimachinery/pkg/labels"

	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

func BenchmarkWorkflowArchive(b *testing.B) {
	// Workaround for https://github.com/stretchr/testify/issues/811
	suite := fixtures.E2ESuite{}
	suite.SetT(&testing.T{})
	suite.SetupSuite()
	b.ResetTimer()

	// Uncomment the following line to log queries to stdout
	//db.LC().SetLevel(db.LogLevelDebug)

	ctx := logging.TestContext(b.Context())

	b.Run("ListWorkflows", func(b *testing.B) {
		for range b.N {
			wfs, err := suite.Persistence.WorkflowArchive.ListWorkflows(ctx, sutils.ListOptions{
				Limit: 100,
			})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("Found %d workflows", wfs.Len())
		}
	})

	b.Run("ListWorkflows with label selector", func(b *testing.B) {
		requirements, err := labels.ParseToRequirements("workflows.argoproj.io/phase=Succeeded")
		if err != nil {
			b.Fatal(err)
		}
		for range b.N {
			wfs, err := suite.Persistence.WorkflowArchive.ListWorkflows(ctx, sutils.ListOptions{
				Limit:             100,
				LabelRequirements: requirements,
			})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("Found %d workflows", wfs.Len())
		}
	})

	b.Run("CountWorkflows", func(b *testing.B) {
		for range b.N {
			wfCount, err := suite.Persistence.WorkflowArchive.CountWorkflows(ctx, sutils.ListOptions{})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("Found %d workflows", wfCount)
		}
	})

	suite.TearDownSuite()
}
