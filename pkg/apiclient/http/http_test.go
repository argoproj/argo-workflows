package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFacade_do(t *testing.T) {
	f := Facade{baseUrl: "http://my-url"}
	u, err := f.url("GET", "/{name}", &metav1.ObjectMeta{Name: "my-name", Labels: map[string]string{"foo": "1"}})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://my-url/my-name?labels.foo=1", u.String())
	}
}
