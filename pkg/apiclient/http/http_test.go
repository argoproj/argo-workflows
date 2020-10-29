package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_errFromResponse(t *testing.T) {
	for _, tt := range []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"200", 200, false},
		{"400", 400, true},
		{"500", 500, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := errFromResponse(tt.statusCode); (err != nil) != tt.wantErr {
				t.Errorf("errFromResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFacade_do(t *testing.T) {
	f := Facade{baseUrl: "http://my-url"}
	u, err := f.url("GET", "/{name}", &metav1.ObjectMeta{Name: "my-name", Labels: map[string]string{"foo": "1"}})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://my-url/my-name?labels.foo=1", u.String())
	}
}
