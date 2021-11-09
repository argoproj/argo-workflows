package patch

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
)

func Obj(old interface{}, patch interface{}) error {
	orig, err := json.Marshal(old)
	if err != nil {
		return err
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	mergePatch, err := jsonpatch.CreateMergePatch([]byte("{}"), patchBytes)
	if err != nil {
		return err
	}
	data, err := jsonpatch.MergePatch(orig, mergePatch)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(data, old)
	if err != nil {
		return err
	}
	return nil
}
