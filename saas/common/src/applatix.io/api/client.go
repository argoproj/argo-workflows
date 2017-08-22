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

	client  http.Client
	baseURL string
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
	client := ArgoClient{
		Config: config,
		client: http.Client{
			Jar:     cookieJar,
			Timeout: time.Minute,
		},
	}
	if config.Insecure != nil && *config.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.client.Transport = tr
	}
	return client
}

func (c *ArgoClient) fromURL(path string) string {
	return fmt.Sprintf("%s/v1/%s", c.Config.URL, path)
}

func (c *ArgoClient) get(path string, target interface{}) *axerror.AXError {
	url := c.fromURL(path)
	return c.doRequest("GET", url, nil, target)
}

func (c *ArgoClient) post(path string, body interface{}, target interface{}) *axerror.AXError {
	url := c.fromURL(path)
	return c.doRequest("POST", url, body, target)
}

func (c *ArgoClient) put(path string, body interface{}, target interface{}) *axerror.AXError {
	url := c.fromURL(path)
	return c.doRequest("PUT", url, body, target)
}

func (c *ArgoClient) delete(path string, target interface{}) *axerror.AXError {
	url := c.fromURL(path)
	return c.doRequest("DELETE", url, nil, target)
}

// doRequest is a helper to marshal a JSON body (if supplied) as part of a request, and decode a JSON response into the target interface
// Returns a decoded axErr if API returns back an error
func (c *ArgoClient) doRequest(method, url string, body interface{}, target interface{}) *axerror.AXError {
	if c.Trace {
		log.Printf("%s %s", method, url)
	}
	var bodyBuff io.Reader
	if body != nil {
		jsonValue, err := json.Marshal(body)
		if err != nil {
			return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("%s: %s", url, err.Error())
		}
		bodyBuff = bytes.NewBuffer(jsonValue)
	}
	req, err := http.NewRequest(method, url, bodyBuff)
	if err != nil {
		return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("%s: %s", url, err.Error())
	}
	req.SetBasicAuth(c.Config.Username, c.Config.Password)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("%s: %s", url, err.Error())
	}
	return c.handleResponse(res, target)
}

// handleResponse JSON decodes the body of an HTTP response into the target interface
func (c *ArgoClient) handleResponse(res *http.Response, target interface{}) *axerror.AXError {
	body, err := ioutil.ReadAll(res.Body)
	decodeErr := "Failed to decode response body"
	if res.StatusCode >= 400 {
		var axErr axerror.AXError
		err = json.Unmarshal(body, &axErr)
		if err != nil {
			decodeErr = fmt.Sprintf("%s: %s", decodeErr, err.Error())
			if c.Trace {
				fmt.Println(decodeErr)
				fmt.Println(string(body))
			}
			return axerror.ERR_AX_INTERNAL.NewWithMessagef(decodeErr)
		}
		if c.Trace {
			fmt.Printf("Server returned %d: %s: %s\n", res.StatusCode, axErr.Code, axErr.Message)
		}
		return &axErr
	}
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
