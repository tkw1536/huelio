package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tkw1536/huelio/gui"
	"github.com/tkw1536/huelio/logging"
)

func init() {
	runtime.LockOSThread() // we need to run everything on main because of the webview
}

func init() {
	var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	logging.Init(&logger)
}

func main() {
	ww := gui.NewWindowLoop(gui.WebView)

	go func() {
		<-globalContext.Done()
		ww.Close()
	}()

	listener := gui.NewComboListener(gui.NewKeyCombination("ctrl+h"), globalContext)
	go func() {
		for range listener {
			ww.OpenOrFocus(gui.WindowParams{
				Title: "Hello world",
				URL:   "http://example.com/",
			})
		}
	}()

	ww.Run()
}

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
