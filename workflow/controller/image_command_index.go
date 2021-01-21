package controller

import (
	"encoding/json"
	"fmt"
	"os"
)

var imageCommandIndex = make(map[string][]string)

func init() {
	value, ok := os.LookupEnv("IMAGE_COMMAND_INDEX")
	if !ok {
		value = `
{
	"argoproj/argosay:v1": ["cowsay"],
	"argoproj/argosay:v2": ["/argosay"],
	"docker/whalesay:latest": ["cowsay"],
	"python:alpine3.6": ["python3"]
}`
	}
	err := json.Unmarshal([]byte(value), &imageCommandIndex)
	if err != nil {
		panic(fmt.Errorf("failed to parse IMAGE_COMMAND_INDEX=%s: %w", value, err))
	}
}
