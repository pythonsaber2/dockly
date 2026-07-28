package deploy

import (
	"context"
	"strings"
	"testing"

	"github.com/pythonsaber2/dockly/internal/core"
)

type fakeRunner struct{ calls []string }

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if name == "git" && strings.Contains(strings.Join(args, " "), "rev-parse HEAD") {
		return "abc123\n", nil
	}
	if name == "docker" && len(args) > 1 && args[0] == "inspect" {
		return "old:image\n", nil
	}
	return "ok", nil
}

type healthy struct{}

func (healthy) Check(context.Context, string) error { return nil }

type recordingHealth struct{ url string }

func (h *recordingHealth) Check(_ context.Context, url string) error {
	h.url = url
	return nil
}

func TestDeployBuildsAndStartsContainerWithoutSecretsInArguments(t *testing.T) {
	r := &fakeRunner{}
	e := NewEngine(t.TempDir(), r, healthy{})
	app := core.Application{ID: "app1", Name: "hello", Repository: "https://example.com/hello.git", Branch: "main", Dockerfile: "Dockerfile", Context: ".", Port: 8080, PublishPort: 9090, HealthPath: "/health", Env: map[string]string{"SECRET": "do-not-leak"}}
	dep := core.Deployment{ID: "dep1", AppID: app.ID, Trigger: "manual"}
	got, err := e.Deploy(context.Background(), app, dep)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "successful" || got.Commit != "abc123" {
		t.Fatalf("unexpected deployment: %#v", got)
	}
	joined := strings.Join(r.calls, "\n")
	for _, want := range []string{"git clone", "docker build", "docker run"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in calls:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "do-not-leak") {
		t.Fatal("secret leaked into process arguments")
	}
}

func TestSafeJoinRejectsPathsOutsideRepository(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, "../Dockerfile"); err == nil {
		t.Fatal("expected parent traversal to be rejected")
	}
	got, err := safeJoin(root, "services/api/Dockerfile")
	if err != nil || !strings.HasPrefix(got, root) {
		t.Fatalf("expected safe path, got %q, %v", got, err)
	}
}

func TestDeployUsesConfiguredHealthHost(t *testing.T) {
	r := &fakeRunner{}
	h := &recordingHealth{}
	e := NewEngine(t.TempDir(), r, h, "host.docker.internal")
	app := core.Application{ID: "app1", Name: "hello", Repository: "https://example.com/hello.git", Branch: "main", Dockerfile: "Dockerfile", Context: ".", Port: 8080, PublishPort: 9090, HealthPath: "health"}
	_, err := e.Deploy(context.Background(), app, core.Deployment{ID: "dep1", AppID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if h.url != "http://host.docker.internal:9090/health" {
		t.Fatalf("unexpected health URL %q", h.url)
	}
}
