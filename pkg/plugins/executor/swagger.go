// Package executor The API for an executor plugin.
//
//	Schemes: http
//	Host: localhost
//	BasePath: /api/v1
//	Version: 0.0.1
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
// swagger:meta
package executor

//go:generate env SWAGGER_GENERATE_EXTENSION=false swagger generate spec -o swagger.yml
//go:generate env SWAGGER_GENERATE_EXTENSION=false swagger generate markdown -f swagger.yml --output ../../../docs/executor_swagger.md
