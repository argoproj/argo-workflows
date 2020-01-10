package cron

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/pkg/humanize"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"testing"
	"time"
)

type CronSuite struct {
	fixtures.E2ESuite
}

func (s *CronSuite) TestBasic() {
	s.Given().
		CronWorkflow("@testdata/basic.yaml").
		When().
		CreateCronWorkflow().
		Wait(1 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.True(t, cronWf.Status.LastScheduledTime.Time.After(time.Now().Add(-1*time.Minute)))
		})
}

func (s *CronSuite) TestSuspend() {
	s.Given().
		CronWorkflow("@testdata/basic.yaml").
		When().
		CreateCronWorkflow().
		Then().
		RunCli([]string{"cron", "suspend", "test-cron-wf-basic"}, func(t *testing.T, output string) {
			assert.Equal(t, "CronWorkflow 'test-cron-wf-basic' suspended", output)
		}).ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
		assert.True(t, cronWf.Spec.Suspend)
	})
}

func (s *CronSuite) TestResume() {
	s.Given().
		CronWorkflow("@testdata/basic.yaml").
		When().
		CreateCronWorkflow().
		Then().
		RunCli([]string{"cron", "resume", "test-cron-wf-basic"}, func(t *testing.T, output string) {
			assert.Equal(t, "CronWorkflow 'test-cron-wf-basic' resumed", output)
		}).ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
		assert.False(t, cronWf.Spec.Suspend)
	})
}

func (s *CronSuite) TestBasicForbid() {
	s.Given().
		CronWorkflow("@testdata/basic-forbid.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Equal(t, 1, len(cronWf.Status.Active))
			assert.True(t, cronWf.Status.LastScheduledTime.Time.Before(time.Now().Add(-1*time.Minute)))
		})
}

func (s *CronSuite) TestBasicAllow() {
	s.Given().
		CronWorkflow("@testdata/basic-allow.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Equal(t, 2, len(cronWf.Status.Active))
		})
}

func (s *CronSuite) TestBasicReplace() {
	s.Given().
		CronWorkflow("@testdata/basic-replace.yaml").
		When().
		CreateCronWorkflow().
		Wait(2 * time.Minute).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Equal(t, 1, len(cronWf.Status.Active))
			assert.True(t, cronWf.Status.LastScheduledTime.Time.After(time.Now().Add(-1*time.Minute)))
		})
}

func (s *CronSuite) TestSuccessfulJobHistoryLimit() {
	var listOptions v1.ListOptions
	wfInformerListOptionsFunc(&listOptions, "test-cron-wf-succeed-1")
	s.Given().
		CronWorkflow("@testdata/always-succeed-1.yaml").
		When().
		CreateCronWorkflow().
		Wait(2*time.Minute).
		Then().
		ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Equal(t, 1, len(wfList.Items))
			assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1*time.Minute)))
		})
}

func (s *CronSuite) TestFailedJobHistoryLimit() {
	var listOptions v1.ListOptions
	wfInformerListOptionsFunc(&listOptions, "test-cron-wf-fail-1")
	s.Given().
		CronWorkflow("@testdata/always-fail-1.yaml").
		When().
		CreateCronWorkflow().
		Wait(2*time.Minute).
		Then().
		ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Equal(t, 1, len(wfList.Items))
			assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1*time.Minute)))
		})
}

func wfInformerListOptionsFunc(options *v1.ListOptions, cronWfName string) {
	options.FieldSelector = fields.Everything().String()
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelCronWorkflow, selection.Equals, []string{cronWfName})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*isCronWorkflowChildReq)
	options.LabelSelector = labelSelector.String()
}

func TestCronSuite(t *testing.T) {
	// To ensure consistency, always start at the next 30 second mark
	_, _, sec := time.Now().Clock()
	var toWait time.Duration
	if sec <= 30 {
		toWait = time.Duration(30-sec) * time.Second
	} else {
		toWait = time.Duration(90-sec) * time.Second
	}
	logrus.Infof("Waiting %s to start", humanize.Duration(toWait))
	time.Sleep(toWait)
	suite.Run(t, new(CronSuite))
}
