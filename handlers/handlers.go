package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"go_final_project/database"
	"go_final_project/pocessing"
	"go_final_project/structures"
	"net/http"
	"strconv"
	"time"
)

// Вычисляем следующую дату выполнения задачи на основе правила повторения
func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Проверяем, что переданы все необходимые параметры
	if nowStr == "" || dateStr == "" || repeat == "" {
		http.Error(w, "missing required parameters: now, date or repeat", http.StatusBadRequest)
		return
	}

	// Преобразуем текущую дату в нужный формат
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "invalid format for 'now': "+nowStr, http.StatusBadRequest)
		return
	}

	// Вычисляем следующую дату повторения задачи
	nextDate, err := pocessing.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

// Обрабатываем запрос на добавление новой задачи в базу данных
func AddTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var task structures.Task
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		responseError(w, "failed to read the request body", http.StatusBadRequest)
		return
	}

	// Десериализуем JSON в структуру задачи
	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		responseError(w, "failed to deserialize JSON", http.StatusBadRequest)
		return
	}

	// Обрабатываем задачу (например, проверяем корректность данных)
	if err = pocessing.ProcessTask(&task); err != nil {
		responseError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Добавляем задачу в базу данных
	if err = database.AddTask(db, &task); err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := structures.Response{ID: task.ID}
	writeJSON(w, response, http.StatusOK)
}

// отправляем JSON-ответ с ошибкой
func responseError(w http.ResponseWriter, message string, statusCode int) {
	response := structures.Response{Error: message}
	writeJSON(w, response, statusCode)
}

// Сериализуем данные в JSON и отправляет их клиенту
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)

	jsonResponse, err := json.Marshal(data)
	if err != nil {
		http.Error(w, `{"error":"failed to serialize JSON"}`, http.StatusInternalServerError)
		return
	}
	w.Write(jsonResponse)
}

// Получаем список задач с возможностью поиска по заголовку или комментарию
func GetTasksHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		responseError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	search := r.URL.Query().Get("search")

	tasks, err := database.GetTasksWithSearch(db, search)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := structures.TaskListResponse{Tasks: tasks}
	writeJSON(w, response, http.StatusOK)
}

// Получаем конкретную задачу по её ID
func GetTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		responseError(w, "task ID is required", http.StatusBadRequest)
		return
	}

	parsedId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		responseError(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := database.GetTask(db, parsedId)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := structures.Task{
		ID:      task.ID,
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	}
	writeJSON(w, response, http.StatusOK)
}

// Обновляем информацию о задаче
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var task structures.Task
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		responseError(w, "failed to read the request body", http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		responseError(w, "failed to deserialize JSON", http.StatusBadRequest)
		return
	}

	if err = pocessing.ProcessTask(&task); err != nil {
		responseError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateTask(db, &task)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{}, http.StatusOK)
}

// Удаляем задачу по её ID
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		responseError(w, "task ID is required", http.StatusBadRequest)
		return
	}

	parsedId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		responseError(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	err = database.DeleteTask(db, parsedId)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{}, http.StatusOK)
}

// Отмечаем задачу выполненной, обновляя её дату или удаляя её
func TaskDoneHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		responseError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		responseError(w, "task ID is required", http.StatusBadRequest)
		return
	}

	parsedId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		responseError(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := database.GetTask(db, parsedId)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if task.Repeat == "" {
		err = database.DeleteTask(db, parsedId)
		if err != nil {
			responseError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{}, http.StatusOK)
		return
	}

	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		responseError(w, "invalid date format, expected YYYYMMDD", http.StatusBadRequest)
	}

	nextDate, err := pocessing.NextDate(parsedDate, task.Date, task.Repeat)
	if err != nil {
		responseError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateTaskDate(db, parsedId, nextDate)
	if err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{}, http.StatusOK)
}
