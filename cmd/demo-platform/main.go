package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "scaffold":
		err = scaffold(args)
	case "validate-project":
		err = validateProject(args)
	case "build":
		err = build(args)
	case "generate-deploy":
		err = generateDeploy(args)
	case "inspect-deploy":
		err = inspectDeploy(args)
	case "validate-deploy":
		err = validateDeploy(args)
	case "deploy":
		err = deploy(args)
	case "verify":
		err = verify(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "demo-platform scaffold|validate-project|build|generate-deploy|inspect-deploy|validate-deploy|deploy|verify")
}

func frictionful() bool {
	return os.Getenv("FRICTION_DEMO_MODE") == "frictionful"
}

func scaffold(args []string) error {
	fs := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	name := fs.String("name", "", "service name")
	out := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *out == "" {
		return fmt.Errorf("scaffold requires --name and --out")
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	mod := `module example.com/` + *name + `

go 1.22
`
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("hello from ` + *name + `")
}
`
	if err := os.WriteFile(filepath.Join(*out, "go.mod"), []byte(mod), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "main.go"), []byte(mainGo), 0o644)
}

func validateProject(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	for _, f := range []string{"go.mod", "main.go"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return fmt.Errorf("validate-project: missing %s", f)
		}
	}
	return nil
}

func build(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", "app", ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func generateDeploy(args []string) error {
	fs := flag.NewFlagSet("generate-deploy", flag.ContinueOnError)
	dir := fs.String("dir", "", "service directory")
	ns := fs.String("namespace", "", "kubernetes namespace")
	cpu := fs.String("cpu-limit", "", "cpu limit")
	ing := fs.String("ingress-class", "", "ingress class")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("generate-deploy requires --dir")
	}
	if frictionful() {
		missing := make([]string, 0, 3)
		if *ns == "" {
			missing = append(missing, "--namespace")
		}
		if *cpu == "" {
			missing = append(missing, "--cpu-limit")
		}
		if *ing == "" {
			missing = append(missing, "--ingress-class")
		}
		if len(missing) > 0 {
			return fmt.Errorf("frictionful mode requires %s", strings.Join(missing, ", "))
		}
	} else {
		if *ns == "" {
			*ns = "default"
		}
		if *cpu == "" {
			*cpu = "100m"
		}
		if *ing == "" {
			*ing = "nginx"
		}
	}
	body := fmt.Sprintf("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app\n  namespace: %s\nspec:\n  replicas: 1\n  template:\n    spec:\n      containers:\n        - name: app\n          resources:\n            limits:\n              cpu: %s\n---\napiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: app\n  annotations:\n    kubernetes.io/ingress.class: %s\n", *ns, *cpu, *ing)
	return os.WriteFile(filepath.Join(*dir, "deploy.yaml"), []byte(body), 0o644)
}

func inspectDeploy(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "deploy.yaml"))
	if err != nil {
		return err
	}
	fmt.Print(string(raw))
	return nil
}

func validateDeploy(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "deploy.yaml")); err != nil {
		return fmt.Errorf("validate-deploy: missing deploy.yaml")
	}
	if frictionful() {
		marker := filepath.Join(dir, ".admission-ready")
		if _, err := os.Stat(marker); err != nil {
			_ = os.WriteFile(marker, []byte("1"), 0o644)
			return fmt.Errorf("admission webhook not ready")
		}
	}
	return nil
}

func deploy(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "deploy.yaml")); err != nil {
		return fmt.Errorf("deploy: missing deploy.yaml")
	}
	if frictionful() {
		if _, err := os.Stat(filepath.Join(dir, "namespace.txt")); err != nil {
			return fmt.Errorf("deploy: namespace not set; run demo-kube set-namespace")
		}
	}
	return os.WriteFile(filepath.Join(dir, "deployed"), []byte("ok\n"), 0o644)
}

func verify(args []string) error {
	dir, err := dirFlag(args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "deployed")); err != nil {
		return fmt.Errorf("verify: service is not deployed")
	}
	return nil
}

func dirFlag(args []string) (string, error) {
	fs := flag.NewFlagSet("dir", flag.ContinueOnError)
	dir := fs.String("dir", "", "service directory")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if *dir == "" {
		return "", fmt.Errorf("--dir is required")
	}
	return *dir, nil
}
