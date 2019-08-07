package common

import (
	"testing"

	"github.com/argoproj/argo/test/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetServiceAccountTokenByAccountName verifies service account token retrieved by service account name.
func TestGetServiceAccountTokenByAccountName(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	_, err := util.CreateServiceAccountWithToken(clientset, "", "test", "test-token")
	assert.NoError(t, err)
	token, err := GetServiceAccountTokenByAccountName(clientset, "", "test")
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test-token", token.Name)
}

// TestGetReferencedServiceAccountToken verifies service account token retrieved by service account name.
func TestGetReferencedServiceAccountToken(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	sa, err := util.CreateServiceAccountWithToken(clientset, "", "test", "test-token")
	assert.NoError(t, err)
	token, err := GetReferencedServiceAccountToken(clientset, sa)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test-token", token.Name)
}

// TestGetReferencedServiceAccountToken verifies service account token retrieved by service account name.
func TestGetServiceAccountTokens(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	sa, err := util.CreateServiceAccountWithToken(clientset, "", "test", "test-token")
	assert.NoError(t, err)
	tokens, err := GetServiceAccountTokens(clientset, sa)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tokens))
	assert.Equal(t, "test-token", tokens[0].Name)
}
