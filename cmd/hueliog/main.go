package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tkw1536/huelio"
	"github.com/tkw1536/huelio/gui"
	"github.com/tkw1536/huelio/logging"
	"github.com/tkw1536/huelio/service"
)

func init() {
	runtime.LockOSThread() // we need to run everything on main because of the webview
}

func main() {
	iL, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Error().Err(err).Msg("Unable to listen")
	}
	addr := iL.Addr().String()

	ww := gui.NewWindowLoop(gui.WebView)

	doneChan := make(chan struct{})
	go func() {
		config.Main(iL, globalContext)
		close(doneChan)
	}()

	go func() {
		<-globalContext.Done()
		ww.Close()
	}()

	listener := gui.NewComboListener(gui.NewKeyCombination("ctrl+h"), globalContext)
	go func() {
		for range listener {
			ww.OpenOrFocus(gui.WindowParams{
				Title: "Huelio G.",
				URL:   "http://" + addr,

				Width:  800,
				Height: 800,
			})
		}
	}()

	ww.Run()
	<-doneChan
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

var flagCombo = "ctrl+h"

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
	flag.StringVar(&flagCombo, "combo", flagCombo, "key combination to press for trigger")
	flag.Parse()
}
