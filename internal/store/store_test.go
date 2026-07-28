package store

import (
	"path/filepath"
	"testing"

	"github.com/pythonsaber2/dockly/internal/core"
)

func TestStorePersistsApplicationWithoutExposingSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dockly.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	app := core.Application{Name: "hello", Repository: "https://github.com/example/hello.git", Branch: "main", Port: 8080, Env: map[string]string{"SECRET": "value"}}
	created, err := s.CreateApplication(app)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected generated ID")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Application(created.ID)
	if !ok {
		t.Fatal("application was not persisted")
	}
	if got.Env["SECRET"] != "value" {
		t.Fatal("environment was not persisted")
	}

	public := got.Public()
	if public.Env != nil {
		t.Fatal("public application must not expose environment values")
	}
	if len(public.EnvKeys) != 1 || public.EnvKeys[0] != "SECRET" {
		t.Fatalf("expected env key metadata, got %#v", public.EnvKeys)
	}
}
