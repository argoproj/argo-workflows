package secret_test

import (
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/restcl"
	"flag"
	"gopkg.in/check.v1"
	"testing"
	"time"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

const (
	axdburl        = "http://localhost:8080/v1"
	kafkaurl       = "localhost:9092"
	axopsurl       = "http://localhost:8085/v1"
	axopsexurl     = "http://localhost:8086/v1"
	gatewayurl     = "http://localhost:9090/v1"
	workflowadcurl = "http://localhost:9090/v1"
	axmonurl       = "http://localhost:9090/v1"
	axnotifierurl  = "http://localhost:9090/v1"
	fixmgrurl      = "http://localhost:9091/v1"
	schedulerurl   = "http://localhost:9090/v1"
	verbose        = true
)

// Use a client explicitly. Later replace the client with one that uses TLS
var axdbClient = axdbcl.NewAXDBClientWithTimeout(axdburl, time.Second*60)

//var axopsClient = restcl.NewRestClient(axopsurl)
var axopsExternalClient = restcl.NewRestClientWithTimeout(axopsexurl, 60*time.Second)

func (s *S) SetUpSuite(c *check.C) {
	flag.Parse()
	// We test against our REST API. So we need to start our main program here.
	core.InitLoggers()

}
