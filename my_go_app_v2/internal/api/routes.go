package api

import (
	"net/http"

	"project/internal/storage"
)

// SetupRoutes настраивает маршруты API
func SetupRoutes(dbManager *storage.DBManager) {
	apiHandler := NewAPIHandler(dbManager)

	// Обрабатываем только запросы к API, начинающиеся с названия базы данных
	http.HandleFunc("/products_db/", apiHandler.ServeHTTP)
	http.HandleFunc("/suppliers_db/", apiHandler.ServeHTTP)
	http.HandleFunc("/inventory_db/", apiHandler.ServeHTTP)
}
