// Copyright 2015-2016 Applatix, Inc. All rights reserved.

package restcl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"applatix.io/axerror"
	"net"
)

// Parse the response into the result object
func parseResponse(res *http.Response, result interface{}) *axerror.AXError {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprint("Failed to read response body, error:%v", err))
	}

	if res.StatusCode != axerror.REST_STATUS_OK && res.StatusCode != axerror.REST_CREATE_OK {
		bodyMap := make(map[string]interface{})
		err = json.Unmarshal(body, &bodyMap)
		if err != nil {
			fmt.Printf("Respond body: %v     %v\n", string(body), res.StatusCode)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal response body, error: %v", err))
		}
		return axerror.GetErrorFromMap(bodyMap, res.StatusCode)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		fmt.Println(body)
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal response body, error: %v", err))
	}
	return nil
}

type RestClient struct {
	rooturl    string
	httpClient *http.Client
}

func (client RestClient) GetRootUrl() string {
	return client.rooturl
}

// Do a put
func (client *RestClient) update(apiName string, protoName string, payload interface{}, useKafka bool) (map[string]interface{}, *axerror.AXError) {
	var jsonReader io.Reader = nil
	if payload != nil {
		payloadJson, err := json.Marshal(payload)
		if err != nil {
			return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to marshal payload, error:%v", err)
		}
		jsonReader = bytes.NewBuffer(payloadJson)
	}

	fmt.Printf("Request: %v\n", fmt.Sprintf("%s/%s", client.rooturl, apiName))
	req, err := http.NewRequest(protoName, fmt.Sprintf("%s/%s", client.rooturl, apiName), jsonReader)
	if err != nil {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to create http request, error:%v", err)
	}

	if !useKafka {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "application/vnd.kafka.json.v1+json")
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("Failed to create http client, error:%v", err)
	}
	var result map[string]interface{}
	axErr := parseResponse(res, &result)
	return result, axErr
}

func (client *RestClient) update2(apiName string, protoName string, params map[string]interface{}, payload interface{}, response interface{}, useKafka bool) (*axerror.AXError, int) {

	var updateUrl *url.URL
	var err error
	if apiName != "" {
		updateUrl, err = url.Parse(fmt.Sprintf("%s/%s", client.rooturl, apiName))
	} else {
		updateUrl, err = url.Parse(fmt.Sprintf("%s", client.rooturl))
	}

	if err != nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to parse URL: %s/%s, error:%v", client.rooturl, apiName, err), axerror.REST_BAD_REQ
	}

	if params != nil {
		urlValues := url.Values{}
		for k, v := range params {
			if mapVal, ok := v.(map[string]interface{}); ok {
				jsonBytes, err := json.Marshal(mapVal)
				if err != nil {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to prepare the request, error: %v", err), axerror.REST_BAD_REQ
				}
				urlValues.Add(k, fmt.Sprintf("%v", string(jsonBytes)))
			} else {
				urlValues.Add(k, fmt.Sprintf("%v", v))
			}
		}
		updateUrl.RawQuery = urlValues.Encode()
	}

	var jsonReader io.Reader = nil
	if payload != nil {
		payloadJson, err := json.Marshal(payload)
		if err != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to marshal payload, error:%v", err), axerror.REST_BAD_REQ
		}
		jsonReader = bytes.NewBuffer(payloadJson)
	}

	req, err := http.NewRequest(protoName, updateUrl.String(), jsonReader)

	if err != nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Failed to create http request, error:%v", err), axerror.REST_INTERNAL_ERR
	}
	if !useKafka {
		req.Header.Set("Content-Type", "application/json")

		// Work around for unit test to pass
		if params != nil && params["session"] != nil {
			session := params["session"].(string)
			req.Header.Set("Cookie", "session_token="+session)
		}
	} else {
		req.Header.Set("Content-Type", "application/vnd.kafka.json.v1+json")
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		return axerror.ERR_AX_HTTP_CONNECTION.NewWithMessagef("Failed to create http client, error:%v", err), axerror.REST_INTERNAL_ERR
	}

	axErr := parseResponse(res, response)
	return axErr, res.StatusCode
}

