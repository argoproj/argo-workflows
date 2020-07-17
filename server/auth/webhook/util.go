package webhook

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func peekBody(r *http.Request) []byte {
	buf, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return buf
}
