package demo

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed template/*
var templateFS embed.FS

// Render executes the embedded HTML report template against the given trace.
func Render(trace DemoTrace) ([]byte, error) {
	tmplBytes, err := templateFS.ReadFile("template/report.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	cssBytes, err := templateFS.ReadFile("template/style.css")
	if err != nil {
		return nil, fmt.Errorf("read style.css: %w", err)
	}

	jsBytes, err := templateFS.ReadFile("template/inline.js")
	if err != nil {
		return nil, fmt.Errorf("read inline.js: %w", err)
	}

	funcMap := template.FuncMap{
		"shortJKT": func(jkt string) string {
			if len(jkt) <= 8 {
				return jkt
			}
			return jkt[:4] + "..." + jkt[len(jkt)-4:]
		},
		"last4": func(s string) string {
			if len(s) <= 4 {
				return s
			}
			return "..." + s[len(s)-4:]
		},
		"inc": func(i int) int {
			return i + 1
		},
		"css": func() template.CSS {
			return template.CSS(cssBytes)
		},
		"js": func() template.JS {
			return template.JS(jsBytes)
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, trace); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}
