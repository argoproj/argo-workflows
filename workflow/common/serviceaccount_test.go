package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/test/util"
)

// TestGetServiceAccountTokenName verifies service account token retrieved by service account name.
func TestGetServiceAccountTokenName(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	_, err := util.CreateServiceAccountWithToken(clientset, "", "test", "test-token")
	assert.NoError(t, err)
	tokenName, err := GetServiceAccountTokenName(clientset, "", "test")
	assert.NoError(t, err)
	assert.Equal(t, "test-token", tokenName)
}
