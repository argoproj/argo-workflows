package parametrizable

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Int string

func (l *Int) MarshalJSON() ([]byte, error) {
	return json.Marshal(l)
}

func (l *Int) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		if strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}") {
			*l = Int(value)
			return nil
		}
		if _, err := strconv.Atoi(value); err == nil {
			*l = Int(value)
			return nil
		}
	case float64:
		if value == float64(int(value)) {
			*l = Int(fmt.Sprintf("%d", int(value)))
			return nil
		}
	}
	return fmt.Errorf("cannot parse: %s", b)
}

func (l *Int) Int() (int, error) {
	return strconv.Atoi(string(*l))
}

func (l *Int) Int32() (int32, error) {
	i, err := strconv.Atoi(string(*l))
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}

func (l *Int) Int64() (int64, error) {
	i, err := strconv.Atoi(string(*l))
	if err != nil {
		return 0, err
	}
	return int64(i), nil
}

