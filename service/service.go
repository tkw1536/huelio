package service

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio/creds"
	"github.com/tkw1536/huelio/engine"
	"github.com/tkw1536/huelio/frontend"
)

type ServiceConfig struct {
	Ctx        context.Context
	ServerCORS bool

	Quiet bool
	Debug bool

	CacheRefresh time.Duration

	CredsPath string

	HueHost        string
	HueUsername    string
	HueNewUsername string
}

func (s *ServiceConfig) logger() zerolog.Logger {
	return zerolog.Ctx(s.Ctx).With().Str("component", "service.Service").Logger()
}

// DefaultConfig returns a new default config
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		Ctx: context.Background(),

		ServerCORS: false,

		Debug: false,

		CacheRefresh: 1 * time.Minute,

		HueHost:        os.Getenv("HUE_HOST"),
		HueUsername:    os.Getenv("HUE_USER"),
		HueNewUsername: fmt.Sprintf("hueliod-%d", time.Now().UnixMilli()),
	}
}

// AddFlagsTo adds flags for this ServiceConfig to the provided flagset.
// When flagset is nil, uses flag.CommandLine
func (s *ServiceConfig) AddFlagsTo(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.BoolVar(&s.ServerCORS, "cors", s.ServerCORS, "Serve CORS headers")

	flagset.BoolVar(&s.Debug, "debug", s.Debug, "Enable debugging mode: Send debug data and serve the frontend live instead of embedded")

	flagset.DurationVar(&s.CacheRefresh, "refresh", s.CacheRefresh, "time to automatically refresh credentials on")

	flagset.StringVar(&s.CredsPath, "store", s.CredsPath, "Path to read/write credentials from. When omitted, stores credentials in memory only. ")
	flagset.StringVar(&s.HueHost, "host", s.HueHost, "Host to use for connection to Hue Bridge. Can also be given via HUE_HOST environment variable. ")
	flagset.StringVar(&s.HueUsername, "user", s.HueUsername, "Username to use for connection to Hue Bridge. Can also be given via HUE_USER envionment variable. ")
	flagset.StringVar(&s.HueNewUsername, "new-user", s.HueNewUsername, "Username to use when generating new username for hue bridge. Dynamically determined based on current time. ")
}

// Main Starts the service and returns when it is finished
func (s ServiceConfig) Main(listener net.Listener) {
	serviceLogger := s.logger()
	// create a manager and a store
	manager := &creds.Manager{
		Ctx:   s.Ctx,
		Store: &creds.InMemoryStore{},
		Finder: creds.Finder{
			Ctx:     s.Ctx,
			NewName: s.HueNewUsername,

			Username: s.HueUsername,
			Hostname: s.HueHost,
		},
	}
	if s.CredsPath != "" {
		manager.Store = creds.JSONFileStore(s.CredsPath)
	}

	server := &Server{
		Ctx: s.Ctx,

		Engine: &engine.Engine{
			Ctx:     s.Ctx,
			Connect: manager.Connect,
		},

		RefreshInterval: s.CacheRefresh,

		DebugData: s.Debug,
	}
	if s.ServerCORS {
		server.CORSDomains = "*"
	}

	go server.Start()

	mux := http.NewServeMux()
	mux.Handle("/api/", server)

	if !s.Debug {
		mux.Handle("/", frontend.StaticHandler)
	} else {
		// find the dist directorys
		_, fn, _, _ := runtime.Caller(0)
		dist := filepath.Join(filepath.Dir(fn), "..", "frontend", "dist")
		// and run a fileserver
		mux.Handle("/", http.FileServer(http.Dir(dist)))
	}

	httpServer := &http.Server{
		Handler: mux,
	}

	errChan := make(chan error)
	go func() {
		serviceLogger.Info().Str("bind", listener.Addr().String()).Msg("server listening")
		errChan <- httpServer.Serve(listener)
	}()

	go func() {
		<-s.Ctx.Done()
		serviceLogger.Info().Msg("server closing")
		httpServer.Close()
	}()

	<-errChan
}
