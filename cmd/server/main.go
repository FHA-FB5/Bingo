package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"

	"github.com/FHA-FB5/BingoBongo/pkg/model"
	"github.com/FHA-FB5/BingoBongo/pkg/services"
)

func main() {
	var (
		logger      = log.New()
		sessionName = flag.String("session.name", "some-name", "session name")
		sessionKey  = flag.String("session.key", "some_key", "session key")
		tasks       = flag.String("task.filename", "tasks.json", "tasks")
	)
	flag.Parse()

	t := &model.Tasks{}
	b, err := ioutil.ReadFile(*tasks)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(b, t); err != nil {
		log.Fatal(err)
	}

	h := &services.Handler{
		Logger:      logger,
		Store:       sessions.NewCookieStore([]byte(*sessionKey)),
		SessionName: *sessionName,
		Tasks:       t,
	}

	r := mux.NewRouter()
	r.Path("/").Methods("GET").HandlerFunc(h.Index)
	r.Path("/tasks").Methods("GET").HandlerFunc(h.GetTasks)
	r.Path("/upload").Methods("POST").HandlerFunc(h.PostFile)
	http.Handle("/", r)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
