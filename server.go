package huelio

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Server struct {
	CORSDomains string

	refreshInit       sync.Once
	refreshChan       chan struct{}
	refreshCancelChan chan struct{}

	Logger *log.Logger

	// Engine is the engine used by the server
	Engine *Engine
}

// refresh causes a server refresh, and returs once the refresh has been processed.
func (server *Server) RefreshOnce() {
	server.ensureRefreshChan()
	server.refreshChan <- struct{}{}
}

// RefreshInterval refreshes the server once
func (server *Server) RefreshInterval(interval time.Duration) {
	server.ensureRefreshChan()
	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				go server.RefreshOnce()
			case <-server.refreshCancelChan:
				return
			}
		}
	}(time.NewTicker(interval))
}

// Close cloeses the server and stops any ongoing refreshes
func (server *Server) Close() {
	server.ensureRefreshChan()
	select {
	case <-server.refreshCancelChan:
	default:
		server.Logger.Println("closing server")
		close(server.refreshCancelChan)
	}
}

func (server *Server) ensureRefreshChan() {
	server.refreshInit.Do(func() {
		server.refreshChan = make(chan struct{})
		server.refreshCancelChan = make(chan struct{})
		go func() {
			for {
				select {
				case <-server.refreshCancelChan:
					return
				case <-server.refreshChan:
					go func() {
						server.Logger.Print("refreshing caches")
						server.Engine.Use(nil)
						server.Logger.Printf("finished refreshing caches")
					}()
				}
			}
		}()
	})
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
		log.Println("GET")
		server.serveAction(w, r)
	case http.MethodGet:
		log.Println("POST")
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
	bridge, err := server.Engine.Bridge()
	if err != nil {
		return errors.Wrap(err, "Unable to find bridge")
	}
	return action.Do(bridge, server.Logger)

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
