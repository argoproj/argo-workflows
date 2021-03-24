package apiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpts_String(t *testing.T) {
	assert.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false),instanceID=)", Opts{}.String())
	assert.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false),instanceID=my-instanceid)", Opts{InstanceID: "my-instanceid"}.String())
}
