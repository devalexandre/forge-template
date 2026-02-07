package template

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

type Config struct {
	Dir          string
	FuncMap      template.FuncMap
	BeforeRender func(r *http.Request, data map[string]any)
}

type Engine struct {
	templates    map[string]*template.Template
	beforeRender func(r *http.Request, data map[string]any)
}

func New(cfg Config) (*Engine, error) {
	if cfg.Dir == "" {
		return nil, fmt.Errorf("forge-template: Dir is required")
	}

	funcMap := defaultFuncMap()
	for k, v := range cfg.FuncMap {
		funcMap[k] = v
	}

	e := &Engine{
		templates:    make(map[string]*template.Template),
		beforeRender: cfg.BeforeRender,
	}

	layouts, _ := filepath.Glob(filepath.Join(cfg.Dir, "layouts", "*.html"))
	partials, _ := filepath.Glob(filepath.Join(cfg.Dir, "partials", "*.html"))
	components, _ := filepath.Glob(filepath.Join(cfg.Dir, "components", "*.html"))

	shared := make([]string, 0, len(layouts)+len(partials)+len(components))
	shared = append(shared, layouts...)
	shared = append(shared, partials...)
	shared = append(shared, components...)

	pages, _ := filepath.Glob(filepath.Join(cfg.Dir, "pages", "*.html"))
	adminPages, _ := filepath.Glob(filepath.Join(cfg.Dir, "pages", "admin", "*.html"))
	errorPages, _ := filepath.Glob(filepath.Join(cfg.Dir, "pages", "errors", "*.html"))

	allPages := make([]string, 0, len(pages)+len(adminPages)+len(errorPages))
	allPages = append(allPages, pages...)
	allPages = append(allPages, adminPages...)
	allPages = append(allPages, errorPages...)

	pagesDir := filepath.Join(cfg.Dir, "pages") + "/"

	for _, page := range allPages {
		name := strings.TrimPrefix(page, pagesDir)
		files := make([]string, 0, len(shared)+1)
		files = append(files, shared...)
		files = append(files, page)

		tmpl, err := template.New(filepath.Base(page)).Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
		e.templates[name] = tmpl
	}

	for _, partial := range partials {
		name := "partials/" + filepath.Base(partial)
		tmpl, err := template.New(filepath.Base(partial)).Funcs(funcMap).ParseFiles(partial)
		if err != nil {
			return nil, fmt.Errorf("parsing partial %s: %w", name, err)
		}
		e.templates[name] = tmpl
	}

	return e, nil
}

func (e *Engine) RenderPage(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	tmpl, ok := e.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}

	if e.beforeRender != nil {
		e.beforeRender(r, data)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (e *Engine) RenderPartial(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	key := "partials/" + name
	tmpl, ok := e.templates[key]
	if !ok {
		http.Error(w, fmt.Sprintf("partial %q not found", name), http.StatusInternalServerError)
		return
	}

	if e.beforeRender != nil {
		e.beforeRender(r, data)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	baseName := strings.TrimSuffix(filepath.Base(name), ".html")
	if err := tmpl.ExecuteTemplate(w, baseName, data); err != nil {
		tmpl.Execute(w, data)
	}
}
