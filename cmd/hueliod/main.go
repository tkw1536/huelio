package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tkw1536/huelio"
	"github.com/tkw1536/huelio/logging"
	"github.com/tkw1536/huelio/service"
)

func main() {
	listener, err := net.Listen("tcp", flagServerBind)
	defer listener.Close()
	if err != nil {
		logger.Error().Err(err).Msg("Unable to listen")
		return
	}
	config.Main(listener, globalContext)
}

//
// ctrl+c
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
// command line flags
//

var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config = service.DefaultConfig()

var flagServerBind = "localhost:8080"

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

	config.AddFlagsTo(nil)
	flag.StringVar(&flagServerBind, "bind", flagServerBind, "Address to bind server on")
	flag.Parse()
}
