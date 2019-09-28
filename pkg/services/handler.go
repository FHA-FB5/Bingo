package services

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/sessions"

	"github.com/FHA-FB5/BingoBongo/pkg/model"
)

type Handler struct {
	Logger      log.FieldLogger
	Store       *sessions.CookieStore
	SessionName string
	Tasks       *model.Tasks
}

const (
	// maxFileSize is 10MB
	maxFileSize = int64(10 << 20)
)

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) PostFile(w http.ResponseWriter, r *http.Request) {
	contentType, options, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		http.Error(w, "no content-type specified", http.StatusBadRequest)
		return
	}
	if contentType != "multipart/form-data" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}
	boundary, ok := options["boundary"]
	if !ok || boundary == "" {
		http.Error(w, "invalid boundary", http.StatusBadRequest)
		return
	}
	mr := multipart.NewReader(r.Body, boundary)
	form, err := mr.ReadForm(maxFileSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("maximum file size is %.3fMB", float64(maxFileSize)/(1<<20)), http.StatusBadRequest)
		return
	}
	task, ok := form.Value["task"]
	if !ok || len(task) < 1 {
		http.Error(w, "task id missing", http.StatusBadRequest)
		return
	}
	taskID, err := strconv.Atoi(task[0])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	t, ok := form.Value["token"]
	if !ok || len(t) < 1 {
		http.Error(w, "token is missing", http.StatusBadRequest)
		return
	}
	token := t[0]
	fileType, err := h.Tasks.TypeByID(taskID)
	if err != nil {
		http.Error(w, "task does not exist", http.StatusNotFound)
		return
	}
	path := fmt.Sprintf("storage/%s/", token)
	name := fmt.Sprintf("%d_%s.%s", taskID, time.Now().Format(time.RFC3339), fileType)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	f, err := os.Create(path + name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error(err)
		}
	}()

	header, ok := form.File["file"]
	if !ok || len(header) < 1 {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	file, err := header[0].Open()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(f, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusNoContent)
}

func (h *Handler) GetTasks(w http.ResponseWriter, _ *http.Request) {
	if err := json.NewEncoder(w).Encode(h.Tasks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
