// Package sqldb implements workflow archiving
package sqldb

import (
	"encoding/json"
	"strconv"
)

type pair struct {
	k string
	v interface{}
}

func convertStrings(m map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range m {
		switch v := v.(type) {
		case string:
			tmp := strconv.QuoteToASCII(v)
			tmp = tmp[1:]
			tmp = tmp[:len(tmp)-1]
			m[k] = tmp
		case map[string]interface{}:
			convertedInner, err := convertStrings(v)
			if err != nil {
				return nil, err
			}
			m[k] = convertedInner
		case []any:
			newList := []any{}
			for _, val := range v {
				// bit of a hack to reuse code
				valM := map[string]interface{}{"entry": val}
				retVal, err := convertStrings(valM)
				if err != nil {
					return nil, err
				}
				newList = append(newList, retVal["entry"])
			}
			m[k] = newList
		default:
			m[k] = v
		}
	}
	return m, nil
}

func convertMap(jsonObject any) (map[string]interface{}, error) {

	bytes, err := json.Marshal(jsonObject)
	if err != nil {
		return nil, err
	}
	// we do this to handle json tags without explicitly programming this
	// ourselves.
	oldMap := make(map[string]interface{})
	err = json.Unmarshal(bytes, &oldMap)
	if err != nil {
		return nil, err
	}

	newMap, err := convertStrings(oldMap)
	if err != nil {
		return nil, err
	}

	return newMap, nil
}

func jsonMarshallRawStrings(jsonObject any) ([]byte, error) {
	m, err := convertMap(jsonObject)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, nil
}
