package http1

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFacade_do(t *testing.T) {
	f := Facade{baseURL: "http://my-url"}
	u, err := f.url("GET", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())

	u, err = f.url("DELETE", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())
}

func TestFacade_proxyFunc(t *testing.T) {
	proxyFunc := func(_ *http.Request) (*url.URL, error) {
		return nil, nil
	}
	tests := []struct {
		name  string
		proxy func(*http.Request) (*url.URL, error)
		want  func(*http.Request) (*url.URL, error)
	}{
		{
			name:  "use proxy settings from environment variables",
			proxy: nil,
			want:  http.ProxyFromEnvironment,
		},
		{
			name:  "use specific proxy",
			proxy: proxyFunc,
			want:  proxyFunc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Facade{proxy: tt.want}
			got := f.proxyFunc()
			if reflect.ValueOf(got).Pointer() != reflect.ValueOf(tt.want).Pointer() {
				t.Errorf("Facade.proxyURL() = %p, want %p", got, tt.want)
			}
		})
	}
}
