package execution

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCanonicalFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/run-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	if err := run.Validate(); err != nil {
		t.Fatal(err)
	}
	if run.Result == nil || !run.Result.Passed || len(run.Attempts) != 2 {
		t.Fatalf("fixture lost execution detail: %+v", run)
	}
	// Cost travels under the names a run evidence record uses.
	if a := run.Attempts[1]; a.Provider != "openai" || a.TokensIn != 2400 || a.TokensOut != 600 || a.Cost != 0.0421 || a.Currency != "USD" {
		t.Fatalf("fixture lost cost: %+v", a)
	}
}

func TestValidRunID(t *testing.T) {
	for _, id := range []string{"run-000001", "run-1234567"} {
		if !ValidRunID(id) {
			t.Fatalf("valid id rejected: %s", id)
		}
	}
	for _, id := range []string{"TASK-001", "run-1", "run-00000x"} {
		if ValidRunID(id) {
			t.Fatalf("invalid id accepted: %s", id)
		}
	}
}
