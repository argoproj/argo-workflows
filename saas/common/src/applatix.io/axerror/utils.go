package axerror

import "fmt"

func (e *AXError) New() *AXError {
	r := *e
	return &r
}

func (e *AXError) NewWithMessage(message string) *AXError {
	return &AXError{Code: e.Code, Message: message}
}

func (e *AXError) NewWithMessagef(format string, a ...interface{}) *AXError {
	return &AXError{Code: e.Code, Message: fmt.Sprintf(format, a...)}
}

func (e *AXError) Equals(err *AXError) bool {
	return e.Code == err.Code
}

// Converts a JSON map that contains an AXError to AXError struct. If there is no map, use the http status code to make a guess
func GetErrorFromMap(m map[string]interface{}, status int) *AXError {
	v, ok := m["code"]
	if !ok {
		if status >= 500 {
			return ERR_AX_INTERNAL.NewWithMessagef("server returned http status code %d", status)
		} else {
			return ERR_API_INVALID_PARAM.NewWithMessagef("server returned http status code %d", status)
		}
	}
	e := AXError{Code: v.(string)}
	v, ok = m["message"]
	if ok {
		e.Message = v.(string)
	}
	v, ok = m["detail"]
	if ok {
		e.Detail = v.(string)
	}
	return &e
}

// this makes AXError implement error interface
func (e *AXError) Error() string {
	return e.Message
}
