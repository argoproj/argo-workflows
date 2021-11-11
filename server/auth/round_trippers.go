package auth

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

type impersonateRoundTripper struct {
	rt http.RoundTripper
}

// NewImpersonateRoundTripper provides a RoundTripper which will preform a SubjectAccessReview for K8S API calls
func NewImpersonateRoundTripper(rt http.RoundTripper) http.RoundTripper {
	return &impersonateRoundTripper{rt: rt}
}

func (rt *impersonateRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	impersonateClient := GetImpersonateClient(req.Context())
	if impersonateClient == nil {
		return nil, status.Error(codes.Internal, "`impersonate.Client` is missing from HTTP context")
	}

	kubeRequest, err := k8s.ParseRequest(req)
	if err != nil {
		return nil, err
	}

	err = impersonateClient.AccessReview(
		req.Context(),
		kubeRequest.Namespace,
		kubeRequest.Verb,
		kubeRequest.Group,
		kubeRequest.Kind,
		kubeRequest.Name,
		kubeRequest.Subresource,
	)
	if err != nil {
		responseBody := err.Error()
		return &http.Response{
			Status:        "403 Forbidden",
			StatusCode:    403,
			Proto:         req.Proto,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			Body:          ioutil.NopCloser(bytes.NewBufferString(responseBody)),
			ContentLength: int64(len(responseBody)),
			Request:       req,
			Header:        make(http.Header),
		}, nil
	}

	return rt.rt.RoundTrip(req)
}
