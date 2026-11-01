package parameterization

import (
	"encoding/json"
	"os"
	"testing"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Adapter authors can validate an external inventory with the exact compiler
// used by the server. Fixtures and credentials are not part of this repository.
func TestExternalFormContractInventory(t *testing.T) {
	path := os.Getenv("ARGO_PARAMETERIZATION_CONTRACT_FIXTURE")
	if path == "" {
		t.Skip("external contract inventory is opt-in")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var contracts []struct {
		Model p.Model              `json:"model"`
		Form  api.WorkflowFormSpec `json:"form"`
	}
	if err := json.Unmarshal(raw, &contracts); err != nil {
		t.Fatal(err)
	}
	if len(contracts) == 0 {
		t.Fatal("empty inventory")
	}
	failed := 0
	for i, contract := range contracts {
		if errors := p.ValidateModelSchema(contract.Model); len(errors) > 0 {
			if failed < 10 {
				t.Errorf("contract %d: %v", i, errors)
			}
			failed++
			continue
		}
		if err := validateForm(contract.Form, contract.Model, contract.Form.ParameterTemplate); err != nil {
			if failed < 10 {
				t.Errorf("contract %d: %v", i, err)
			}
			failed++
		}
	}
	if failed > 0 {
		t.Fatalf("%d/%d contracts rejected", failed, len(contracts))
	}
	t.Logf("validated %d model/form contracts", len(contracts))
}
