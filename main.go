package main

import (
	"go_final_project/database"
	"go_final_project/handlers"
	"go_final_project/tests"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {
	// Загружаем переменные окружения
	err := godotenv.Load("variable.env")
	if err != nil {
		log.Fatalf("Error loading variables.env file: %v", err)
	}

	// Получаем порт сервера
	port, err := strconv.Atoi(os.Getenv("TODO_PORT"))
	if err != nil {
		port = tests.Port
	}

	// Инициализируем базу данных
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Настраиваем маршруты HTTP-сервера
	http.Handle("/", http.FileServer(http.Dir("web")))
	// Маршрут для расчёта следующей даты
	http.HandleFunc("/api/nextdate", handlers.NextDateHandler)
	// Маршрут для работы с одной задачей
	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTaskHandler(w, r, db)
		case http.MethodPost:
			handlers.AddTaskHandler(w, r, db)
		case http.MethodPut:
			handlers.UpdateTaskHandler(w, r, db)
		case http.MethodDelete:
			handlers.DeleteTaskHandler(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Маршрут для получения всех задач
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetTasksHandler(w, r, db)
	})
	// Маршрут для пометки задачи как выполненной
	http.HandleFunc("/api/task/done", func(w http.ResponseWriter, r *http.Request) {
		handlers.TaskDoneHandler(w, r, db)
	})

	// Запускаем сервер в горутине
	server := &http.Server{Addr: ":" + strconv.Itoa(port)}

	go func() {
		log.Printf("Server is running on port %d...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Блокируем выполнение, чтобы программа не завершалась
	select {}
}
