package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

func TestCreateServiceExample(t *testing.T) {
	root := repoRoot(t)
	bin := t.TempDir()
	build(t, root, bin, "frictionctl", "./cmd/frictionctl")
	build(t, root, bin, "demo-platform", "./cmd/demo-platform")
	build(t, root, bin, "demo-kube", "./cmd/demo-kube")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	dir := filepath.Join(root, "examples", "create-service")
	paved := filepath.Join(t.TempDir(), "paved.json")
	friction := filepath.Join(t.TempDir(), "frictionful.json")

	var stdout, stderr bytes.Buffer
	code := ExecuteWith([]string{"run", "create-service", "--dir", dir, "--output", paved}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("paved run exit %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("PASS")) {
		t.Fatalf("paved output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = ExecuteWith([]string{"run", "create-service-frictionful", "--dir", dir, "--output", friction}, &stdout, &stderr)
	if code != ExitSLO {
		t.Fatalf("frictionful run exit %d, want %d\nstdout=%s\nstderr=%s", code, ExitSLO, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("FAIL")) {
		t.Fatalf("frictionful output:\n%s", stdout.String())
	}

	pavedRes := loadResult(t, paved)
	frictionRes := loadResult(t, friction)
	if pavedRes.Contract != "create-service" || frictionRes.Contract != "create-service" {
		t.Fatalf("contracts paved=%q frictionful=%q", pavedRes.Contract, frictionRes.Contract)
	}
	if pavedRes.Objective != "running-service" || frictionRes.Objective != "running-service" {
		t.Fatalf("objectives paved=%q frictionful=%q", pavedRes.Objective, frictionRes.Objective)
	}
	if pavedRes.Signals.ToolTransitions != 0 {
		t.Fatalf("paved tool_transitions = %d, want 0", pavedRes.Signals.ToolTransitions)
	}
	if frictionRes.Signals.ToolTransitions != 2 {
		t.Fatalf("frictionful tool_transitions = %d, want 2", frictionRes.Signals.ToolTransitions)
	}

	stdout.Reset()
	stderr.Reset()
	code = ExecuteWith([]string{"compare", paved, friction}, &stdout, &stderr)
	if code != ExitSLO {
		t.Fatalf("compare exit %d, want %d\nstdout=%s\nstderr=%s", code, ExitSLO, stdout.String(), stderr.String())
	}

	other := filepath.Join(t.TempDir(), "other.json")
	unrelated := pavedRes
	unrelated.Contract = "onboard-engineer"
	unrelated.Journey = "onboard-engineer"
	writeResult(t, other, unrelated)
	stdout.Reset()
	stderr.Reset()
	code = ExecuteWith([]string{"compare", paved, other}, &stdout, &stderr)
	if code != ExitErr {
		t.Fatalf("unrelated compare exit %d, want %d\nstdout=%s\nstderr=%s", code, ExitErr, stdout.String(), stderr.String())
	}

	noisy := filepath.Join(t.TempDir(), "noisy.json")
	slower := pavedRes
	slower.Signals.TotalTimeMS += 5000
	slower.Signals.WaitingMS += 5000
	slower.Score.Value += 5
	writeResult(t, noisy, slower)
	stdout.Reset()
	stderr.Reset()
	code = ExecuteWith([]string{"compare", paved, noisy}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("timing-only compare exit %d, want %d\nstdout=%s\nstderr=%s", code, ExitOK, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = ExecuteWith([]string{"list", "--dir", dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("list exit %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("create-service")) {
		t.Fatalf("list output:\n%s", stdout.String())
	}

	stdout.Reset()
	code = ExecuteWith([]string{"explain", "create-service", "--dir", dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("explain exit %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("human_actions")) {
		t.Fatalf("explain output:\n%s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("tool_transitions:       0")) {
		t.Fatalf("explain should report 0 tool transitions for paved path:\n%s", stdout.String())
	}
}

func loadResult(t *testing.T, path string) result.Result {
	t.Helper()
	r, err := result.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func writeResult(t *testing.T, path string, r result.Result) {
	t.Helper()
	if err := result.WriteFile(path, r); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func build(t *testing.T, root, bin, name, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", filepath.Join(bin, name), pkg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
}
