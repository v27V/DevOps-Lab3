package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"project/internal/models"

	// Драйвера для баз данных
	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DBManager управляет подключениями к различным базам данных
type DBManager struct {
	MongoDB    *MongoDBClient
	PostgresDB *PostgresClient
	MySQLDB    *MySQLClient
}

// MongoDBClient клиент для MongoDB (products_db)
type MongoDBClient struct {
	Client     *mongo.Client
	Database   *mongo.Database
	Collection *mongo.Collection
}

// PostgresClient клиент для PostgreSQL (suppliers_db)
type PostgresClient struct {
	DB *sql.DB
}

// MySQLClient клиент для MySQL (inventory_db)
type MySQLClient struct {
	DB *sql.DB
}

// NewDBManager создает и инициализирует менеджер баз данных
func NewDBManager() (*DBManager, error) {
	// Инициализация MongoDB
	mongoClient, err := initMongoDB()
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации MongoDB: %v", err)
	}

	// Инициализация PostgreSQL
	postgresClient, err := initPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации PostgreSQL: %v", err)
	}

	// Инициализация MySQL
	mysqlClient, err := initMySQLDB()
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации MySQL: %v", err)
	}

	manager := &DBManager{
		MongoDB:    mongoClient,
		PostgresDB: postgresClient,
		MySQLDB:    mysqlClient,
	}

	return manager, nil
}

// Инициализация MongoDB
func initMongoDB() (*MongoDBClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Измените URI на ваш актуальный адрес MongoDB
	clientOptions := options.Client().ApplyURI("mongodb://user:qwerty123@mongo_db:27017/?authSource=products_db&authMechanism=SCRAM-SHA-256")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	database := client.Database("products_db")
	collection := database.Collection("products")

	// Создаем индекс по полю ID для быстрого поиска
	_, err = collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		return nil, err
	}

	return &MongoDBClient{
		Client:     client,
		Database:   database,
		Collection: collection,
	}, nil
}

// Инициализация PostgreSQL
func initPostgresDB() (*PostgresClient, error) {
	// Данные для подключения
	connStr := "host=postgres_db port=5432 user=user password=P@ssw0rd dbname=suppliers_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Создание таблицы products, если она не существует
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			category VARCHAR(50) NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			description TEXT,
			in_stock BOOLEAN NOT NULL DEFAULT FALSE,
			supplier VARCHAR(100) NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return &PostgresClient{DB: db}, nil
}

// Инициализация MySQL
func initMySQLDB() (*MySQLClient, error) {
	// Настройка конфигурации MySQL
	cfg := mysql.Config{
		User:                 "user",
		Passwd:               "P@ssw0rd",
		Net:                  "tcp",
		Addr:                 "mysql_db:3306",
		DBName:               "inventory_db",
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	// Открытие соединения
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Создание таблицы products, если она не существует
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			category VARCHAR(50) NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			description TEXT,
			in_stock BOOLEAN NOT NULL DEFAULT FALSE,
			supplier VARCHAR(100) NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return &MySQLClient{DB: db}, nil
}

// Закрытие всех соединений
func (m *DBManager) Close() {
	if m.MongoDB != nil && m.MongoDB.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.MongoDB.Client.Disconnect(ctx); err != nil {
			log.Printf("Ошибка закрытия соединения с MongoDB: %v", err)
		}
	}

	if m.PostgresDB != nil && m.PostgresDB.DB != nil {
		if err := m.PostgresDB.DB.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с PostgreSQL: %v", err)
		}
	}

	if m.MySQLDB != nil && m.MySQLDB.DB != nil {
		if err := m.MySQLDB.DB.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с MySQL: %v", err)
		}
	}
}

// ----- MongoDB (products_db) операции -----

// GetProduct получает продукт из MongoDB по ID
func (m *MongoDBClient) GetProduct(id int) (models.Product, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var product models.Product
	filter := bson.M{"id": id}
	err := m.Collection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Product{}, false, nil
		}
		return models.Product{}, false, err
	}

	return product, true, nil
}

