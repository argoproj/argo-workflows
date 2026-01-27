package fields

import (
	"encoding/json"
	"strings"
)

func NewCleaner(x string) Cleaner {
	y := Cleaner{false, make(map[string]bool)}
	if x != "" {
		if strings.HasPrefix(x, "-") {
			x = x[1:]
			y.exclude = true
		}
		for field := range strings.SplitSeq(x, ",") {
			y.fields[field] = true
		}
	}
	return y
}

type Cleaner struct {
	exclude bool
	fields  map[string]bool
}

func (f Cleaner) Clean(x, y interface{}) (bool, error) {
	if len(f.fields) == 0 {
		return false, nil
	}
	v, err := json.Marshal(x)
	if err != nil {
		return false, err
	}
	data := make(map[string]interface{})
	if err := json.Unmarshal(v, &data); err != nil {
		return false, err
	}
	f.cleanItem([]string{}, data)
	w, err := json.Marshal(data)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(w, &y); err != nil {
		return false, err
	}
	return true, nil
}

func (f Cleaner) WillExclude(x string) bool {
	if len(f.fields) == 0 {
		return false
	}
	if f.matches(x) {
		return f.exclude
	} else {
		return !f.exclude
	}
}

func (f Cleaner) matches(x string) bool {
	for y := range f.fields {
		if strings.HasPrefix(x, y) || strings.HasPrefix(y, x) {
			return true
		}
	}
	return false
}

func (f Cleaner) cleanItem(path []string, item interface{}) {
	if mapItem, ok := item.(map[string]interface{}); ok {
		// Special handling for status.nodes map
		if len(path) > 0 && path[len(path)-1] == "nodes" {
			nodePath := strings.Join(path, ".")

			// Collect fields for node filtering
			includeFields := make(map[string]bool)
			excludeFields := make(map[string]bool)

			for field := range f.fields {
				fieldName := field
				var exclude bool

				if strings.HasPrefix(field, "-") {
					fieldName = field[1:]
					exclude = true
				}

				if strings.HasPrefix(fieldName, nodePath+".") {
					parts := strings.SplitN(strings.TrimPrefix(fieldName, nodePath+"."), ".", 2)
					if len(parts) == 1 {
						if exclude {
							excludeFields[parts[0]] = true
						} else {
							includeFields[parts[0]] = true
						}
					}
				}
			}

			// Apply filtering to node fields if needed
			if len(includeFields) > 0 || len(excludeFields) > 0 {
				for _, nodeMap := range mapItem {
					if nodeObj, ok := nodeMap.(map[string]interface{}); ok {
						for key := range nodeObj {
							if excludeFields[key] || !f.exclude && len(includeFields) > 0 && !includeFields[key] {
								delete(nodeObj, key)
							}
						}
					}
				}
				return
			}
		}

		// Standard field handling (similar to original)
		for k, v := range mapItem {
			fieldPath := strings.Join(append(path, k), ".")
			_, pathIn := f.fields[fieldPath]
			parentPathIn := pathIn

			// Check for parent paths
			if !parentPathIn {
				for field := range f.fields {
					// Check for field without "-" prefix
					checkField := field
					if strings.HasPrefix(field, "-") {
						checkField = field[1:]
					}

					// More precise prefix check with dot
					if strings.HasPrefix(checkField, fieldPath+".") {
						parentPathIn = true
						break
					}
				}
			}

			// Check for fields with "-" prefix
			shouldExclude := false
			for field := range f.fields {
				if strings.HasPrefix(field, "-") && fieldPath == field[1:] {
					shouldExclude = true
					break
				}
			}

			if shouldExclude || (f.exclude && pathIn) || (!f.exclude && !parentPathIn) {
				delete(mapItem, k)
			} else {
				f.cleanItem(append(path, k), v)
			}
		}
	} else if arrayItem, ok := item.([]interface{}); ok {
		for i := range arrayItem {
			f.cleanItem(path, arrayItem[i])
		}
	}
}

func (f Cleaner) WithoutPrefix(prefix string) Cleaner {
	y := Cleaner{f.exclude, map[string]bool{}}
	for k, v := range f.fields {
		y.fields[strings.TrimPrefix(k, prefix)] = v
	}
	return y
}
