package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"simple-http-server-GO/internal/taskstore"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type taskServer struct {
	store *taskstore.MongoTaskStore
}

func NewTaskServer() *taskServer {
	mongoUser := os.Getenv("MONGO_INITDB_ROOT_USERNAME")
	mongoPass := os.Getenv("MONGO_INITDB_ROOT_PASSWORD")
	mongoHost := os.Getenv("MONGO_HOST")
	if mongoUser == "" || mongoPass == "" || mongoHost == "" {
		log.Fatal("MongoDB credentials or host not set in environment variables")
	}

	// Формируем URI для подключения
	uri := fmt.Sprintf("mongodb://%s:%s@%s:27017", mongoUser, mongoPass, mongoHost)

	store, err := taskstore.NewMongo(uri, "tasksdb", "tasks")
	if err != nil {
		log.Fatalf("Cannot connect to MongoDB: %v", err)
	}

	return &taskServer{store: store}
}

// main обработчик для /task/
func (s *taskServer) taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == "/task/" {
			s.getAllTasksHandler(w, r)
		} else {
			s.getTaskHandler(w, r)
		}
	case http.MethodPost:
		s.createTaskHandler(w, r)
	case http.MethodDelete:
		if r.URL.Path == "/task/" {
			s.deleteAllTasksHandler(w, r)
		} else {
			s.deleteTaskHandler(w, r)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Обработчики
func (s *taskServer) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string   `json:"text"`
		Tags []string `json:"tags"`
		Due  string   `json:"due"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	due, err := time.Parse(time.RFC3339, req.Due)
	if err != nil {
		http.Error(w, "invalid due date format", http.StatusBadRequest)
		return
	}
	id := s.store.CreateTask(req.Text, req.Tags, due)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (s *taskServer) getAllTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.GetAllTasks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *taskServer) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	task, err := s.store.GetTask(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *taskServer) deleteAllTasksHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAllTasks(); err != nil {
		http.Error(w, "failed to delete tasks", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *taskServer) deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteTask(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *taskServer) getTasksByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		http.Error(w, "missing tag", http.StatusBadRequest)
		return
	}
	tasks := s.store.GetTasksByTag(tag)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *taskServer) getTasksByDueDateHandler(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	dayStr := r.URL.Query().Get("day")

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "invalid year", http.StatusBadRequest)
		return
	}
	monthInt, err := strconv.Atoi(monthStr)
	if err != nil {
		http.Error(w, "invalid month", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		http.Error(w, "invalid day", http.StatusBadRequest)
		return
	}

	tasks := s.store.GetTasksByDueDate(year, time.Month(monthInt), day)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

//	func init() {
//		// loads values from .env into the system
//		if err := godotenv.Load(); err != nil {
//			log.Print("No .env file found")
//		}
//	}
func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	server := NewTaskServer()
	mux := http.NewServeMux()

	// Главный обработчик для /task/
	mux.HandleFunc("/task/", server.taskHandler)

	// Отдельные обработчики для тегов и даты
	mux.HandleFunc("/tag/", server.getTasksByTagHandler)
	mux.HandleFunc("/due/", server.getTasksByDueDateHandler)

	// Порт
	port := os.Getenv("SERVERPORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on http://localhost:%s ...", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
