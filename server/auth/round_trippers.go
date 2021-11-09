package auth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	urlPath := req.URL.Path
	urlQuery := req.URL.Query()

	//log.WithFields(
	//	log.Fields{"Method": req.Method, "Path": urlPath, "Query": req.URL.RawQuery},
	//).Debug("ImpersonateRoundTripper")

	// extract ResourceAttributes from URL path
	re := regexp.MustCompile(
		`^(?:/api|/apis/(?P<GROUP>[^/]+))/(?P<VERSION>[^/]+)(?:/namespaces/(?P<NAMESPACE>[^/]+))?/(?P<RESOURCETYPE>[^/\n]+)(?:/(?P<NAME>[^/\n]+))?(?:/(?P<SUBRESOURCE>[^/\n]+))?$`,
	)
	matches := re.FindStringSubmatch(urlPath)
	if matches == nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Invalid Kubernetes Resource URI path: %s", urlPath))
	}
	namespace := ""
	if re.SubexpIndex("NAMESPACE") != -1 {
		namespace = matches[re.SubexpIndex("NAMESPACE")]
	}
	resourceGroup := ""
	if re.SubexpIndex("GROUP") != -1 {
		resourceGroup = matches[re.SubexpIndex("GROUP")]
	}
	resourceType := ""
	if re.SubexpIndex("RESOURCETYPE") != -1 {
		resourceType = matches[re.SubexpIndex("RESOURCETYPE")]
	}
	resourceName := ""
	if re.SubexpIndex("NAME") != -1 {
		resourceName = matches[re.SubexpIndex("NAME")]
	}
	subresource := ""
	if re.SubexpIndex("NAME") != -1 {
		subresource = matches[re.SubexpIndex("NAME")]
	}

	// extract flags from URL query
	isWatch := false
	if urlQuery.Get("watch") == "1" {
		isWatch = true
	}

	// calculate the resource verb
	verb := ""
	switch req.Method {
	case "", "GET":
		if isWatch {
			verb = "watch"
		} else {
			if resourceName != "" {
				verb = "get"
			} else {
				verb = "list"
			}
		}
	case "POST":
		verb = "create"
	case "PUT":
		verb = "update"
	case "PATCH":
		verb = "patch"
	case "DELETE":
		if resourceName != "" {
			verb = "delete"
		} else {
			verb = "deletecollection"
		}
	default:
		return nil, status.Error(codes.Internal, fmt.Sprintf("Could not calcluate kubernetes resource verb for %s on %s", req.Method, req.URL))
	}

	err := impersonateClient.AccessReview(
		req.Context(),
		namespace,
		verb,
		resourceGroup,
		resourceType,
		resourceName,
		subresource,
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
