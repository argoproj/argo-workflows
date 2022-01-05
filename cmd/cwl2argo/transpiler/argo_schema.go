package transpiler

import (
	_ "embed"

	log "github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
)

var (
	schemaLocation = "https://raw.githubusercontent.com/argoproj/argo-workflows/master/api/jsonschema/schema.json"
)

//go:embed argoschema.json
var schema string

func VerifyArgoSchema(argo string) error {
	schemaLoader := gojsonschema.NewReferenceLoader(schemaLocation)

	// local here means local variable, this is obviously fetched remotely
	localSchema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		log.Warnf("Could not load %s, attempting to load embedded schema file", schemaLocation)
	}

	schemaLoader = gojsonschema.NewStringLoader(schema)

	localSchema, err = gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		log.Errorf("Could not load from %s and from the embeded file, something is probably wrong with your schema", schemaLoader)
		return err
	}
	log.Info("Successfully loaded schema from embedded file")
	_ = localSchema
	return nil
}
