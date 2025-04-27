package api

import (
	"net/http"
	"path/filepath"
)

// SetupStaticFiles настраивает обработку статических файлов
func SetupStaticFiles() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Перенаправление корневого пути на index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join("static", "index.html"))
			return
		}
		// Все остальные пути будут обрабатываться API обработчиком
	})
}
