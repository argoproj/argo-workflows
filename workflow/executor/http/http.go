package http

import (
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func SendHttpRequest(request *http.Request) (string, error) {
	out, err := http.DefaultClient.Do(request)

	if err != nil {
		return "", err
	}
	// Close the connection
	defer out.Body.Close()

	log.WithFields(log.Fields{"url": request.URL, "status": out.Status}).Info("HTTP request made")
	if out.StatusCode >= 300 {
		return "", fmt.Errorf(out.Status)
	}

	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil

}
