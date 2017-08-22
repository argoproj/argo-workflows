// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package main

import (
	"flag"
	"net/http"
	"strings"
	"text/template"

	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

var SWAGDIR = "/swagger"
var staticContent = flag.String("staticPath", SWAGDIR, "Path to folder with Swagger UI")
var apiurl = flag.String("api", "http://"+common.GetPublicDNS(), "The base path URI of the API service")

func swaggify(router *gin.Engine) {

	resourceListingJson = strings.Replace(resourceListingJson, "\"/v1\"", "\"/docs/v1\"", -1)

	for k, _ := range apiDescriptionsJson {
		apiDescriptionsJson[k] = strings.Replace(apiDescriptionsJson[k], "\"dataType\": \"bool\"", "\"dataType\": \"boolean\"", -1)
		apiDescriptionsJson[k] = strings.Replace(apiDescriptionsJson[k], "\"type\": \"bool\"", "\"type\": \"boolean\"", -1)
	}

	// Swagger Routes
	router.GET("/docs/v1/", IndexHandler)
	router.Static("/swagger", *staticContent)
	for apiKey := range apiDescriptionsJson {
		router.GET("/docs/v1/"+apiKey, ApiDescriptionHandler)
	}

	// API json data
	router.ApiDescriptionsJson = apiDescriptionsJson
}

func IndexHandler(c *gin.Context) {
	w := c.Writer
	r := c.Request

	isJsonRequest := false

	if acceptHeaders, ok := r.Header["Accept"]; ok {
		for _, acceptHeader := range acceptHeaders {
			if strings.Contains(acceptHeader, "json") {
				isJsonRequest = true
				break
			}
		}
	}

	if isJsonRequest {
		t, e := template.New("desc").Parse(resourceListingJson)
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Execute(w, *apiurl)
	} else {
		http.Redirect(w, r, "/swagger", http.StatusFound)
	}
}

func ApiDescriptionHandler(c *gin.Context) {
	w := c.Writer
	r := c.Request

	apiKey := strings.Replace(r.RequestURI, "/docs/v1/", "", -1)
	apiKey = strings.TrimRight(apiKey, "/")

	if json, ok := apiDescriptionsJson[apiKey]; ok {
		t, e := template.New("desc").Parse(json)
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Execute(w, *apiurl)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
