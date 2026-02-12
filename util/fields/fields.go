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

func (f Cleaner) Clean(x, y any) (bool, error) {
	if len(f.fields) == 0 {
		return false, nil
	}
	v, err := json.Marshal(x)
	if err != nil {
		return false, err
	}
	data := make(map[string]any)
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
	}
	return !f.exclude
}

func (f Cleaner) matches(x string) bool {
	for y := range f.fields {
		if strings.HasPrefix(x, y) || strings.HasPrefix(y, x) {
			return true
		}
	}
	return false
}

func (f Cleaner) cleanItem(path []string, item any) {
	if mapItem, ok := item.(map[string]any); ok {
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
	} else if arrayItem, ok := item.([]any); ok {
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
