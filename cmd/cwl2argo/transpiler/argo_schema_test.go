package transpiler

import "testing"

func TestSchema(t *testing.T) {
	if schema == "" {
		t.Errorf("schema was empty")
	}
	err := VerifyArgoSchema("")
	if err != nil {
		t.Errorf("%s", err)
	}
}
