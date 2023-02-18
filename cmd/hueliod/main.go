package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio"
	"github.com/tkw1536/huelio/service"
)

func main() {
	listener, err := net.Listen("tcp", flagServerBind)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to listen")
		return
	}
	defer listener.Close()
	config.Main(listener)
}

//
// ctrl+c
//

func initcontext() {
	logger = logger.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.DateTime,
	}).With().Timestamp().Logger()

	config.Ctx = logger.WithContext(context.Background())

	// handle ctrl + c
	var cancel context.CancelFunc
	config.Ctx, cancel = signal.NotifyContext(config.Ctx, os.Interrupt)

	go func() {
		defer cancel()
		<-config.Ctx.Done()
	}()

}

//
// command line flags
//

var logger = zerolog.New(os.Stdout)
var config = service.DefaultConfig()

var flagServerBind = "localhost:8080"

func init() {
	defer initcontext()

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
	}()

	config.AddFlagsTo(nil)
	flag.StringVar(&flagServerBind, "bind", flagServerBind, "Address to bind server on")
	flag.Parse()
}
