package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"project/internal/models"
	"project/internal/storage"
)

// APIHandler обрабатывает API запросы
type APIHandler struct {
	dbManager *storage.DBManager
}

// NewAPIHandler создает новый обработчик API
func NewAPIHandler(dbManager *storage.DBManager) *APIHandler {
	return &APIHandler{dbManager: dbManager}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Парсим путь запроса
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Неверный путь запроса", http.StatusBadRequest)
		return
	}

	dbName := pathParts[0]
	resource := pathParts[1]

	// Проверяем, какая база данных запрошена
	if !h.validateDBName(dbName) {
		http.Error(w, "База данных не найдена", http.StatusNotFound)
		return
	}

	// Обрабатываем запрос в зависимости от метода HTTP
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, dbName, resource, pathParts)
	case http.MethodPost:
		h.handlePost(w, r, dbName, resource)
	case http.MethodPut:
		h.handlePut(w, r, dbName, resource, pathParts)
	case http.MethodDelete:
		h.handleDelete(w, r, dbName, resource, pathParts)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

// validateDBName проверяет валидность имени базы данных
func (h *APIHandler) validateDBName(dbName string) bool {
	return dbName == "products_db" || dbName == "suppliers_db" || dbName == "inventory_db"
}

// handleGet обрабатывает GET запросы
func (h *APIHandler) handleGet(w http.ResponseWriter, r *http.Request, dbName, resource string, pathParts []string) {
	if resource != "products" {
		http.Error(w, "Ресурс не найден", http.StatusNotFound)
		return
	}

	// Если в пути есть идентификатор - возвращаем один товар
	if len(pathParts) > 2 {
		id, err := strconv.Atoi(pathParts[2])
		if err != nil {
			http.Error(w, "Неверный формат идентификатора", http.StatusBadRequest)
			return
		}

		var product models.Product
		var exists bool
		var getErr error

		// Выбираем соответствующую БД
		switch dbName {
		case "products_db":
			product, exists, getErr = h.dbManager.MongoDB.GetProduct(id)
		case "suppliers_db":
			product, exists, getErr = h.dbManager.PostgresDB.GetProduct(id)
		case "inventory_db":
			product, exists, getErr = h.dbManager.MySQLDB.GetProduct(id)
		}

		if getErr != nil {
			http.Error(w, "Ошибка при получении товара: "+getErr.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "Товар не найден", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(product)
		return
	}

	// Возвращаем все товары
	var products []models.Product
	var err error

	// Выбираем соответствующую БД
	switch dbName {
	case "products_db":
		products, err = h.dbManager.MongoDB.GetAllProducts()
	case "suppliers_db":
		products, err = h.dbManager.PostgresDB.GetAllProducts()
	case "inventory_db":
		products, err = h.dbManager.MySQLDB.GetAllProducts()
	}

	if err != nil {
		http.Error(w, "Ошибка при получении товаров: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(products)
}

// handlePost обрабатывает POST запросы (создание нового товара)
func (h *APIHandler) handlePost(w http.ResponseWriter, r *http.Request, dbName, resource string) {
	if resource != "products" {
		http.Error(w, "Ресурс не найден", http.StatusNotFound)
		return
	}

	var product models.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "Ошибка чтения данных: "+err.Error(), http.StatusBadRequest)
		return
	}

	var err error

	// Добавляем товар в соответствующую БД
	switch dbName {
	case "products_db":
		err = h.dbManager.MongoDB.AddProduct(product)
	case "suppliers_db":
		err = h.dbManager.PostgresDB.AddProduct(product)
	case "inventory_db":
		err = h.dbManager.MySQLDB.AddProduct(product)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Товар с ID %d успешно создан в базе %s", product.ID, dbName),
	})
}

// handlePut обрабатывает PUT запросы (обновление товара)
func (h *APIHandler) handlePut(w http.ResponseWriter, r *http.Request, dbName, resource string, pathParts []string) {
	if resource != "products" || len(pathParts) <= 2 {
		http.Error(w, "Неверный путь запроса", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.Error(w, "Неверный формат идентификатора", http.StatusBadRequest)
		return
	}

	var product models.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "Ошибка чтения данных: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Удостоверимся, что ID в пути и в теле запроса совпадают
	if product.ID != id {
		http.Error(w, "ID в пути и в теле запроса не совпадают", http.StatusBadRequest)
		return
	}

	// Обновляем товар в соответствующей БД
	switch dbName {
	case "products_db":
		err = h.dbManager.MongoDB.UpdateProduct(product)
	case "suppliers_db":
		err = h.dbManager.PostgresDB.UpdateProduct(product)
	case "inventory_db":
		err = h.dbManager.MySQLDB.UpdateProduct(product)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Товар с ID %d успешно обновлен в базе %s", id, dbName),
	})
}

// handleDelete обрабатывает DELETE запросы (удаление товара)
func (h *APIHandler) handleDelete(w http.ResponseWriter, r *http.Request, dbName, resource string, pathParts []string) {
	if resource != "products" || len(pathParts) <= 2 {
		http.Error(w, "Неверный путь запроса", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.Error(w, "Неверный формат идентификатора", http.StatusBadRequest)
		return
	}

	// Удаляем товар из соответствующей БД
	switch dbName {
	case "products_db":
		err = h.dbManager.MongoDB.DeleteProduct(id)
	case "suppliers_db":
		err = h.dbManager.PostgresDB.DeleteProduct(id)
	case "inventory_db":
		err = h.dbManager.MySQLDB.DeleteProduct(id)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Товар с ID %d успешно удален из базы %s", id, dbName),
	})
}
