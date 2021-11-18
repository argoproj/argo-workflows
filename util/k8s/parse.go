package k8s

import (
	"net/http"
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type kubeRequest struct {
	Verb        string
	Namespace   string
	Group       string
	Version     string
	Kind        string
	Name        string
	Subresource string
}

func ParseRequest(r *http.Request) (*kubeRequest, error) {
	// extract ResourceAttributes from URL path
	// https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-uris
	urlPath := r.URL.Path
	re := regexp.MustCompile(
		`^(?:/api|/apis/(?P<GROUP>[^/]+))/(?P<VERSION>[^/]+)(?:/namespaces/(?P<NAMESPACE>[^/]+))?/(?P<KIND>[^/\n]+)(?:/(?P<NAME>[^/\n]+))?(?:/(?P<SUBRESOURCE>[^/\n]+))?$`,
	)
	matches := re.FindStringSubmatch(urlPath)
	if matches == nil {
		return nil, status.Errorf(codes.Internal, "invalid kubernetes request path: %s", urlPath)
	}
	namespace := ""
	if re.SubexpIndex("NAMESPACE") != -1 {
		namespace = matches[re.SubexpIndex("NAMESPACE")]
	}
	resourceGroup := ""
	if re.SubexpIndex("GROUP") != -1 {
		resourceGroup = matches[re.SubexpIndex("GROUP")]
	}
	resourceVersion := ""
	if re.SubexpIndex("VERSION") != -1 {
		resourceVersion = matches[re.SubexpIndex("VERSION")]
	}
	resourceKind := ""
	if re.SubexpIndex("KIND") != -1 {
		resourceKind = matches[re.SubexpIndex("KIND")]
	}
	resourceName := ""
	if re.SubexpIndex("NAME") != -1 {
		resourceName = matches[re.SubexpIndex("NAME")]
	}
	subresource := ""
	if re.SubexpIndex("SUBRESOURCE") != -1 {
		subresource = matches[re.SubexpIndex("SUBRESOURCE")]
	}

	// extract flags from URL query
	urlQuery := r.URL.Query()
	isWatch := urlQuery.Get("watch") != ""

	// calculate the resource verb
	// https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb
	urlMethod := r.Method
	verb := ""
	switch urlMethod {
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
		return nil, status.Errorf(codes.Internal, "could not calculate kubernetes resource verb for %s on %s", urlMethod, urlPath)
	}

	return &kubeRequest{
		Verb:        verb,
		Namespace:   namespace,
		Group:       resourceGroup,
		Version:     resourceVersion,
		Kind:        resourceKind,
		Name:        resourceName,
		Subresource: subresource,
	}, nil
}
