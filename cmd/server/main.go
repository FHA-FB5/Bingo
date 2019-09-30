package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"

	"github.com/FHA-FB5/BingoBongo/pkg/model"
	"github.com/FHA-FB5/BingoBongo/pkg/services"
)

func main() {
	var (
		logger       = log.New()
		sessionName  = flag.String("session.name", "some-name", "session name")
		sessionKey   = flag.String("session.key", "some_key", "session key")
		tasks        = flag.String("task.filename", "tasks/rally.json", "tasks")
		templatePath = flag.String("template.path", "web/template", "templates")
		groupsFile   = flag.String("groups.filename", "groups/groups.json", "groups")
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
	groupData, err := ioutil.ReadFile(*groupsFile)
	if err != nil {
		log.Fatal(err)
	}
	var groups map[string]string
	if err := json.Unmarshal(groupData, &groups); err != nil {
		log.Fatal(err)
	}
	h := &services.Handler{
		Logger:      logger,
		Store:       sessions.NewCookieStore([]byte(*sessionKey)),
		SessionName: *sessionName,
		Tasks:       t,
		Templates:   template.Must(template.ParseGlob(filepath.Join(*templatePath, "*.tmpl"))),
		Groups:      groups,
	}
	r := mux.NewRouter()
	r.PathPrefix("/static").Handler(http.FileServer(http.Dir("web")))
	r.Path("/").Methods("GET").HandlerFunc(h.Index)
	r.Path("/event").Methods("GET").HandlerFunc(h.Event)
	r.Path("/upload").Methods("POST").HandlerFunc(h.PostFile)
	http.Handle("/", r)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
