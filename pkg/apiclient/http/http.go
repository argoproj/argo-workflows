package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo/util/flatten"
)

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
	req.Close = true
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	err = errFromResponse(resp.StatusCode)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(resp.Body)
	return reader, nil
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = errFromResponse(resp.StatusCode)
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
	f := flatten.Flatten(in)
	for s, v := range f {
		x := "{" + s + "}"
		pathParam := strings.Contains(path, x)
		if pathParam {
			path = strings.Replace(path, x, v, 1)
		} else if method == "GET" {
			query.Set(s, v)
		}
	}
	return url.Parse(h.baseUrl + "/" + path + "?" + query.Encode())
}

func errFromResponse(statusCode int) error {
	if statusCode == http.StatusOK {
		return nil
	}
	code, ok := map[int]codes.Code{
		http.StatusNotFound:            codes.NotFound,
		http.StatusConflict:            codes.AlreadyExists,
		http.StatusBadRequest:          codes.InvalidArgument,
		http.StatusMethodNotAllowed:    codes.Unimplemented,
		http.StatusServiceUnavailable:  codes.Unavailable,
		http.StatusPreconditionFailed:  codes.FailedPrecondition,
		http.StatusUnauthorized:        codes.Unauthenticated,
		http.StatusForbidden:           codes.PermissionDenied,
		http.StatusRequestTimeout:      codes.DeadlineExceeded,
		http.StatusGatewayTimeout:      codes.DeadlineExceeded,
		http.StatusInternalServerError: codes.Internal,
	}[statusCode]
	if ok {
		return status.Error(code, "")
	}
	return status.Error(codes.Internal, fmt.Sprintf("unknown error: %v", statusCode))
}
