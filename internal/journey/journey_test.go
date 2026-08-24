package journey

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadAndFindExample(t *testing.T) {
	dir := exampleDir(t)
	j, err := Find(dir, "create-service")
	if err != nil {
		t.Fatal(err)
	}
	if j.Name != "create-service" {
		t.Fatalf("name = %s", j.Name)
	}
	if len(j.Steps) != 7 {
		t.Fatalf("steps = %d, want 7", len(j.Steps))
	}
	if j.BudgetName() != "create-service" {
		t.Fatalf("budget = %s", j.BudgetName())
	}
	fr, err := Find(dir, "create-service-frictionful")
	if err != nil {
		t.Fatal(err)
	}
	if fr.BudgetName() != "create-service" {
		t.Fatalf("frictionful budget = %s", fr.BudgetName())
	}
	if fr.Steps[6].MaxRetries != 1 {
		t.Fatalf("validate-deploy max_retries = %d, want 1", fr.Steps[6].MaxRetries)
	}
}

func TestValidateRejectsEmpty(t *testing.T) {
	err := Journey{}.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
}

func exampleDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "examples", "create-service")
}
