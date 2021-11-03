package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func SendHttpRequest(request *http.Request, timeout *int64) (string, error) {
	httpClient := http.DefaultClient
	if timeout != nil {
		httpClient.Timeout = time.Duration(*timeout) * time.Second
	}
	out, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer out.Body.Close()

	log.WithFields(log.Fields{"url": request.URL, "status": out.Status}).Info("HTTP Request Sent")
	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return "", err
	}
	if out.StatusCode >= 300 {
		return "", fmt.Errorf(out.Status)
	}

	return string(data), nil
}