// GetAllProducts получает все продукты из MongoDB
func (m *MongoDBClient) GetAllProducts() ([]models.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := m.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

// AddProduct добавляет продукт в MongoDB
func (m *MongoDBClient) AddProduct(product models.Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем, существует ли продукт с таким ID
	filter := bson.M{"id": product.ID}
	count, err := m.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("продукт с ID %d уже существует", product.ID)
	}

	_, err = m.Collection.InsertOne(ctx, product)
	return err
}

// UpdateProduct обновляет продукт в MongoDB
func (m *MongoDBClient) UpdateProduct(product models.Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": product.ID}
	update := bson.M{"$set": product}

	result, err := m.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("продукт с ID %d не найден", product.ID)
	}

	return nil
}

// DeleteProduct удаляет продукт из MongoDB
func (m *MongoDBClient) DeleteProduct(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	result, err := m.Collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("продукт с ID %d не найден", id)
	}

	return nil
}

// ----- PostgreSQL (suppliers_db) операции -----

// GetProduct получает продукт из PostgreSQL по ID
func (p *PostgresClient) GetProduct(id int) (models.Product, bool, error) {
	var product models.Product

	query := `SELECT id, name, category, price, description, in_stock, supplier 
			  FROM products WHERE id = $1`

	row := p.DB.QueryRow(query, id)
	err := row.Scan(&product.ID, &product.Name, &product.Category, &product.Price,
		&product.Description, &product.InStock, &product.Supplier)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Product{}, false, nil
		}
		return models.Product{}, false, err
	}

	return product, true, nil
}

// GetAllProducts получает все продукты из PostgreSQL
func (p *PostgresClient) GetAllProducts() ([]models.Product, error) {
	query := `SELECT id, name, category, price, description, in_stock, supplier FROM products`

	rows, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ID, &product.Name, &product.Category, &product.Price,
			&product.Description, &product.InStock, &product.Supplier)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// AddProduct добавляет продукт в PostgreSQL
func (p *PostgresClient) AddProduct(product models.Product) error {
	// Проверка существования продукта с таким ID
	var exists bool
	err := p.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", product.ID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("продукт с ID %d уже существует", product.ID)
	}

	query := `INSERT INTO products (id, name, category, price, description, in_stock, supplier) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = p.DB.Exec(query, product.ID, product.Name, product.Category, product.Price,
		product.Description, product.InStock, product.Supplier)

	return err
}

// UpdateProduct обновляет продукт в PostgreSQL
func (p *PostgresClient) UpdateProduct(product models.Product) error {
	// Проверка существования продукта
	var exists bool
	err := p.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", product.ID).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("продукт с ID %d не найден", product.ID)
	}

	query := `UPDATE products SET name = $1, category = $2, price = $3, 
			  description = $4, in_stock = $5, supplier = $6 WHERE id = $7`

	_, err = p.DB.Exec(query, product.Name, product.Category, product.Price,
		product.Description, product.InStock, product.Supplier, product.ID)

	return err
}

// DeleteProduct удаляет продукт из PostgreSQL
func (p *PostgresClient) DeleteProduct(id int) error {
	// Проверка существования продукта
	var exists bool
	err := p.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("продукт с ID %d не найден", id)
	}

	query := `DELETE FROM products WHERE id = $1`
	_, err = p.DB.Exec(query, id)

	return err
}

// ----- MySQL (inventory_db) операции -----

// GetProduct получает продукт из MySQL по ID
func (m *MySQLClient) GetProduct(id int) (models.Product, bool, error) {
	var product models.Product

	query := `SELECT id, name, category, price, description, in_stock, supplier 
			  FROM products WHERE id = ?`

	row := m.DB.QueryRow(query, id)
	err := row.Scan(&product.ID, &product.Name, &product.Category, &product.Price,
		&product.Description, &product.InStock, &product.Supplier)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Product{}, false, nil
		}
		return models.Product{}, false, err
	}

	return product, true, nil
}

// GetAllProducts получает все продукты из MySQL
func (m *MySQLClient) GetAllProducts() ([]models.Product, error) {
	query := `SELECT id, name, category, price, description, in_stock, supplier FROM products`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ID, &product.Name, &product.Category, &product.Price,
			&product.Description, &product.InStock, &product.Supplier)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// AddProduct добавляет продукт в MySQL
func (m *MySQLClient) AddProduct(product models.Product) error {
	// Проверка существования продукта с таким ID
	var count int
	err := m.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", product.ID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("продукт с ID %d уже существует", product.ID)
	}

	query := `INSERT INTO products (id, name, category, price, description, in_stock, supplier) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = m.DB.Exec(query, product.ID, product.Name, product.Category, product.Price,
		product.Description, product.InStock, product.Supplier)

	return err
}

