//go:build gui

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
		config.Main(iL, config.Ctx)
		close(doneChan)
	}()

	go func() {
		<-config.Ctx.Done()
		ww.Close()
	}()

	listener := gui.NewComboListener(gui.NewKeyCombination("ctrl+h"), config.Ctx)
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

func initctrlc() {
	config.Ctx = logger.WithContext(context.Background())

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

var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config = service.DefaultConfig()

var flagCombo = "ctrl+h"

func init() {
	defer initctrlc()

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
	flag.StringVar(&flagCombo, "combo", flagCombo, "key combination to press for trigger")
	flag.Parse()
}
