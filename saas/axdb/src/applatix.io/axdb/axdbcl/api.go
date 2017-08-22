// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package axdbcl

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/restcl"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

// get response into map. If there is an error, return nil
func parseResponse(res *http.Response) (map[string]interface{}, *axerror.AXError) {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage("failed to read response body")
	}

	bodyMap := make(map[string]interface{})
	err = json.Unmarshal(body, &bodyMap)
	if err != nil {
		return nil, axerror.ERR_AXDB_INTERNAL.NewWithMessage("failed to unmarshal response body")
	}

	if res.StatusCode != axdb.RestStatusOK {
		return nil, axerror.GetErrorFromMap(bodyMap, res.StatusCode)
	}

	return bodyMap, nil
}

func parseArrayResponse(res *http.Response, result interface{}) *axerror.AXError {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return axerror.ERR_AXDB_INTERNAL.NewWithMessage("failed to read response body")
	}

	if res.StatusCode != axdb.RestStatusOK {
		bodyMap := make(map[string]interface{})
		err = json.Unmarshal(body, &bodyMap)
		if err != nil {
			return axerror.ERR_AXDB_INTERNAL.NewWithMessage("failed to unmarshal response body")
		}
		return axerror.GetErrorFromMap(bodyMap, res.StatusCode)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return axerror.ERR_AXDB_INTERNAL.NewWithMessage("failed to unmarshal response body")
	}
	return nil
}

type AXDBClient struct {
	rooturl    string
	httpClient *http.Client
}

func (client *AXDBClient) GetRootUrl() string {
	return client.rooturl
}

// Do a put
func (client *AXDBClient) update(appName string, tableName string, protoName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	var jsonReader io.Reader = nil
	if payload != nil {
		payloadJson, err := json.Marshal(payload)
		if err != nil {
			return nil, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("failed to marshal payload")
		}
		jsonReader = bytes.NewBuffer(payloadJson)
	}

	req, err := http.NewRequest(protoName, fmt.Sprintf("%s/%s/%s", client.rooturl, appName, tableName), jsonReader)
	if err != nil {
		return nil, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("failed to create http request")
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("%s: failed to create http client", err.Error())
		return nil, axerror.ERR_AX_HTTP_CONNECTION.NewWithMessage(errMsg)
	}

	return parseResponse(res)
}

// Do a put
func (client *AXDBClient) updateWithTimeRetry(appName string, tableName string, protoName string, payload interface{}, retryConfig *restcl.RetryConfig) (map[string]interface{}, *axerror.AXError) {

	errCodeMap := map[string]bool{
		axerror.ERR_AX_HTTP_CONNECTION.Code: true,
	}

	if retryConfig != nil && retryConfig.TriableCodes != nil {
		for _, errCode := range retryConfig.TriableCodes {
			errCodeMap[errCode] = true
		}
	}

	startTime := time.Now()
	var count int
	var result map[string]interface{}
	var axErr *axerror.AXError
	var gap time.Duration = time.Second

	if retryConfig == nil {
		result, axErr = client.update(appName, tableName, protoName, payload)
	} else {
		for {
			result, axErr = client.update(appName, tableName, protoName, payload)
			if axErr == nil {
				break
			} else {
				fmt.Printf("[RESTCL] http client (%v %v %v) try count %v, err: %v %v %v\n", client.GetRootUrl(), appName, tableName, count, axErr.Code, axErr.Message, axErr.Detail)
			}

			if !errCodeMap["*"] && !errCodeMap[axErr.Code] {
				break
			}

			count++
			if count != 1 {
				if gap < 60*time.Second {
					gap = 2 * gap
				}
				time.Sleep(gap)
				fmt.Printf("[RESTCL] http client (%v %v %v %v) try count %v, left over %v seconds\n", client.GetRootUrl(), appName, tableName, protoName, count, int64(retryConfig.Timeout.Seconds())-(time.Now().Unix()-startTime.Unix()))
			}

			if time.Now().Unix()-startTime.Unix() > int64(retryConfig.Timeout.Seconds()) {
				fmt.Printf("[RESTCL] http client (%v %v %v %v) gives up with timeout %v\n", client.GetRootUrl(), appName, tableName, protoName, retryConfig.Timeout.String())
				break
			}
		}
	}

	return result, axErr
}

func (client *AXDBClient) PutWithTimeRetry(appName string, tableName string, payload interface{}, retryConfig *restcl.RetryConfig) (map[string]interface{}, *axerror.AXError) {
	return client.updateWithTimeRetry(appName, tableName, "PUT", payload, retryConfig)
}

