package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tkw1536/huelio"
)

func main() {
	// create a new partial finder
	pf := &huelio.PartialFinder{
		Logger: logger,

		NewName: apiNewUsername,

		Username: apiUsername,
		Hostname: apiHost,
	}

	// create a manager and a store
	manager := &huelio.StoreManager{
		Store:  &huelio.InMemoryStore{},
		Finder: pf,
	}
	if apiStore != "" {
		manager.Store = huelio.JSONFileStore(apiStore)
	}

	server := &huelio.Server{
		Engine: &huelio.Engine{
			Connect: manager.Connect,
		},
		Logger: logger,

		RefreshInterval: refreshInterval,
	}
	if apiCORS {
		server.CORSDomains = "*"
	}

	go server.Start(context.Background())

	server.Logger.Printf("Listening on %q", bindHost)
	http.ListenAndServe(bindHost, server)
}

var logger = log.New(os.Stderr, "", log.LstdFlags)

var bindHost string

var apiStore string

var apiHost string
var apiUsername string
var apiNewUsername = fmt.Sprintf("hueliod-%d", time.Now().UnixMilli())

var apiCORS bool

var refreshInterval time.Duration

func init() {
	defer flag.Parse()

	flag.DurationVar(&refreshInterval, "refresh", 1*time.Minute, "how often to refresh bridge cache")
	flag.StringVar(&bindHost, "bind", "localhost:8080", "host to listen on")

	flag.StringVar(&apiStore, "store", "", "path to store credentials. In-Memory store used when omitted.")
	flag.StringVar(&apiHost, "host", os.Getenv("HUE_HOST"), "hue hostname")
	flag.StringVar(&apiUsername, "user", os.Getenv("HUE_USER"), "hue username")

	flag.BoolVar(&apiCORS, "cors", false, "add CORS headers")
}
