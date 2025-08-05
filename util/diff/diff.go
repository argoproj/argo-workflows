package diff

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
)

func LogChanges(old, new interface{}) {
	if !log.IsLevelEnabled(log.DebugLevel) {
		return
	}
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(new)
	patch, _ := jsonpatch.CreateMergePatch(a, b)
	log.Debugf("Log changes patch: %s", string(patch))
}
