package http

import "testing"

func Test_errFromResponse(t *testing.T) {
	for _, tt := range []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"200", 200, false},
		{"400", 400, true},
		{"500", 500, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := errFromResponse(tt.statusCode); (err != nil) != tt.wantErr {
				t.Errorf("errFromResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
