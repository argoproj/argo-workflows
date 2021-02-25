package util

import (
	"io/ioutil"
	"os"
)

func ReadTerminationMessage() (string, error) {
	data, err := ioutil.ReadFile("/dev/termination-log")
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(data), err
}

func WriteTerminationMessage(message string) {
	err := ioutil.WriteFile("/dev/termination-log", []byte(message), 0644)
	if err != nil {
		panic(err)
	}
}
