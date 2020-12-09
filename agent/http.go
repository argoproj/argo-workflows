package agent

import (
	"encoding/json"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
)

func sendErr(w http.ResponseWriter, err error) {
	switch v := err.(type) {
	case *errors.StatusError:
		send(w, int(v.Status().Code), v.Status())
	default:
		send(w, http.StatusInternalServerError, errors.NewInternalError(err))
	}
}

func send(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