func (client *AXDBClient) PostWithTimeRetry(appName string, tableName string, payload interface{}, retryConfig *restcl.RetryConfig) (map[string]interface{}, *axerror.AXError) {
	return client.updateWithTimeRetry(appName, tableName, "POST", payload, retryConfig)
}

func (client *AXDBClient) DeleteWithTimeRetry(appName string, tableName string, payload interface{}, retryConfig *restcl.RetryConfig) (map[string]interface{}, *axerror.AXError) {
	return client.updateWithTimeRetry(appName, tableName, "DELETE", payload, retryConfig)
}

func (client *AXDBClient) GetWithTimeRetry(appName string, tableName string, params map[string]interface{}, result interface{}, retryConfig *restcl.RetryConfig) *axerror.AXError {

	errCodeMap := map[string]bool{
		axerror.ERR_AX_HTTP_CONNECTION.Code: true,
	}

	if retryConfig != nil && retryConfig.TriableCodes != nil {
		for _, errCode := range retryConfig.TriableCodes {
			errCodeMap[errCode] = true
		}
	}

	startTime := time.Now()
	var count int
	var axErr *axerror.AXError
	var gap time.Duration = time.Second

	if retryConfig == nil {
		axErr = client.Get(appName, tableName, params, result)
	} else {
		for {
			axErr = client.Get(appName, tableName, params, result)
			if axErr == nil {
				break
			} else {
				fmt.Printf("[RESTCL] http client (%v %v %v) try count %v, err: %v %v %v\n", client.GetRootUrl(), appName, tableName, count, axErr.Code, axErr.Message, axErr.Detail)
			}

			if !errCodeMap["*"] && !errCodeMap[axErr.Code] {
				break
			}

			count++
			if count != 1 {
				if gap < 60*time.Second {
					gap = 2 * gap
				}
				time.Sleep(gap)
				fmt.Printf("[RESTCL] http client (%v %v %v) try count %v, left over %v seconds\n", client.GetRootUrl(), appName, tableName, count, int64(retryConfig.Timeout.Seconds())-(time.Now().Unix()-startTime.Unix()))
			}

			if time.Now().Unix()-startTime.Unix() > int64(retryConfig.Timeout.Seconds()) {
				fmt.Printf("[RESTCL] http client (%v %v %v) gives up with timeout %v\n", client.GetRootUrl(), appName, tableName, retryConfig.Timeout.String())
				break
			}
		}
	}

	return axErr
}

func (client *AXDBClient) Put(appName string, tableName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	return client.update(appName, tableName, "PUT", payload)
}

func (client *AXDBClient) Post(appName string, tableName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	return client.update(appName, tableName, "POST", payload)
}

func (client *AXDBClient) Delete(appName string, tableName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	return client.update(appName, tableName, "DELETE", payload)
}

func (client *AXDBClient) Get(appName string, tableName string, params map[string]interface{}, result interface{}) *axerror.AXError {
	var getUrl *url.URL
	getUrl, err := url.Parse(fmt.Sprintf("%s/%s/%s", client.rooturl, appName, tableName))
	if err != nil {
		return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Failed to parse URL: %s/%s/%s", client.rooturl, appName, tableName))
	}

	if params != nil {
		urlValues := url.Values{}
		for k, v := range params {
			switch v.(type) {
			case string:
				urlValues.Add(k, fmt.Sprintf("%v", v))
			case []string:
				for _, value := range v.([]string) {
					urlValues.Add(k, value)
				}
			default:
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Failed to prepare the request, error: %v", err))
				}
				urlValues.Add(k, fmt.Sprintf("%v", string(jsonBytes)))
			}
		}
		getUrl.RawQuery = urlValues.Encode()
	}

	res, err := client.httpClient.Get(getUrl.String())
	if err != nil {
		return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessage(fmt.Sprintf("Failed to get HTTP request, error: %v", err))
	}

	return parseArrayResponse(res, result)
}

func (client *AXDBClient) CopyClient() *AXDBClient {
	return &AXDBClient{
		rooturl: client.rooturl,
		httpClient: &http.Client{
			Timeout: client.httpClient.Timeout,
		},
	}
}

//func NewAXDBClient(rootUrl string) *AXDBClient {
//	return &AXDBClient{
//		rooturl:    rootUrl,
//		httpClient: http.DefaultClient,
//	}
//}

func NewAXDBClientWithTimeout(rootUrl string, timeout time.Duration) *AXDBClient {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	return &AXDBClient{
		rooturl: rootUrl,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}
