package web

import (
	"log/slog"
	"net/http"
)

func (wb *Web) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := wb.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("rendering template", "template", name, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
