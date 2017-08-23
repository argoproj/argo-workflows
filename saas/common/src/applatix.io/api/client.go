package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"applatix.io/axerror"
)

type ArgoClient struct {
	Config ClusterConfig
	Trace  bool

	client    http.Client
	cookieJar http.CookieJar
	baseURL   string
}

// NewArgoClient instantiates a new client from a specified config, or the default search order
func NewArgoClient(configs ...ClusterConfig) ArgoClient {
	var config ClusterConfig
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = NewClusterConfig()
	}
	config.URL = strings.TrimRight(config.URL, "/")
	cookieJar, _ := cookiejar.New(nil)
	argoClient := ArgoClient{
		Config:    config,
		cookieJar: cookieJar,
	}
	argoClient.client = argoClient.newHTTPClient(DefaultHTTPClientTimeout)
	return argoClient
}

func (c *ArgoClient) newHTTPClient(timeout time.Duration) http.Client {
	httpClient := http.Client{
		Jar:     c.cookieJar,
		Timeout: timeout,
	}
	if c.Config.Insecure != nil && *c.Config.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return httpClient
}

func (c *ArgoClient) get(path string, target interface{}) *axerror.AXError {
	return c.doRequest("GET", path, nil, target)
}

func (c *ArgoClient) post(path string, body interface{}, target interface{}) *axerror.AXError {
	return c.doRequest("POST", path, body, target)
}

func (c *ArgoClient) put(path string, body interface{}, target interface{}) *axerror.AXError {
	return c.doRequest("PUT", path, body, target)
}

func (c *ArgoClient) delete(path string, target interface{}) *axerror.AXError {
	return c.doRequest("DELETE", path, nil, target)
}

func (c *ArgoClient) prepareRequest(method, path string, body interface{}) (*http.Request, *axerror.AXError) {
	url := fmt.Sprintf("%s/v1/%s", c.Config.URL, path)
	if c.Trace {
		log.Printf("%s %s", method, url)
	}
	var bodyBuff io.Reader
	if body != nil {
		jsonValue, err := json.Marshal(body)
		if err != nil {
			return nil, axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("%s: %s", url, err.Error())
		}
		bodyBuff = bytes.NewBuffer(jsonValue)
	}
	req, err := http.NewRequest(method, url, bodyBuff)
	if err != nil {
		return nil, axerror.ERR_AX_HTTP_CONNECTION.NewWithMessage(err.Error())
	}
	req.SetBasicAuth(c.Config.Username, c.Config.Password)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// doRequest is a helper to marshal a JSON body (if supplied) as part of a request, and decode a JSON response into the target interface
// Returns a decoded axErr if API returns back an error
func (c *ArgoClient) doRequest(method, path string, body interface{}, target interface{}) *axerror.AXError {
	req, axErr := c.prepareRequest(method, path, body)
	if axErr != nil {
		return axErr
	}
	res, err := c.client.Do(req)
	if err != nil {
		return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessage(err.Error())
	}
	return c.handleResponse(res, target)
}

func (c *ArgoClient) handleErrResponse(res *http.Response) *axerror.AXError {
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessagef("Server returned status %d, but failed to read response body: %s", res.StatusCode, err.Error())
	}
	var axErr axerror.AXError
	err = json.Unmarshal(body, &axErr)
	if err != nil {
		if c.Trace {
			fmt.Println(err)
			fmt.Println(string(body))
		}
		return axerror.ERR_AX_INTERNAL.NewWithMessagef("Server returned status %d, but failed to decode response body: %s", res.StatusCode, err)
	}
	if c.Trace {
		fmt.Printf("Server returned %d: %s: %s\n", res.StatusCode, axErr.Code, axErr.Message)
	}
	return &axErr
}

// handleResponse JSON decodes the body of an HTTP response into the target interface
func (c *ArgoClient) handleResponse(res *http.Response, target interface{}) *axerror.AXError {
	if res.StatusCode >= 400 {
		return c.handleErrResponse(res)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	decodeErr := "Failed to decode response body"
	if target != nil {
		err = json.Unmarshal(body, target)
		if err != nil {
			decodeErr = fmt.Sprintf("%s: %s", decodeErr, err.Error())
			if c.Trace {
				fmt.Println(decodeErr)
				fmt.Println(string(body))
			}
			return axerror.ERR_AX_INTERNAL.NewWithMessage(decodeErr)
		}
	}
	return nil
}
