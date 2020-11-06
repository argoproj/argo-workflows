package http1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo/util/flatten"
)

// Facade provides a adapter from GRPC interface, but uses HTTP to send the messages.
// Errors are extracted from message body and returned as GRPC status errors.
type Facade struct {
	baseUrl       string
	authorization string
}

func NewFacade(baseUrl, authorization string) Facade {
	return Facade{baseUrl, authorization}
}

func (h Facade) Get(in, out interface{}, path string) error {
	return h.do(in, out, "GET", path)
}

func (h Facade) Put(in, out interface{}, path string) error {
	return h.do(in, out, "PUT", path)
}

func (h Facade) Post(in, out interface{}, path string) error {
	return h.do(in, out, "POST", path)
}

func (h Facade) Delete(in, out interface{}, path string) error {
	return h.do(in, out, "DELETE", path)
}

func (h Facade) EventStreamReader(in interface{}, path string) (*bufio.Reader, error) {
	method := "GET"
	u, err := h.url(method, path, in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", h.authorization)
	log.Debugf("curl -H 'Accept: text/event-stream' -H 'Authorization: ******' '%v'", u)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	err = errFromResponse(resp)
	if err != nil {
		return nil, err
	}
	return bufio.NewReader(resp.Body), nil
}

func (h Facade) do(in interface{}, out interface{}, method string, path string) error {
	var data []byte
	if method != "GET" {
		var err error
		data, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	u, err := h.url(method, path, in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.authorization)
	log.Debugf("curl -X %s -H 'Authorization: ******' -d '%s' '%v'", method, string(data), u)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = errFromResponse(resp)
	if err != nil {
		return err
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	} else {
		return nil
	}
}

func (h Facade) url(method, path string, in interface{}) (*url.URL, error) {
	query := url.Values{}
	for s, v := range flatten.Flatten(in) {
		x := "{" + s + "}"
		if strings.Contains(path, x) {
			path = strings.Replace(path, x, v, 1)
		} else if method == "GET" {
			query.Set(s, v)
		}
	}
	// remove any that were not provided
	path = regexp.MustCompile("{[^}]*}").ReplaceAllString(path, "")
	return url.Parse(h.baseUrl + path + "?" + query.Encode())
}

func errFromResponse(r *http.Response) error {
	if r.StatusCode == http.StatusOK {
		return nil
	}
	x := &struct {
		Code    codes.Code `json:"code"`
		Message string     `json:"message"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(x); err == nil {
		return status.Errorf(x.Code, x.Message)
	}
	return status.Error(codes.Internal, fmt.Sprintf(": %v", r))
}
