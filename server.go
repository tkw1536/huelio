package huelio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio/engine"
	"github.com/tkw1536/huelio/logging"
)

var serverLogger zerolog.Logger

func init() {
	logging.ComponentLogger("Server", &serverLogger)
}

type Server struct {
	RefreshInterval time.Duration // how often should the index be refreshed

	DebugData bool // should we marshal out extra debug info (like scores, errors, etc)?

	CORSDomains string // should we include cors headers on every API response?

	Engine *engine.Engine
}

// Start starts server background tasks.
// It blocks and should be started in a seperate goroutine.
func (server *Server) Start(context context.Context) {
	serverLogger.Info().Msg("starting server background tasks")
	defer func() {
		serverLogger.Info().Msg("exiting server background tasks")
	}()

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
			server.Engine.RefreshIndex()
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
	serverLogger.Info().Str("method", r.Method).Stringer("url", r.URL).Msg("request")
	switch r.Method {
	case http.MethodOptions:
		server.writeJSON(w, http.StatusOK, jsonMessage{Message: "this is fine"})
	case http.MethodPost:
		server.serveAction(w, r)
	case http.MethodGet:
		server.serveQuery(w, r)
	default:
		server.writeJSON(w, http.StatusMethodNotAllowed, jsonMessage{Message: "method allowed"})
	}
}

func (server *Server) serveQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	the_query, ok := query["query"]
	if !ok {
		server.writeJSON(w, http.StatusBadRequest, jsonMessage{Message: "missing 'query' url parameter"})
		return
	}

	res, scores, err := server.Engine.Query(strings.Join(the_query, " "))
	if err != nil {
		server.writeJSON(w, http.StatusInternalServerError, jsonMessage{Message: err.Error()})
		return
	}

	server.writeJSON(w, http.StatusOK, MarshalResult{
		Results:   res,
		Scores:    scores,
		WithScore: server.DebugData,
	})
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
	action := engine.Action{}
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		return errors.Wrap(err, "Unable to parse body")
	}
	return server.Engine.Do(action)

}

func (server *Server) writeJSON(w http.ResponseWriter, statusCode int, content interface{}) {
	serverLogger.Info().Int("status", statusCode).Msg("response")

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
