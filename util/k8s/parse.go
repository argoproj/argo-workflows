package k8s

import (
	"net/http"
	"strings"
)

func ParseRequest(r *http.Request) (verb string, kind string) {
	i := strings.Index(r.URL.Path, "/v") + 1
	path := strings.Split(r.URL.Path[i:], "/")
	n := len(path)

	verb = map[string]string{
		http.MethodGet:    "List",
		http.MethodPost:   "Create",
		http.MethodDelete: "Delete",
		http.MethodPatch:  "Patch",
		http.MethodPut:    "Update",
	}[r.Method]

	x := n%2 == 0

	if r.URL.Query().Get("watch") != "" {
		verb = "Watch"
	} else if verb == "List" && !x {
		verb = "Get"
	} else if verb == "Delete" && x {
		verb = "DeleteCollection"
	}

	kind = "Unknown"
	switch verb {
	case "List", "Watch", "Create", "DeleteCollection":
		if n > 2 && n%4 == 2 {
			// sub-resource, e.g. pods/exec
			kind = path[n-3] + "/" + path[n-1]
		} else {
			kind = path[n-1]
		}
	case "Get", "Delete", "Patch", "Update":
		if x {
			// sub-resource
			kind = path[n-3] + "/" + path[n-1]
		} else {
			kind = path[n-2]
		}
	}

	return verb, kind
}
