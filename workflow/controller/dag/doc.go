// Package dag evaluates the tasks of a DAG or Steps template: which are ready
// to run, which are waiting on dependencies, which can never run and should
// be omitted, and what a retry or task-group node's current state amounts
// to. It performs no side effects; the controller's Engine acts on its
// results. See README.md in this directory for how it works.
package dag
