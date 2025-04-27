package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project/internal/api"
	"project/internal/storage"
)

func main() {
	// Инициализируем менеджер баз данных
	dbManager, err := storage.NewDBManager()
	if err != nil {
		log.Fatalf("Ошибка при инициализации менеджера баз данных: %v", err)
	}

	// Отложенное закрытие соединений с базами данных
	defer dbManager.Close()

	// Инициализация тестовых данных
	if err := dbManager.InitializeTestData(); err != nil {
		log.Printf("Предупреждение: не удалось инициализировать тестовые данные: %v", err)
	}

	// Настраиваем обработку статических файлов
	api.SetupStaticFiles()

	// Настраиваем маршруты API
	api.SetupRoutes(dbManager)

	port := ":8080"
	server := &http.Server{
		Addr:    port,
		Handler: nil, // Использует DefaultServeMux
	}

	// Канал для сигналов завершения работы
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Сервер запущен на порту %s", port)
		log.Printf("Веб-интерфейс доступен по адресу http://localhost%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	<-stop
	log.Println("Завершение работы сервера...")

	// Создаем контекст с таймаутом для корректного завершения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Корректно останавливаем сервер
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен")
}
