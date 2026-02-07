# forge-template

Template engine reutilizavel para Go. Renderiza pages e partials com layouts, components e funcoes utilitarias.

```
go get github.com/devalexandre/forge-template
```

## Quick Start

```go
import forgetemplate "github.com/devalexandre/forge-template"

engine, err := forgetemplate.New(forgetemplate.Config{
    Dir: "web/templates",
})
```

## Estrutura de Diretorios

O engine espera esta estrutura:

```
web/templates/
├── layouts/      # layouts compartilhados (ex: base.html com {{define "base"}})
├── components/   # componentes reutilizaveis
├── partials/     # partials (HTMX responses)
└── pages/        # paginas (suporta subdiretorios: admin/, errors/)
```

## Renderizando Pages

Pages sao renderizadas dentro do layout `{{define "base"}}`:

```go
func homeHandler(w http.ResponseWriter, r *http.Request) {
    engine.RenderPage(w, r, "home.html", map[string]any{
        "Title": "Home",
    })
}
```

Subdiretorios sao suportados:

```go
engine.RenderPage(w, r, "admin/dashboard.html", data)
engine.RenderPage(w, r, "errors/404.html", data)
```

## Renderizando Partials

Partials sao renderizados standalone (ideal para respostas HTMX):

```go
func partialPosts(w http.ResponseWriter, r *http.Request) {
    engine.RenderPartial(w, r, "post-list.html", map[string]any{
        "Items": posts,
    })
}
```

## BeforeRender Hook

Injeta dados em todas as renderizacoes (CSRF, user, flash messages, etc.):

```go
import forgecsrf "github.com/devalexandre/forge-http/middleware/csrf"

engine, _ := forgetemplate.New(forgetemplate.Config{
    Dir: "web/templates",
    BeforeRender: func(r *http.Request, data map[string]any) {
        forgecsrf.InjectTemplateData(r, data)
        data["CurrentUser"] = getUserFromSession(r)
    },
})
```

## FuncMap Customizado

Funcoes extras fazem merge com os defaults:

```go
engine, _ := forgetemplate.New(forgetemplate.Config{
    Dir: "web/templates",
    FuncMap: template.FuncMap{
        "uppercase": strings.ToUpper,
    },
})
```

### Funcoes Default

| Funcao | Assinatura | Descricao |
|---|---|---|
| `formatDate` | `(*time.Time) string` | Formata data "Jan 02, 2006" |
| `formatDateTime` | `(time.Time) string` | Formata data+hora "Jan 02, 2006 15:04" |
| `truncate` | `(string, int) string` | Trunca string com "..." |
| `safeHTML` | `(string) template.HTML` | Retorna HTML sem escape |
| `currentYear` | `() int` | Ano atual |
| `add` | `(int, int) int` | Soma inteiros |
| `seq` | `(int) []int` | Gera sequencia [1..n] |
| `eq` | `(string, string) bool` | Compara strings |

## Exemplo Completo

```go
package main

import (
    "net/http"

    forgetemplate "github.com/devalexandre/forge-template"
    forgecsrf "github.com/devalexandre/forge-http/middleware/csrf"
    "github.com/go-chi/chi/v5"
)

func main() {
    engine, err := forgetemplate.New(forgetemplate.Config{
        Dir: "web/templates",
        BeforeRender: func(r *http.Request, data map[string]any) {
            forgecsrf.InjectTemplateData(r, data)
        },
    })
    if err != nil {
        panic(err)
    }

    r := chi.NewRouter()

    r.Use(forgecsrf.New(forgecsrf.Config{
        Key:            []byte("32-byte-auth-key-here-change-me!"),
        Secure:         false,
        AllowPlainHTTP: true,
    }))

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        engine.RenderPage(w, r, "home.html", map[string]any{
            "Title": "Home",
        })
    })

    http.ListenAndServe(":8080", r)
}
```
