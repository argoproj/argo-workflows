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

	if r.URL.Query().Get("watch") != "" {
		verb = "Watch"
	} else if verb == "List" && n%2 == 1 {
		verb = "Get"
	} else if verb == "Delete" && n%2 == 0 {
		verb = "DeleteCollection"
	}

	kind = "Unknown"
	switch verb {
	case "List", "Watch", "Create", "DeleteCollection":
		kind = path[n-1]
	case "Get", "Delete", "Patch", "Update":
		kind = path[n-2]
	}

	return verb, kind
}
