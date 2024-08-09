package apiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpts_String(t *testing.T) {
	assert.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false,clientCert=,clientKey=),instanceID=)", Opts{}.String())
	assert.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false,clientCert=,clientKey=),instanceID=my-instanceid)", Opts{InstanceID: "my-instanceid"}.String())
}
