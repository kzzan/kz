package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kzzan/kz/pkg/generator"
)

func TestInitCurrentDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myapp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	t.Chdir(root)

	cmd := NewRootCmd()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"init", "."})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in current directory: %v", err)
	}

	nestedPath := filepath.Join(root, filepath.Base(root), "go.mod")
	if _, err := os.Stat(nestedPath); !os.IsNotExist(err) {
		t.Fatalf("expected no nested project directory, stat err=%v", err)
	}
}

func TestInitRefusesNonEmptyDirectoryWithoutForce(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myapp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("write keep.txt: %v", err)
	}
	t.Chdir(root)

	cmd := NewRootCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"init", "."})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected init to fail for non-empty directory")
	}
	if exitCode(err) != 2 {
		t.Fatalf("expected usage exit code 2, got %d", exitCode(err))
	}
	if !strings.Contains(err.Error(), "is not empty") {
		t.Fatalf("expected non-empty directory error, got %v", err)
	}
}

func TestGenerateFromNestedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myapp")
	if err := generator.NewProjectGenerator("myapp", root).GenerateProject(); err != nil {
		t.Fatalf("generate project: %v", err)
	}

	t.Chdir(filepath.Join(root, "internal", "service"))

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "model", "order"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate model: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "models", "order.go")); err != nil {
		t.Fatalf("expected generated model in project root: %v", err)
	}
}

func TestLegacyNewComponentAlias(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myapp")
	if err := generator.NewProjectGenerator("myapp", root).GenerateProject(); err != nil {
		t.Fatalf("generate project: %v", err)
	}
	t.Chdir(root)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"new", "service", "order"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute legacy new service: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "service", "order.go")); err != nil {
		t.Fatalf("expected generated service: %v", err)
	}
}

func TestVersionWritesStdout(t *testing.T) {
	cmd := NewRootCmd()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("expected version %q, got %q", version, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}
