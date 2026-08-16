package serviceaccount

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

const sub = "1234567890"
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

const sub2 = "system:serviceaccount:argo:jenkins"
const saName2 = "jenkins"
const saNs2 = "argo"
const iss2 = "https://kubernetes.default.svc.cluster.local"
const token2 = "eyJhbGciOiJSUzI1NiIsImtpZCI6Ijc5dVprMUl0VHZkTXFpLWc4dVQwVkV1Y05UZ21XXzJvZjNuZi1iZkpfVW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiLCJrM3MiXSwiZXhwIjoxNzIwNzYzNjgyLCJpYXQiOjE3MjA3NjAwODIsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJhcmdvIiwic2VydmljZWFjY291bnQiOnsibmFtZSI6ImplbmtpbnMiLCJ1aWQiOiIyMGNjYzBjNS00NmNjLTQ3MjctYmUxMi1iZWY0ZTQ0ZTkxMjYifX0sIm5iZiI6MTcyMDc2MDA4Miwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OmFyZ286amVua2lucyJ9.k0SKX4yKga70R12ScI-mSpEWbwdRtlphxXB-drHNyxPyzz2fpur29ZzdcjE8-TlZE2fQrrV-MaqMBWqaD0TWsld0ILBYRYFOcIwuUOX8s611vqmpDejTT6oAroBEd4WSduP9WoxMsiN82EdDvDJeMpNpq8i-Nz8nYgfWe2VkqV_oYCenWKe9JC3QL1TOxdkqer2qgflGnIpTzSVf7y47vmdlqsS9GPbdXyCg4MdcyDnLgY2VoVQnymD22uTG3Ugp3dO3zhaS-puHMPDOa-_EbAn2MsqETrA8H8iKal-z4sx-R6MxN8fdBrrIkWZLZZD4i_EhBS5paFTNcXXq0-T90w"

func TestClaimSetFor(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{})
		require.NoError(t, err)
		assert.Nil(t, claims)
	})
	t.Run("Basic", func(t *testing.T) {
		const username = "my-username"
		claims, err := ClaimSetFor(&rest.Config{Username: username})
		require.NoError(t, err)
		assert.Empty(t, claims.Issuer)
		assert.Equal(t, username, claims.Subject)
	})
	t.Run("BadBearerToken", func(t *testing.T) {
		_, err := ClaimSetFor(&rest.Config{BearerToken: "bad"})
		require.Error(t, err)
	})
	t.Run("BearerToken", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{BearerToken: token})
		require.NoError(t, err)
		assert.Empty(t, claims.Issuer)
		assert.Equal(t, sub, claims.Subject)
	})

	// set-up test
	tmp, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	err = os.WriteFile(tmp.Name(), []byte(token), 0o600)
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()

	t.Run("BearerTokenFile", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{BearerTokenFile: tmp.Name()})
		require.NoError(t, err)
		assert.Empty(t, claims.Issuer)
		assert.Equal(t, sub, claims.Subject)
	})

	t.Run("BearerToken with SA details", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{BearerToken: token2})
		require.NoError(t, err)
		assert.Equal(t, iss2, claims.Issuer)
		assert.Equal(t, sub2, claims.Subject)
		assert.Equal(t, saName2, claims.ServiceAccountName)
		assert.Equal(t, saNs2, claims.ServiceAccountNamespace)
	})
}
