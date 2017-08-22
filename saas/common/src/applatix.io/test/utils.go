// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"applatix.io/axerror"
	"github.com/gin-gonic/gin"
)

func fakeFixtureHandler(c *gin.Context) {
	nullResult := make(map[string]interface{})
	fmt.Printf("====== fake processing %s %s/%s ===========", c.Request.Method, c.Param("version"), c.Param("method"))
	switch c.Request.Method {
	case "POST", "PUT":
		payload := map[string]interface{}{}
		err := getUnmarshalledBody(c, &payload)
		if err != nil {
			panic(err)
		}
		convertFloatToInt64(payload)
		c.JSON(axerror.REST_STATUS_OK, payload)
	case "GET", "DELETE":
		c.JSON(axerror.REST_STATUS_OK, nullResult)
	}
}

func fakeHandler(c *gin.Context) {
	nullResult := make(map[string]interface{})
	fmt.Printf("====== fake processing %s %s/%s ===========", c.Request.Method, c.Param("version"), c.Param("method"))
	c.JSON(axerror.REST_STATUS_OK, nullResult)
}

var convertFloatToInt64 = func(serviceMap map[string]interface{}) {
	for fieldName, _ := range serviceMap {
		if v := serviceMap[fieldName]; v != nil {
			if _, ok := v.(float64); ok {
				serviceMap[fieldName] = int64(v.(float64))
			}
		}
	}
}

// start http server that always succeeds, so that we can use it to pretend that it's the gateway or axmon server
func StartFixtureFakeRouter(port int) {
	router := gin.Default()

	router.GET("/:version/:method/:id", fakeFixtureHandler)
	router.POST("/:version/:method/:id", fakeFixtureHandler)
	router.PUT("/:version/:method/:id", fakeFixtureHandler)
	router.DELETE("/:version/:method/:id", fakeFixtureHandler)

	router.GET("/:version/:method/:id/:id", fakeFixtureHandler)
	router.POST("/:version/:method/:id/:id", fakeFixtureHandler)
	router.PUT("/:version/:method/:id/:id", fakeFixtureHandler)
	router.DELETE("/:version/:method/:id/:id", fakeFixtureHandler)

	router.GET("/:version/:method/:id/:id/:id", fakeFixtureHandler)
	router.POST("/:version/:method/:id/:id/:id", fakeFixtureHandler)
	router.PUT("/:version/:method/:id/:id/:id", fakeFixtureHandler)
	router.DELETE("/:version/:method/:id/:id/:id", fakeFixtureHandler)

	router.GET("/:version/:method/:id/:id/:id/:id", fakeFixtureHandler)
	router.POST("/:version/:method/:id/:id/:id/:id", fakeFixtureHandler)
	router.PUT("/:version/:method/:id/:id/:id/:id", fakeFixtureHandler)
	router.DELETE("/:version/:method/:id/:id/:id/:id", fakeFixtureHandler)

	router.Run(fmt.Sprintf(":%v", port))
}

func StartFakeRouter(port int) {
	router := gin.Default()

	router.GET("/:version/:method/", fakeHandler)
	router.POST("/:version/:method/", fakeHandler)
	router.PUT("/:version/:method/", fakeHandler)
	router.DELETE("/:version/:method/", fakeHandler)

	router.GET("/:version/:method", fakeHandler)
	router.POST("/:version/:method", fakeHandler)
	router.PUT("/:version/:method", fakeHandler)
	router.DELETE("/:version/:method", fakeHandler)

	router.GET("/:version/:method/:id", fakeHandler)
	router.POST("/:version/:method/:id", fakeHandler)
	router.PUT("/:version/:method/:id", fakeHandler)
	router.DELETE("/:version/:method/:id", fakeHandler)

	router.GET("/:version/:method/:id/", fakeHandler)
	router.POST("/:version/:method/:id/", fakeHandler)
	router.PUT("/:version/:method/:id/", fakeHandler)
	router.DELETE("/:version/:method/:id/", fakeHandler)

	router.GET("/:version/:method/:id/:id", fakeHandler)
	router.POST("/:version/:method/:id/:id", fakeHandler)
	router.PUT("/:version/:method/:id/:id", fakeHandler)
	router.DELETE("/:version/:method/:id/:id", fakeHandler)

	router.Run(fmt.Sprintf(":%v", port))
}

func RandStr() string {
	return strconv.Itoa(rand.Int())
}

func getBodyString(c *gin.Context) ([]byte, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(c.Request.Body)
	if err != nil {
		return nil, err
	}
	body := buffer.Bytes()
	return body, nil
}

func getUnmarshalledBody(c *gin.Context, obj interface{}) error {
	body, err := getBodyString(c)
	if err != nil {
		return err
	}

	jsonErr := json.Unmarshal(body, obj)
	if jsonErr != nil {
		return jsonErr
	}

	return nil
}

func NewTrue() *bool {
	b := true
	return &b
}

func NewFalse() *bool {
	b := false
	return &b
}
