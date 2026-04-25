package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/yaml"
)

func main() {
	inPath := "api/openapi-spec/swagger.json"
	outPath := "api/openapi-spec/openapi.yaml"

	data, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", inPath, err)
		os.Exit(1)
	}

	var doc2 openapi2.T
	if err := json.Unmarshal(data, &doc2); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse swagger v2: %v\n", err)
		os.Exit(1)
	}

	loader := openapi3.NewLoader()
	doc3, err := openapi2conv.ToV3WithLoader(&doc2, loader, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert to openapi v3: %v\n", err)
		os.Exit(1)
	}

	if err := doc3.Validate(loader.Context); err != nil {
		fmt.Fprintf(os.Stderr, "openapi v3 validation failed: %v\n", err)
		os.Exit(1)
	}

	raw, err := doc3.MarshalYAML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal openapi v3 as YAML: %v\n", err)
		os.Exit(1)
	}

	// MarshalYAML returns interface{}, encode it properly
	out, err := yaml.Marshal(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode YAML: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s\n", outPath)
}
