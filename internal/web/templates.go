// Package web implements the HTTP server: routing, handlers, and
// server-rendered HTML templates for Date Chooser.
package web

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// pageTemplates holds one composed *template.Template per page, each built
// from the shared layout.html plus that page's content definition. They are
// parsed separately (rather than as one glob) because every page defines a
// template named "content"; parsing them into a single set would let the
// last-parsed page silently win for every other page.
type pageTemplates struct {
	create *template.Template
	links  *template.Template
}

// parseTemplates parses every embedded template once at server startup.
func parseTemplates() (*pageTemplates, error) {
	create, err := template.ParseFS(templateFS, "templates/layout.html", "templates/create.html")
	if err != nil {
		return nil, err
	}
	links, err := template.ParseFS(templateFS, "templates/layout.html", "templates/links.html")
	if err != nil {
		return nil, err
	}
	return &pageTemplates{create: create, links: links}, nil
}
