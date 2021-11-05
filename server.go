package huelio

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Server struct {
	CORSDomains     string
	RefreshInterval time.Duration

	Logger *log.Logger
	Engine *Engine
}

// Start starts server background tasks.
// It blocks and should be started in a seperate goroutine.
func (server *Server) Start(context context.Context) {
	go server.Engine.Link()

	var c <-chan time.Time
	if server.RefreshInterval > 0 {
		ticker := time.NewTicker(server.RefreshInterval)
		defer ticker.Stop()

		c = ticker.C
	}

	for {
		select {
		case <-c:
			server.Logger.Printf("Refreshing Index")
			server.Engine.RefreshIndex()
			server.Logger.Printf("Index refreshed")
		case <-context.Done():
			return
		}
	}
}

type jsonMessage struct {
	Message string `json:"message"`
}

// ServeHTTP responds to a http request
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		log.Println("OPTIONS")
		server.writeJSON(w, http.StatusOK, jsonMessage{Message: "this is fine"})
	case http.MethodPost:
		log.Println("POST")
		server.serveAction(w, r)
	case http.MethodGet:
		log.Println("GET")
		server.serveQuery(w, r)
	default:
		server.writeJSON(w, http.StatusMethodNotAllowed, jsonMessage{Message: "method allowed"})
	}
}

var emptyResultHack = []QueryAction{}

func (server *Server) serveQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	the_query, ok := query["query"]
	if !ok {
		server.writeJSON(w, http.StatusBadRequest, jsonMessage{Message: "missing 'query' url parameter"})
		return
	}

	res, err := server.Engine.Query(strings.Join(the_query, " "))
	if err != nil {
		server.writeJSON(w, http.StatusInternalServerError, jsonMessage{Message: err.Error()})
		return
	}
	if res == nil {
		res = emptyResultHack
	}
	server.writeJSON(w, http.StatusOK, res)
}

func (server *Server) serveAction(w http.ResponseWriter, r *http.Request) {
	err := server.doAction(w, r)
	if err != nil {
		server.writeJSON(w, http.StatusInternalServerError, jsonMessage{Message: err.Error()})
		return
	}
	server.writeJSON(w, http.StatusOK, jsonMessage{Message: "Success"})
}

func (server *Server) doAction(w http.ResponseWriter, r *http.Request) error {
	action := QueryAction{}
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		return errors.Wrap(err, "Unable to parse body")
	}
	return server.Engine.Do(action)

}

func (server *Server) writeJSON(w http.ResponseWriter, statusCode int, content interface{}) {
	bytes, err := json.Marshal(content)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}
	h := w.Header()

	h.Add("Content-Type", "application/json")
	if server.CORSDomains != "" {
		h.Add("Access-Control-Allow-Origin", server.CORSDomains)
		h.Add("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		h.Add("Access-Control-Allow-Headers", "*")
	}
	w.WriteHeader(statusCode)

	w.Write(bytes)
}

func init() {
	var _ http.Handler
}