// UpdateProduct обновляет продукт в MySQL
func (m *MySQLClient) UpdateProduct(product models.Product) error {
	// Проверка существования продукта
	var count int
	err := m.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", product.ID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("продукт с ID %d не найден", product.ID)
	}

	query := `UPDATE products SET name = ?, category = ?, price = ?, 
			  description = ?, in_stock = ?, supplier = ? WHERE id = ?`

	_, err = m.DB.Exec(query, product.Name, product.Category, product.Price,
		product.Description, product.InStock, product.Supplier, product.ID)

	return err
}

// DeleteProduct удаляет продукт из MySQL
func (m *MySQLClient) DeleteProduct(id int) error {
	// Проверка существования продукта
	var count int
	err := m.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", id).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("продукт с ID %d не найден", id)
	}

	query := `DELETE FROM products WHERE id = ?`
	_, err = m.DB.Exec(query, id)

	return err
}

// InitializeTestData заполняет базы данных тестовыми данными (если они пусты)
func (m *DBManager) InitializeTestData() error {
	// Проверка наличия данных в MongoDB
	products, err := m.MongoDB.GetAllProducts()
	if err != nil {
		return err
	}

	if len(products) == 0 {
		// Добавляем тестовые данные в MongoDB (products_db)
		err = m.MongoDB.AddProduct(models.Product{
			ID:          1,
			Name:        "Кирпич облицовочный",
			Category:    "Стеновые материалы",
			Price:       15.50,
			Description: "Кирпич керамический облицовочный",
			InStock:     true,
			Supplier:    "ООО Кирпичный завод",
		})
		if err != nil {
			return err
		}

		err = m.MongoDB.AddProduct(models.Product{
			ID:          2,
			Name:        "Цемент М500",
			Category:    "Вяжущие материалы",
			Price:       350.00,
			Description: "Цемент М500 Д0, мешок 50 кг",
			InStock:     true,
			Supplier:    "Евроцемент",
		})
		if err != nil {
			return err
		}
	}

	// Проверка наличия данных в PostgreSQL
	products, err = m.PostgresDB.GetAllProducts()
	if err != nil {
		return err
	}

	if len(products) == 0 {
		// Добавляем тестовые данные в PostgreSQL (suppliers_db)
		err = m.PostgresDB.AddProduct(models.Product{
			ID:          1,
			Name:        "Клей для плитки",
			Category:    "Клеевые составы",
			Price:       280.00,
			Description: "Клей для керамической плитки, 25 кг",
			InStock:     true,
			Supplier:    "Цемикс",
		})
		if err != nil {
			return err
		}
	}

	// Проверка наличия данных в MySQL
	products, err = m.MySQLDB.GetAllProducts()
	if err != nil {
		return err
	}

	if len(products) == 0 {
		// Добавляем тестовые данные в MySQL (inventory_db)
		err = m.MySQLDB.AddProduct(models.Product{
			ID:          1,
			Name:        "Гипсокартон",
			Category:    "Листовые материалы",
			Price:       450.00,
			Description: "Гипсокартон влагостойкий, 12.5 мм, 1.2x2.5 м",
			InStock:     false,
			Supplier:    "Кнауф",
		})
		if err != nil {
			return err
		}
	}

	return nil
}
