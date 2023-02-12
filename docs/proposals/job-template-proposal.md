# Proposal for Job Template

## Introduction

A job template is a way to define a job that can be used in a workflow. A job is an ordered list of named steps. For example:

```yaml
      job:
        image: golang:1.18
        workingDir: /go/src/github.com/golang/example
        steps:
          - name: clone
            run: git clone -v -b "{{workflow.parameters.branch}}" --single-branch --depth 1 https://github.com/golang/example.git .
          - name: deps
            run: go mod download -x
          - name: build
            run: go build ./...
          - name: test
            run: |
              go test -v ./... 2>&1 > test.out
          - name: test-report
            if: always()
            run: |
              go test -v ./... 2>&1 > test.out
```