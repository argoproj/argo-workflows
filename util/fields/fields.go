package fields

import (
	"encoding/json"
	"strings"
)

func NewCleaner(x string) Cleaner {
	y := Cleaner{false, make(map[string]interface{})}
	if x != "" {
		if strings.HasPrefix(x, "-") {
			x = x[1:]
			y.exclude = true
		}
		for _, field := range strings.Split(x, ",") {
			y.fields[field] = true
		}
	}
	return y
}

type Cleaner struct {
	exclude bool
	fields  map[string]interface{}
}

func (f Cleaner) CleanFields(dataBytes []byte) ([]byte, error) {
	if len(f.fields) == 0 {
		return dataBytes, nil // abort early to avoid CPU and memory intensive json marshaling
	}
	data := make(map[string]interface{})
	err := json.Unmarshal(dataBytes, &data)
	if err != nil {
		return nil, err
	}
	f.cleanItem([]string{}, data)
	clean, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return clean, nil
}

func (f Cleaner) cleanItem(path []string, item interface{}) {
	if mapItem, ok := item.(map[string]interface{}); ok {
		for k, v := range mapItem {

			fieldPath := strings.Join(append(path, k), ".")
			_, pathIn := f.fields[fieldPath]
			parentPathIn := pathIn
			if !parentPathIn {
				for k := range f.fields {
					if strings.HasPrefix(k, fieldPath) {
						parentPathIn = true
						break
					}
				}
			}

			if f.exclude && !pathIn || !f.exclude && parentPathIn {
				if !pathIn {
					f.cleanItem(append(path, k), v)
				}
			} else {
				delete(mapItem, k)
			}
		}
	} else if arrayItem, ok := item.([]interface{}); ok {
		for i := range arrayItem {
			f.cleanItem(path, arrayItem[i])
		}
	}
}
