package fields

import (
	"encoding/json"
	"strings"
)

// CleanFields remove fields based on the query
// `status.phase` - only return status.phase
// `-status.phase`
func CleanFields(fieldsQuery string, dataBytes []byte) ([]byte, error) {
	if fieldsQuery == "" {
		return dataBytes, nil // abort early to avoid CPU and memory intensive json marshaling
	}
	exclude := false
	fields := make(map[string]interface{})
	if fieldsQuery != "" {
		if strings.HasPrefix(fieldsQuery, "-") {
			fieldsQuery = fieldsQuery[1:]
			exclude = true
		}
		for _, field := range strings.Split(fieldsQuery, ",") {
			fields[field] = true
		}
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(dataBytes, &data)
	if err != nil {
		return nil, err
	}
	processItem([]string{}, data, exclude, fields)
	clean, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return clean, nil
}

func processItem(path []string, item interface{}, exclude bool, fields map[string]interface{}) {
	if mapItem, ok := item.(map[string]interface{}); ok {
		for k, v := range mapItem {

			fieldPath := strings.Join(append(path, k), ".")
			_, pathIn := fields[fieldPath]
			parentPathIn := pathIn
			if !parentPathIn {
				for k := range fields {
					if strings.HasPrefix(k, fieldPath) {
						parentPathIn = true
						break
					}
				}
			}

			if exclude && !pathIn || !exclude && parentPathIn {
				if !pathIn {
					processItem(append(path, k), v, exclude, fields)
				}
			} else {
				delete(mapItem, k)
			}
		}
	} else if arrayItem, ok := item.([]interface{}); ok {
		for i := range arrayItem {
			processItem(path, arrayItem[i], exclude, fields)
		}
	}
}
