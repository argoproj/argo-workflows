package main

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"os"
	"sort"
	"strings"
)

type obj = map[string]interface{}

func main() {
	swagger := obj{}

	f, err := os.Open("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(f).Decode(&swagger)
	if err != nil {
		panic(err)
	}

	resources := map[string]map[string]bool{}

	for _, path := range swagger["paths"].(obj) {
		for _, method := range path.(obj) {
			operationId := method.(obj)["operationId"].(string)
			act, resource := auth.ParseMethod(strings.Split(operationId, "_")[1])
			if _, ok := resources[resource]; !ok {
				resources[resource] = map[string]bool{}
			}
			resources[resource][act] = true
		}
	}

	for resource, acts := range resources {
		var x []string
		for act := range acts {
			x = append(x, act)
		}
		sort.Strings(x)
		_, _ = os.Stdout.WriteString(fmt.Sprintf("| %s | %s |\n", resource, strings.Join(x, ",")))
	}
}
