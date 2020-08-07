package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strings"
)

type failure struct {
	Text string `xml:",chardata"`
}

type testcase struct {
	Failure failure `xml:"failure,omitempty"`
}

type testsuite struct {
	Name      string     `xml:"name,attr"`
	TestCases []testcase `xml:"testcase"`
}

type report struct {
	XMLName    xml.Name    `xml:"testsuites"`
	TestSuites []testsuite `xml:"testsuite"`
}

func printTestResultAnnotations() {
	data, err := ioutil.ReadFile("test-results/junit.xml")
	if err != nil {
		panic(err)
	}
	v := &report{}
	err = xml.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
	for _, s := range v.TestSuites {
		for _, c := range s.TestCases {
			if c.Failure.Text != "" {
				parts := strings.SplitN(c.Failure.Text, ":", 3)
				// Replace ‘/n’ with ‘%0A’ for multiple strings output.
				_, _ = fmt.Printf("::error file=%s,line=%v,col=0::%s", s.Name+"/"+parts[0], parts[1], strings.ReplaceAll(parts[2], "\n", "%0A"))
			}
		}
	}
}
