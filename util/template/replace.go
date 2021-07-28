package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/valyala/fasttemplate"

	exprenv "github.com/argoproj/argo-workflows/v3/util/expr/env"
)

const (
	prefix = "{{"
	suffix = "}}"
)

func replace(v interface{}, f func(string) (string, error)) (interface{}, error) {
	switch x := v.(type) {
	case string:
		y, err := f(x)
		return y, err
	case []interface{}:
		for m, n := range x {
			y, err := replace(n, f)
			if err != nil {
				return nil, err
			}
			x[m] = y
		}
		return x, nil
	case map[string]interface{}:
		for m, n := range x {
			y, err := replace(n, f)
			if err != nil {
				return nil, err
			}
			x[m] = y
		}
		return x, nil
	default:
		// int, float etc
		return v, nil
	}
}

func Replace(obj interface{}, replaceMap map[string]string, allowUnresolved bool) error {
	switch kind := reflect.ValueOf(obj).Kind(); kind {
	case reflect.Ptr, reflect.Slice, reflect.Map:
	default:
		return fmt.Errorf("obj must be pointer, slice or map, but is %q", kind)
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	log.Debugf("replacing %T; %q", obj, data)
	var x interface{}
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	replaceText := func(text string) (string, error) {
		template, err := fasttemplate.NewTemplate(text, prefix, suffix)
		if err != nil {
			return "", err
		}
		replacedTmpl := &bytes.Buffer{}
		_, err = template.ExecuteFunc(replacedTmpl, func(w io.Writer, tag string) (int, error) {
			kind, expression := parseTag(tag)
			switch kind {
			case kindExpression:
				env := exprenv.GetFuncMap(envMap(replaceMap))
				return expressionReplace(w, expression, env, allowUnresolved)
			default:
				return simpleReplace(w, tag, replaceMap, allowUnresolved)
			}
		})
		return replacedTmpl.String(), err
	}
	y, err := replace(x, replaceText)
	if err != nil {
		return err
	}
	data, err = json.Marshal(y)
	if err != nil {
		return err
	}
	log.Debugf("replaced  %T: %q", obj, data)
	return json.Unmarshal(data, &obj)
}
