package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

var attributeHeaderTmpl = template.Must(template.New("header").Parse(`{{.Banner}}
    //
    //go:generate go run ./builder --attributesGo {{.Filename}}
    package telemetry

    type Attributes []Attribute
    type Attribute struct {
    	Name  string
    	Value interface{}
    }

`))

func createAttributesGo(filename string, attributes *attributesList) error {
	err := writeAttributesGo(filename, attributes)
	if err != nil {
		return err
	}
	return goFmtFile(filename)
}

func writeAttributesGo(filename string, attributes *attributesList) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	err = attributeHeaderTmpl.Execute(f, map[string]string{"Banner": generatedBanner, "Filename": filepath.Base(filename)})
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "const(\n")
	for _, attrib := range *attributes {
		fmt.Fprintf(f, "\tAttrib%s string = `%s`\n", attrib.Name, attrib.displayName())
	}
	fmt.Fprintf(f, ")\n")
	return nil
}

func goFmtFile(filename string) error {
	cmd := exec.Command("go", "fmt", filename)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}
