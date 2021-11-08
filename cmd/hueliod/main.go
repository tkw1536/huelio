package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tkw1536/huelio"
	"github.com/tkw1536/huelio/creds"
	"github.com/tkw1536/huelio/engine"
	"github.com/tkw1536/huelio/logging"
)

func main() {
	// create a manager and a store
	manager := &creds.Manager{
		Store: &creds.InMemoryStore{},
		Finder: creds.Finder{
			NewName: flagHueNewUsername,

			Username: flagHueUsername,
			Hostname: flagHueHost,
		},
	}
	if flagCredsPath != "" {
		manager.Store = creds.JSONFileStore(flagCredsPath)
	}

	server := &huelio.Server{
		Engine: &engine.Engine{
			Connect: manager.Connect,
		},

		RefreshInterval: flagCacheRefresh,

		DebugData: flagDebug,
	}
	if flagServerCORS {
		server.CORSDomains = "*"
	}

	go server.Start(globalContext)

	mux := http.NewServeMux()
	mux.Handle("/api/", server)

	if !flagDebug {
		mux.Handle("/", distServer)
	} else {
		mux.Handle("/", http.FileServer(http.Dir("./frontend/dist")))
	}

	httpServer := &http.Server{
		Addr:    flagServerBind,
		Handler: mux,
	}

	errChan := make(chan error)
	go func() {
		logger.Info().Str("bind", flagServerBind).Msg("server listening")
		errChan <- httpServer.ListenAndServe()
	}()

	go func() {
		<-globalContext.Done()
		logger.Info().Msg("server closing")
		httpServer.Close()
	}()

	<-errChan
}

//
// ctrl
//

var globalContext context.Context

func init() {
	var cancel context.CancelFunc
	globalContext, cancel = context.WithCancel(context.Background())

	cancelChan := make(chan os.Signal)
	signal.Notify(cancelChan, os.Interrupt)

	go func() {
		<-cancelChan
		cancel()
	}()
}

//
// static server
//

//go:embed frontend/dist
var dist embed.FS

var distServer http.Handler

func init() {
	dist, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}
	distServer = http.FileServer(http.FS(dist))
}

//
// logging
//

var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

//
// command line flags
//

var flagServerBind string = "localhost:8080"
var flagServerCORS bool = false
var flagDebug bool = false

var flagCacheRefresh time.Duration = 1 * time.Minute

var flagCredsPath string = ""
var flagHueHost string = os.Getenv("HUE_HOST")
var flagHueUsername string = os.Getenv("HUE_USER")
var flagHueNewUsername = fmt.Sprintf("hueliod-%d", time.Now().UnixMilli())

func init() {
	var legalFlag bool = false
	flag.BoolVar(&legalFlag, "legal", legalFlag, "Display legal notices and exit")
	defer func() {
		if legalFlag {
			fmt.Print(huelio.LegalText())
			os.Exit(0)
		}
	}()

	var flagQuiet bool = false
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet, "Supress all logging output")
	defer func() {
		if flagQuiet {
			logger = logger.Level(zerolog.Disabled)
		}
		logging.Init(&logger)
	}()

	defer flag.Parse()

	flag.StringVar(&flagServerBind, "bind", flagServerBind, "Address to bind server on")
	flag.BoolVar(&flagServerCORS, "cors", flagServerCORS, "Serve CORS headers")
	flag.BoolVar(&flagDebug, "debug", flagDebug, "Enable debugging mode: Send debug data and serve the frontend live instead of embedded")

	flag.DurationVar(&flagCacheRefresh, "refresh", flagCacheRefresh, "time to automatically refresh credentials on")

	flag.StringVar(&flagCredsPath, "store", flagCredsPath, "Path to read/write credentials from. When omitted, stores credentials in memory only. ")
	flag.StringVar(&flagHueHost, "host", flagHueHost, "Host to use for connection to Hue Bridge. Can also be given via HUE_HOST environment variable. ")
	flag.StringVar(&flagHueUsername, "user", flagHueUsername, "Username to use for connection to Hue Bridge. Can also be given via HUE_USER envionment variable. ")
	flag.StringVar(&flagHueNewUsername, "new-user", flagHueNewUsername, "Username to use when generating new username for hue bridge. Dynamically determined based on current time. ")
}
