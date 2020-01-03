package cron

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/workflow/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

type CronSuite struct {
	fixtures.E2ESuite
}

func (s *CronSuite) TestBasic() {
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/basic.yaml").
		When().
		CreateCronWorkflow().
		Wait(1 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflowStatus) {
			assert.True(t, cronWf.LastScheduledTime.Time.After(time.Now().Add(-1 * time.Minute)))
		})
}

func (s *CronSuite) TestBasicForbid() {
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/basic-forbid.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflowStatus) {
			assert.Equal(t, 1, len(cronWf.Active))
		})
}

func (s *CronSuite) TestBasicAllow() {
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/basic-allow.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflowStatus) {
			assert.Equal(t, 2, len(cronWf.Active))
		})
}

func (s *CronSuite) TestBasicReplace() {
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/basic-replace.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflowStatus) {
			assert.Equal(t, 1, len(cronWf.Active))
		})
}

func (s *CronSuite) TestSuccessfulJobHistoryLimit() {
	var listOptions v1.ListOptions
	cron.WfInformerListOptionsFunc(&listOptions)
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/always-succeed-1.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Equal(t, 1, len(wfList.Items))
			assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1 * time.Minute)))
		})
}

func (s *CronSuite) TestFailedJobHistoryLimit() {
	var listOptions v1.ListOptions
	cron.WfInformerListOptionsFunc(&listOptions)
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/always-fail-1.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Equal(t, 1, len(wfList.Items))
			assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1 * time.Minute)))
		})
}

func (s *CronSuite) TestFailedJobHistoryLimitConcurrent() {
	var listOptions v1.ListOptions
	cron.WfInformerListOptionsFunc(&listOptions)
	s.T().Parallel()
	s.Given().
		CronWorkflow("@testdata/always-fail-2.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Equal(t, 1, len(wfList.Items))
			assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1 * time.Minute)))
		})
}

func TestCronSuite(t *testing.T) {
	suite.Run(t, new(CronSuite))
}
