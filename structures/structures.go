package structures

// Представляем задачу с основными атрибутами
type Task struct {
	ID      int64  `json:"id,string"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Представляем ответ API с возможными ошибками
type Response struct {
	ID    int64  `json:"id,omitempty,string"`
	Error string `json:"error,omitempty"`
}

// Представляем ответ API, содержащий список задач
type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
}
