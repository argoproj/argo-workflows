// Package controller The API for a controller plugin.
//
//     Schemes: http
//     Host: localhost
//     BasePath: /api/v1
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
// swagger:meta
package controller

//go:generate env SWAGGER_GENERATE_EXTENSION=false swagger generate spec -o swagger.yml
//go:generate env SWAGGER_GENERATE_EXTENSION=false swagger generate markdown -f swagger.yml --output ../../../docs/controller_swagger.md
