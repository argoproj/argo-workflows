package webhook

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type testHTTPHandler struct{}

func (t testHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func TestInterceptor(t *testing.T) {
	// we ignore these
	t.Run("WrongMethod", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "GET", "/api/v1/events/", nil)
		assert.Empty(t, r.Header["Authorization"])
	})
	t.Run("ExistingAuthorization", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{"Authorization": "existing"})
		assert.Equal(t, []string{"existing"}, r.Header["Authorization"])
	})
	t.Run("WrongPathPrefix", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/xxx/", nil)
		assert.Empty(t, r.Header["Authorization"])
	})
	t.Run("NoNamespace", func(t *testing.T) {
		r, w := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events//my-d", nil)
		assert.Empty(t, r.Header["Authorization"])
		// we check the status code here - because we get a 403
		assert.Equal(t, 403, w.Code)
		assert.JSONEq(t, `{"message": "failed to process webhook request"}`, w.Body.String())
	})
	t.Run("NoDiscriminator", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/", nil)
		assert.Empty(t, r.Header["Authorization"])
	})
	// we accept these
	t.Run("Bitbucket", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Event-Key": "repo:push",
			"X-Hook-UUID": "sh!",
		})
		assert.Equal(t, []string{"Bearer my-bitbucket-token"}, r.Header["Authorization"])
	})
	t.Run("Bitbucketserver", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Event-Key":     "pr:modified",
			"X-Hub-Signature": "0000000926ceeb8dcd67d5979fd7d726e3905af6d220f7fd6b2d8cce946906f7cf35963",
		})
		assert.Equal(t, []string{"Bearer my-bitbucketserver-token"}, r.Header["Authorization"])
	})
	t.Run("Github", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": "sha256=926ceeb8dcd67d5979fd7d726e3905af6d220f7fd6b2d8cce946906f7cf35963",
		})
		assert.Equal(t, []string{"Bearer my-github-token"}, r.Header["Authorization"])
	})
	t.Run("Gitlab", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Gitlab-Event": "Push Hook",
			"X-Gitlab-Token": "sh!",
		})
		assert.Equal(t, []string{"Bearer my-gitlab-token"}, r.Header["Authorization"])
	})
	// x-hub: default values
	t.Run("X-Hub-default", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Hub-Signature-256": "sha256=43922693c6a1c6b72cb3e006184e093b5eb01cb9df3569fa39ab75eff8495e5a",
		})
		assert.Equal(t, []string{"Bearer my-x-hub-default-token"}, r.Header["Authorization"])
	})
	// x-hub: default values, unprefixed header value
	t.Run("X-Hub-default-no-value-prefix", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Hub-Signature-256": "43922693c6a1c6b72cb3e006184e093b5eb01cb9df3569fa39ab75eff8495e5a",
		})
		assert.Equal(t, []string{"Bearer my-x-hub-default-token"}, r.Header["Authorization"])
	})
	// x-hub: custom header name
	t.Run("X-Hub-custom-header", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Custom-Header": "sha256=43922693c6a1c6b72cb3e006184e093b5eb01cb9df3569fa39ab75eff8495e5a",
		})
		assert.Equal(t, []string{"Bearer my-x-hub-custom-header-token"}, r.Header["Authorization"])
	})
	// x-hub: custom hash algo
	t.Run("X-Hub-custom-hash", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Hub-Signature": "sha1=956497d238fdac40c567570cb1a614e7d399875a",
		})
		assert.Equal(t, []string{"Bearer my-x-hub-custom-hash-token"}, r.Header["Authorization"])
	})
	// x-hub: base64 signature
	t.Run("X-Hub-base64-signature", func(t *testing.T) {
		r, _ := intercept(logging.TestContext(t.Context()), "POST", "/api/v1/events/my-ns/my-d", map[string]string{
			"X-Hub-Signature-256": "Q5Imk8ahxrcss+AGGE4JO16wHLnfNWn6Oat17/hJXlo=",
		})
		assert.Equal(t, []string{"Bearer my-x-hub-base64-signature-token"}, r.Header["Authorization"])
	})
}

func intercept(ctx context.Context, method string, target string, headers map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	// set-up
	k := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "argo-workflows-webhook-clients", Namespace: "my-ns"},
			Data: map[string][]byte{
				"bitbucket":              []byte("type: bitbucket\nsecret: sh!"),
				"bitbucketserver":        []byte("type: bitbucketserver\nsecret: sh!"),
				"github":                 []byte("type: github\nsecret: sh!"),
				"gitlab":                 []byte("type: gitlab\nsecret: sh!"),
				"x-hub-default":          []byte("type: x-hub\nsecret: sh!sh!"),
				"x-hub-custom-header":    []byte("type: x-hub\nsecret: sh!sh!\nx-hub-header-name: X-Custom-Header"),
				"x-hub-custom-hash":      []byte("type: x-hub\nsecret: sh!sh!\nx-hub-header-name: X-Hub-Signature\nx-hub-hash: sha1"),
				"x-hub-base64-signature": []byte("type: x-hub\nsecret: sh!sh!\nx-hub-encoding: base64"),
			},
		},
		// bitbucket
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "bitbucket", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "bitbucket-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "bitbucket-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-bitbucket-token")},
		},
		// bitbucketserver
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "bitbucketserver", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "bitbucketserver-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "bitbucketserver-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-bitbucketserver-token")},
		},
		// github
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "github-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-github-token")},
		},
		// gitlab
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "gitlab", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "gitlab-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "gitlab-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-gitlab-token")},
		},
		// x-hub default
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-default", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "x-hub-default-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-default-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-x-hub-default-token")},
		},
		// x-hub custom header
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-custom-header", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "x-hub-custom-header-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-custom-header-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-x-hub-custom-header-token")},
		},
		// x-hub custom hash
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-custom-hash", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "x-hub-custom-hash-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-custom-hash-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-x-hub-custom-hash-token")},
		},
		// x-hub base64 signature
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-base64-signature", Namespace: "my-ns"},
			Secrets:    []corev1.ObjectReference{{Name: "x-hub-base64-signature-token"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "x-hub-base64-signature-token", Namespace: "my-ns"},
			Data:       map[string][]byte{"token": []byte("my-x-hub-base64-signature-token")},
		},
	)
	i := NewInterceptor(logging.RequireLoggerFromContext(ctx)).Interceptor(k)
	w := httptest.NewRecorder()
	b := &bytes.Buffer{}
	b.WriteString("{}")
	r := httptest.NewRequest(method, target, b)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	h := &testHTTPHandler{}
	// act
	i(w, r, h)
	return r, w
}
