package ui

import (
	"embed"
	"html/template"
	"io"
	"net/http"
)

//go:embed static
var Assets embed.FS

//go:embed views
var views embed.FS
var templates = template.Must(template.ParseFS(views, "views/*.html"))

func Render(w http.ResponseWriter, name string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return templates.ExecuteTemplate(w, name, nil)
}
func RenderTo(w io.Writer, name string) error { return templates.ExecuteTemplate(w, name, nil) }
