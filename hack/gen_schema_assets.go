package main

import (
	"encoding/json"
	"io/ioutil"
)

/*
	Generate JSON schemas good enough for the UI to validate against.

	We're a bit lazy with these, we do not scrub the assets, and we just bundle all
	the definitions, when we could cherry-pick the relevant ones.
*/
func genSchemaAssets() {
	data, err := ioutil.ReadFile("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := swagger{}
	err = json.Unmarshal(data, &swagger)
	if err != nil {
		panic(err)
	}

	for _, kind := range kinds {
		name := "io.argoproj.workflow.v1alpha1." + kind
		filename := "ui/src/app/assets/schemas/" + kind + ".json"

		println(filename)

		schema := swagger.definitionByName(name)
		delete(schema["properties"].(obj), "status")
		schema["definitions"] = swagger["definitions"]
		definitions := schema["definitions"].(obj)
		delete(definitions, name)
		data, err = json.MarshalIndent(schema, "", "  ")
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filename, data, 0666)
		if err != nil {
			panic(err)
		}
	}

}
