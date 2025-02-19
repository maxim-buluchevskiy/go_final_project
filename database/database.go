package database

import (
	"database/sql"
	"fmt"
	"go_final_project/structures"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Инициализируем базу данных SQLite
func InitDB() (*sql.DB, error) {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		appPath, err := os.Executable()
		if err != nil {
			return nil, err
		}
		dbFile = filepath.Join(filepath.Dir(appPath), "scheduler.db")
	}
	_, err := os.Stat(dbFile)
	install := os.IsNotExist(err)

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("error while open db: %w", err)
	}

	// Если база данных отсутствовала, создаем таблицу
	if install {
		if err = createTable(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func createTable(db *sql.DB) error {
	query := `
		CREATE TABLE scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date CHAR(8) NOT NULL DEFAULT "",
			title TEXT NOT NULL DEFAULT "",
			comment TEXT DEFAULT "",
			repeat VARCHAR(128) DEFAULT ""
		);
		CREATE INDEX scheduler_date ON scheduler (date)
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Failed to create table: %v", err)
		return err
	}
	return nil
}

// Добавляем новую задачу в БД
func AddTask(db *sql.DB, task *structures.Task) error {
	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	// Получаем ID добавленной задачи
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to retrieve last insert ID: %w", err)
	}

	task.ID = id
	return nil
}

// Получаем задачи с возможностью поиска по дате или тексту
func GetTasksWithSearch(db *sql.DB, search string) ([]structures.Task, error) {
	var query string
	var args []interface{}

	if search != "" {
		// Пробуем интерпретировать поисковый запрос как дату
		parsedDate, err := time.Parse("02.01.2006", search)
		if err == nil {
			// Если это дата, ищем задачи по ней
			query = `
				SELECT id, date, title, comment, repeat
				FROM scheduler
				WHERE date = ?
				ORDER BY date
				LIMIT 50
			`
			args = append(args, parsedDate.Format("20060102"))
		} else {
			// Иначе ищем по заголовку и комментарию
			searchPattern := "%" + search + "%"
			query = `
				SELECT id, date, title, comment, repeat
				FROM scheduler
				WHERE title LIKE ? OR comment LIKE ?
				ORDER BY date
				LIMIT 50
			`
			args = append(args, searchPattern, searchPattern)
		}
	} else {
		// Если поиска нет, получаем все задачи (ограничено 50 записями)
		query = `
			SELECT id, date, title, comment, repeat
			FROM scheduler
			ORDER BY date
			LIMIT 50
		`
	}

	// Выполняем SQL-запрос
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer rows.Close()

	var tasks []structures.Task
	for rows.Next() {
		var task structures.Task
		if err = rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, fmt.Errorf("failed to parse tasks: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tasks: %w", err)
	}

	if tasks == nil {
		tasks = []structures.Task{}
	}

	return tasks, nil
}

// Получаем задачу по её ID
func GetTask(db *sql.DB, id int64) (*structures.Task, error) {
	query := `
		SELECT id, date, title, comment, repeat
		FROM scheduler
		WHERE id = ?
	`

	row := db.QueryRow(query, id)

	var task structures.Task
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return nil, fmt.Errorf("task not found")
	}

	return &task, nil
}

// Обновляем информацию о задаче в БД
func UpdateTask(db *sql.DB, task *structures.Task) error {
	query := `UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`

	res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Проверяем, была ли обновлена хотя бы одна строка
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Удаляем задачу по её ID
func DeleteTask(db *sql.DB, id int64) error {
	query := `DELETE FROM scheduler WHERE id = ?`

	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Обновляем только дату выполнения задачи
func UpdateTaskDate(db *sql.DB, id int64, nextDate string) error {
	query := `UPDATE scheduler SET date = ? WHERE id = ?`

	res, err := db.Exec(query, nextDate, id)
	if err != nil {
		return fmt.Errorf("failed to update task date: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}
