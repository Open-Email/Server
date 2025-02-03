package main

import (
	"email.mercata.com/ui"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
)

type templateData struct {
	RandomNonce string
	CurrentYear int
	CurrentPath string
	CSRFToken   string
}

var functions = template.FuncMap{
	"startsWith": strings.HasPrefix,
	"contains":   strings.Contains,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
