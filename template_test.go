package template

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestTemplates(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	dirs := []string{"layouts", "partials", "components", "pages", "pages/admin", "pages/errors"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0o755)
	}

	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("layouts/base.html", `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`)
	write("partials/header.html", `{{define "header"}}<header>{{.Title}}</header>{{end}}`)
	write("components/btn.html", `{{define "btn"}}<button>{{.}}</button>{{end}}`)
	write("pages/home.html", `{{define "content"}}<h1>{{.Title}}</h1>{{template "header" .}}{{end}}`)
	write("pages/admin/dashboard.html", `{{define "content"}}<h1>Admin: {{.Title}}</h1>{{end}}`)
	write("pages/errors/404.html", `{{define "content"}}<h1>Not Found</h1>{{end}}`)
	write("partials/post-list.html", `{{define "post-list"}}<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>{{end}}`)

	return dir
}

func TestNew_RequiresDir(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for empty Dir")
	}
}

func TestNew_ParsesTemplates(t *testing.T) {
	dir := setupTestTemplates(t)
	e, err := New(Config{Dir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	expected := []string{"home.html", "admin/dashboard.html", "errors/404.html", "partials/header.html", "partials/post-list.html"}
	for _, name := range expected {
		if _, ok := e.templates[name]; !ok {
			t.Errorf("template %q not found", name)
		}
	}
}

func TestRenderPage(t *testing.T) {
	dir := setupTestTemplates(t)
	e, err := New(Config{Dir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	e.RenderPage(w, r, "home.html", map[string]any{"Title": "Hello"})

	body := w.Body.String()
	if !strings.Contains(body, "<h1>Hello</h1>") {
		t.Errorf("expected <h1>Hello</h1>, got: %s", body)
	}
	if !strings.Contains(body, "<header>Hello</header>") {
		t.Errorf("expected header partial, got: %s", body)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestRenderPage_AdminSubdir(t *testing.T) {
	dir := setupTestTemplates(t)
	e, _ := New(Config{Dir: dir})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/admin", nil)
	e.RenderPage(w, r, "admin/dashboard.html", map[string]any{"Title": "Dash"})

	if !strings.Contains(w.Body.String(), "Admin: Dash") {
		t.Errorf("expected admin content, got: %s", w.Body.String())
	}
}

func TestRenderPage_NotFound(t *testing.T) {
	dir := setupTestTemplates(t)
	e, _ := New(Config{Dir: dir})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	e.RenderPage(w, r, "nope.html", map[string]any{})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRenderPartial(t *testing.T) {
	dir := setupTestTemplates(t)
	e, _ := New(Config{Dir: dir})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/partials/post-list", nil)
	e.RenderPartial(w, r, "post-list.html", map[string]any{"Items": []string{"A", "B"}})

	body := w.Body.String()
	if !strings.Contains(body, "<li>A</li>") || !strings.Contains(body, "<li>B</li>") {
		t.Errorf("expected list items, got: %s", body)
	}
}

func TestRenderPartial_NotFound(t *testing.T) {
	dir := setupTestTemplates(t)
	e, _ := New(Config{Dir: dir})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	e.RenderPartial(w, r, "nope.html", map[string]any{})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestBeforeRender(t *testing.T) {
	dir := setupTestTemplates(t)
	called := false
	e, _ := New(Config{
		Dir: dir,
		BeforeRender: func(r *http.Request, data map[string]any) {
			called = true
			data["Title"] = "Injected"
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	e.RenderPage(w, r, "home.html", map[string]any{})

	if !called {
		t.Error("BeforeRender not called")
	}
	if !strings.Contains(w.Body.String(), "Injected") {
		t.Errorf("expected injected title, got: %s", w.Body.String())
	}
}

func TestCustomFuncMap(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pages"), 0o755)

	os.WriteFile(filepath.Join(dir, "layouts", "base.html"),
		[]byte(`{{define "base"}}{{template "content" .}}{{end}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "pages", "test.html"),
		[]byte(`{{define "content"}}{{greet .Name}}{{end}}`), 0o644)

	e, err := New(Config{
		Dir: dir,
		FuncMap: map[string]any{
			"greet": func(name string) string { return "Hi, " + name + "!" },
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	e.RenderPage(w, r, "test.html", map[string]any{"Name": "Go"})

	if !strings.Contains(w.Body.String(), "Hi, Go!") {
		t.Errorf("expected custom func output, got: %s", w.Body.String())
	}
}

func TestDefaultFuncMap_FormatDate(t *testing.T) {
	fm := defaultFuncMap()
	fn := fm["formatDate"].(func(*time.Time) string)

	if fn(nil) != "" {
		t.Error("expected empty for nil")
	}

	ts := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	if got := fn(&ts); got != "Mar 15, 2025" {
		t.Errorf("formatDate = %q", got)
	}
}

func TestDefaultFuncMap_Truncate(t *testing.T) {
	fm := defaultFuncMap()
	fn := fm["truncate"].(func(string, int) string)

	if got := fn("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	if got := fn("hello world", 5); got != "hello..." {
		t.Errorf("truncate long = %q", got)
	}
}

func TestDefaultFuncMap_Seq(t *testing.T) {
	fm := defaultFuncMap()
	fn := fm["seq"].(func(int) []int)

	got := fn(3)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("seq(3) = %v", got)
	}
}

func TestDefaultFuncMap_Add(t *testing.T) {
	fm := defaultFuncMap()
	fn := fm["add"].(func(int, int) int)

	if got := fn(2, 3); got != 5 {
		t.Errorf("add(2,3) = %d", got)
	}
}
