package services

import (
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
	Templates   *template.Template
	Groups      map[string]string
}

const (
	// maxFileSize is 10MB
	maxFileSize = int64(10 << 20)
)

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if err := h.Templates.ExecuteTemplate(w, "index.tmpl", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) Event(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token == "" {
		http.Error(w, "token missing", http.StatusBadRequest)
		return
	}
	group := h.getGroupForToken(token)
	if group == "" {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}
	if h.Tasks == nil {
		log.Error("tasks is nil")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tasks := []model.Task(*h.Tasks)
	for i := range tasks {
		tasks[i].ID = i
	}
	tmplData := model.Bingo{
		Token: token,
		Tasks: tasks,
	}
	if err := h.Templates.ExecuteTemplate(w, "base.tmpl", tmplData); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Handler) PostFile(w http.ResponseWriter, r *http.Request) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		http.Error(w, "no content-type specified", http.StatusBadRequest)
		return
	}
	switch contentType {
	case "multipart/form-data":
		h.handleFile(w, r)
	case "application/x-www-form-urlencoded":
		h.handleText(w, r)
	default:
		http.Error(w, "invalid type specified", http.StatusBadRequest)
	}
}

func (h Handler) handleFile(w http.ResponseWriter, r *http.Request) {
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
	if err == multipart.ErrMessageTooLarge {
		http.Error(w, fmt.Sprintf("maximum file size is %.3fMB", float64(maxFileSize)/(1<<20)), http.StatusBadRequest)
		return
	} else if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
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
		http.Error(w, "token missing", http.StatusBadRequest)
		return
	}
	token := t[0]
	fileType, err := h.Tasks.TypeByID(taskID)
	if err != nil {
		http.Error(w, "task does not exist", http.StatusNotFound)
		return
	}
	group := h.getGroupForToken(token)
	if group == "" {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}
	path := fmt.Sprintf("storage/%s/", group)
	name := fmt.Sprintf("%d_%s.%s", taskID, time.Now().Format(time.RFC3339), fileType)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	f, err := os.Create(path + name)
	if err != nil {
		log.Error(err)
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
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(f, file); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/event?token="+token, http.StatusNoContent)
}

func (h Handler) handleText(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token == "" {
		http.Error(w, "token missing", http.StatusBadRequest)
		return
	}
	task := r.FormValue("task")
	taskID, err := strconv.Atoi(task)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	fileType, err := h.Tasks.TypeByID(taskID)
	if err != nil {
		http.Error(w, "task does not exist", http.StatusNotFound)
		return
	}
	group := h.getGroupForToken(token)
	if group == "" {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}
	path := fmt.Sprintf("storage/%s/", group)
	name := fmt.Sprintf("%d_%s.%s", taskID, time.Now().Format(time.RFC3339), fileType)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	f, err := os.Create(path + name)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error(err)
		}
	}()
	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "no text", http.StatusBadRequest)
		return
	}
	if _, err := f.WriteString(text); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/event?token="+token, http.StatusNoContent)
}

func (h Handler) getGroupForToken(token string) string {
	for t, group := range h.Groups {
		if t == token {
			return group
		}
	}
	return ""
}