func (client *RestClient) updateWithTimeRetry(apiName string, protoName string, params map[string]interface{}, payload interface{}, response interface{}, retryConfig *RetryConfig) (*axerror.AXError, int) {

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
	var code int
	var axErr *axerror.AXError
	var gap time.Duration = time.Second

	if retryConfig == nil {
		axErr, code = client.update2(apiName, protoName, params, payload, response, false)
	} else {
		for {
			axErr, code = client.update2(apiName, protoName, params, payload, response, false)
			if axErr == nil {
				break
			} else {
				fmt.Printf("[RESTCL] http client (%v %v %v) try count %v, err: %v %v %v\n", client.GetRootUrl(), apiName, protoName, count, axErr.Code, axErr.Message, axErr.Detail)
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
				fmt.Printf("[RESTCL] http client (%v %v %v) try count %v, left over %v seconds\n", client.GetRootUrl(), apiName, protoName, count, int64(retryConfig.Timeout.Seconds())-(time.Now().Unix()-startTime.Unix()))
			}

			if time.Now().Unix()-startTime.Unix() > int64(retryConfig.Timeout.Seconds()) {
				fmt.Printf("[RESTCL] http client (%v %v %v) gives up with timeout %v\n", client.GetRootUrl(), apiName, protoName, retryConfig.Timeout.String())
				break
			}
		}
	}

	return axErr, code
}

func (client *RestClient) Put(apiName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	return client.update(apiName, "PUT", payload, false)
}

func (client *RestClient) Post(apiName string, payload interface{}, useKafka ...bool) (map[string]interface{}, *axerror.AXError) {
	if len(useKafka) == 0 || !useKafka[0] {
		return client.update(apiName, "POST", payload, false)
	} else {
		return client.update(apiName, "POST", payload, true)
	}
}

func (client *RestClient) Put2(apiName string, params map[string]interface{}, payload, response interface{}) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}
	return client.update2(apiName, "PUT", params, payload, response, false)
}

func (client *RestClient) Post2(apiName string, params map[string]interface{}, payload, response interface{}, useKafka ...bool) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}

	if len(useKafka) == 0 || !useKafka[0] {
		return client.update2(apiName, "POST", params, payload, response, false)
	} else {
		return client.update2(apiName, "POST", params, payload, response, true)
	}

}

func (client *RestClient) Delete(apiName string, payload interface{}) (map[string]interface{}, *axerror.AXError) {
	return client.update(apiName, "DELETE", payload, false)
}

func (client *RestClient) Delete2(apiName string, params map[string]interface{}, payload, response interface{}) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}
	return client.update2(apiName, "DELETE", params, payload, response, false)
}

func (client *RestClient) Get(apiName string, params map[string]interface{}, result interface{}) *axerror.AXError {
	axErr, _ := client.update2(apiName, "GET", params, nil, result, false)
	return axErr
}

func (client *RestClient) GetWithTimeRetry(apiName string, params map[string]interface{}, result interface{}, retryConfig *RetryConfig) (*axerror.AXError, int) {
	axErr, code := client.updateWithTimeRetry(apiName, "GET", params, nil, result, retryConfig)
	return axErr, code
}

func (client *RestClient) PostWithTimeRetry(apiName string, params map[string]interface{}, payload, response interface{}, retryConfig *RetryConfig) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}
	return client.updateWithTimeRetry(apiName, "POST", params, payload, response, retryConfig)

}

func (client *RestClient) PutWithTimeRetry(apiName string, params map[string]interface{}, payload, response interface{}, retryConfig *RetryConfig) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}
	return client.updateWithTimeRetry(apiName, "PUT", params, payload, response, retryConfig)
}

func (client *RestClient) DeleteWithTimeRetry(apiName string, params map[string]interface{}, payload, response interface{}, retryConfig *RetryConfig) (*axerror.AXError, int) {
	if response == nil {
		response = &map[string]interface{}{}
	}
	return client.updateWithTimeRetry(apiName, "DELETE", params, payload, response, retryConfig)
}

//func NewRestClient(rootUrl string) *RestClient {
//	return &RestClient{rootUrl, http.DefaultClient}
//}

func NewRestClientWithTimeout(rootUrl string, timeout time.Duration) *RestClient {

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

	return &RestClient{
		rooturl: rootUrl,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}
