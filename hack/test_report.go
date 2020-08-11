package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strconv"
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

func testReport() {
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
				x := newFailureText(s.Name, c.Failure.Text)
				if x.file == "" {
					continue
				}
				// https://docs.github.com/en/actions/reference/workflow-commands-for-github-actions#setting-an-error-message
				// Replace ‘/n’ with ‘%0A’ for multiple strings output.
				_, _ = fmt.Printf("::error file=%s,line=%v,col=0::%s\n", x.file, x.line, x.message)
			}
		}
	}
}

type failureText struct {
	file    string
	line    int
	message string
}

func trimStdoutLines(text string) string {
	split := strings.Split(text, "\n")
	for i, s := range split {
		if strings.Contains(s, "_test.go") {
			return strings.Join(split[i:], "\n")
		}
	}
	return text
}

func newFailureText(suite, text string) failureText {
	text = trimStdoutLines(text)
	parts := strings.SplitN(text, ":", 3)
	if len(parts) != 3 {
		return failureText{}
	}
	file := strings.TrimPrefix(suite, "github.com/argoproj/argo/") + "/" + parts[0]
	line, _ := strconv.Atoi(parts[1])
	message := strings.ReplaceAll(strings.TrimSpace(parts[2]), "\n", "%0A")
	return failureText{
		file:    file,
		line:    line,
		message: message,
	}
}
