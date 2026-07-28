package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/pythonsaber2/dockly/internal/store"
)

func TestSetupCreatesAdminSessionAndProtectsApplications(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := New(s, nil, Config{SecureCookies: false})

	unauth := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	unauthRec := httptest.NewRecorder()
	h.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthRec.Code)
	}

	body := bytes.NewBufferString(`{"password":"correct horse battery staple"}`)
	setup := httptest.NewRequest(http.MethodPost, "/api/setup", body)
	setup.Header.Set("Content-Type", "application/json")
	setupRec := httptest.NewRecorder()
	h.ServeHTTP(setupRec, setup)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", setupRec.Code, setupRec.Body.String())
	}
	cookie := setupRec.Result().Cookies()[0]

	appBody := bytes.NewBufferString(`{"name":"hello","repository":"https://example.com/hello.git","branch":"main","port":8080,"publishPort":9000}`)
	create := httptest.NewRequest(http.MethodPost, "/api/apps", appBody)
	create.Header.Set("Content-Type", "application/json")
	create.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	app := response["application"].(map[string]any)
	if app["name"] != "hello" {
		t.Fatalf("unexpected app: %#v", app)
	}
}

func TestHealthReportsDeploymentEngineReadiness(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := New(s, nil, Config{EngineReady: false, Version: "test"})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["engineReady"] != false || response["version"] != "test" {
		t.Fatalf("unexpected health response: %#v", response)
	}
}
