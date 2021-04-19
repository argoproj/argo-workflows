package controller

import "testing"

var http = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: http
  templates:
    - name: http
      http:
        url: https://www.google.com/

`
func TestHTTPTemplate(t *testing.T){
	wf := unmarshalWF(http)
	cancel, controller := newController()
	defer cancel()


}
