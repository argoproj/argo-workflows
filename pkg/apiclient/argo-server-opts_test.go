package apiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgoServerOpts_String(t *testing.T) {
	assert.Equal(t, "(url=my-url,path=/my-path,secure=false,insecureSkipVerify=false,http=false,clientCert=,clientKey=,caCert=)", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.String())
	assert.Equal(t, "(url=,path=,secure=true,insecureSkipVerify=false,http=false,clientCert=,clientKey=,caCert=)", ArgoServerOpts{Secure: true}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=true,http=false,clientCert=,clientKey=,caCert=)", ArgoServerOpts{InsecureSkipVerify: true}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=true,clientCert=,clientKey=,caCert=)", ArgoServerOpts{HTTP1: true}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=false,clientCert=cert.pem,clientKey=,caCert=)", ArgoServerOpts{ClientCert: "cert.pem"}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=false,clientCert=,clientKey=key.pem,caCert=)", ArgoServerOpts{ClientKey: "key.pem"}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=false,clientCert=,clientKey=,caCert=ca.pem)", ArgoServerOpts{CACert: "ca.pem"}.String())
}

func TestArgoServerOpts_GetURL(t *testing.T) {
	assert.Equal(t, "http://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.GetURL())
	assert.Equal(t, "https://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path", Secure: true}.GetURL())
}
