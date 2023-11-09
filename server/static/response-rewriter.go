package static

import (
	"bytes"
	"net/http"
	"strconv"
)

type responseRewriter struct {
	http.ResponseWriter
	old []byte
	new []byte
}

func (w *responseRewriter) Write(a []byte) (int, error) {
	b := bytes.Replace(a, w.old, w.new, 1)
	// status code and headers are printed out when we write data
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	return w.ResponseWriter.Write(b)
}
