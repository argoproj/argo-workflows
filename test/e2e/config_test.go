package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type ConfigSuite struct {
	fixtures.E2ESuite
}

func (s *ConfigSuite) TestConfigMapChange() {
	s.T().SkipNow()
	configMaps := s.KubeClient.CoreV1().ConfigMaps(fixtures.Namespace)
	cm, err := configMaps.Get("workflow-controller-configmap", metav1.GetOptions{})
	assert.NoError(s.T(), err)
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow()

	cm.Data["test"] = fmt.Sprintf("%v", time.Now().String())
	_, err = configMaps.Update(cm)
	assert.NoError(s.T(), err)
	s.Given().
		WorkflowName("test").
		When().
		WaitForWorkflow(30*time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
	})
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
