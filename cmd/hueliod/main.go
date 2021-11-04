package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/amimof/huego"
	"github.com/tkw1536/huelio"
)

func main() {
	server := &huelio.Server{
		Engine: huelio.NewEngine(&huego.Bridge{Host: apiHost, User: apiUsername}),

		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
	if apiCORS {
		server.CORSDomains = "*"
	}

	server.RefreshInterval(refreshInterval)
	defer server.Close()

	server.Logger.Printf("Listening on %q", hostname)
	http.ListenAndServe(hostname, server)
}

var hostname string

var apiHost string
var apiUsername string

var apiCORS bool

var refreshInterval time.Duration

func init() {
	defer flag.Parse()

	flag.DurationVar(&refreshInterval, "refresh", 1*time.Minute, "how often to refresh bridge cache")
	flag.StringVar(&hostname, "bind", "localhost:8080", "host to listen on")
	flag.StringVar(&apiHost, "host", os.Getenv("HUE_HOST"), "hue hostname")
	flag.StringVar(&apiUsername, "user", os.Getenv("HUE_USER"), "hue username")
	flag.BoolVar(&apiCORS, "cors", false, "add CORS headers")
}
