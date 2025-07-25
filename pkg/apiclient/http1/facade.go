package http1

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v3/util/flatten"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// Facade provides a adapter from GRPC interface, but uses HTTP to send the messages.
// Errors are extracted from message body and returned as GRPC status errors.
type Facade struct {
	baseURL            string
	authorization      string
	insecureSkipVerify bool
	headers            []string
	httpClient         *http.Client
}

func NewFacade(baseURL, authorization string, insecureSkipVerify bool, headers []string, httpClient *http.Client) Facade {
	return Facade{baseURL, authorization, insecureSkipVerify, headers, httpClient}
}

func (h Facade) Get(ctx context.Context, in, out interface{}, path string) error {
	return h.do(ctx, in, out, "GET", path)
}

func (h Facade) Put(ctx context.Context, in, out interface{}, path string) error {
	return h.do(ctx, in, out, "PUT", path)
}

func (h Facade) Post(ctx context.Context, in, out interface{}, path string) error {
	return h.do(ctx, in, out, "POST", path)
}

func (h Facade) Delete(ctx context.Context, in, out interface{}, path string) error {
	return h.do(ctx, in, out, "DELETE", path)
}

func (h Facade) EventStreamReader(ctx context.Context, in interface{}, path string) (*bufio.Reader, error) {
	log := logging.RequireLoggerFromContext(ctx)
	method := "GET"
	u, err := h.url(method, path, in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	headers, err := parseHeaders(h.headers)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", h.authorization)
	log.Debugf(ctx, "curl -H 'Accept: text/event-stream' -H 'Authorization: ******' '%v'", u)
	client := h.httpClient
	if h.httpClient == nil {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: h.insecureSkipVerify,
				},
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	err = errFromResponse(resp)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	return bufio.NewReader(resp.Body), nil
}

func (h Facade) do(ctx context.Context, in interface{}, out interface{}, method string, path string) error {
	log := logging.RequireLoggerFromContext(ctx)
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
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	headers, err := parseHeaders(h.headers)
	if err != nil {
		return err
	}
	req.Header = headers
	req.Header.Set("Authorization", h.authorization)
	log.Debugf(ctx, "curl -X %s -H 'Authorization: ******' -d '%s' '%v'", method, string(data), u)
	client := h.httpClient
	if h.httpClient == nil {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: h.insecureSkipVerify,
				},
				DisableKeepAlives: true,
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
	return url.Parse(h.baseURL + path + "?" + query.Encode())
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
		return status.Error(x.Code, x.Message)
	}
	return status.Error(codes.Internal, fmt.Sprintf(": %v", r))
}

func parseHeaders(headerStrings []string) (http.Header, error) {
	headers := http.Header{}
	for _, kv := range headerStrings {
		items := strings.Split(kv, ":")
		if len(items)%2 == 1 {
			return nil, fmt.Errorf("additional headers must be colon(:)-separated: %s", kv)
		}
		headers.Add(items[0], items[1])
	}
	return headers, nil
}
