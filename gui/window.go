//go:build gui

package gui

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

// WindowLoop represents a window that keeps being shown
type WindowLoop interface {
	// Run runs this WindowLoop.
	// It will block until another thread calls Stop().
	//
	// Run *must* be called from the main thread.
	// This can be achieved by calling runtime.LockOSThread() in an init() function
	// And then calling this function towards the end of main().
	Run()

	// Open opens a new window in this WindowLoop if one is not already open.
	Open(window WindowParams) bool

	// Focus brings the currently open window to the front
	Focus() bool

	// OpenOrFocus opens a new window, or focuses the current one.
	OpenOrFocus(window WindowParams) bool

	// Close stops this WindowLoop, by closing the current window
	// Stop may be called at most once.
	Close()
}

func NewWindowLoop(backend WindowBackend) WindowLoop {
	ww := &windowLoop{
		backend: backend,
	}
	ww.Init()
	return ww
}

// windowLoop represents an eternal windowLoop
type windowLoop struct {
	backend WindowBackend

	Ctx context.Context

	wnChan chan WindowParams // send to open a new window!
	wfChan chan struct{}     // send to focus the window!
	wcChan chan struct{}     // send to close the window!

	close sync.Once // close everything once
}

type WindowParams struct {
	Title string
	URL   string

	Width, Height int
}

func (wl *windowLoop) Open(params WindowParams) bool {
	select {
	case wl.wnChan <- params:
		return true
	default:
		return false
	}
}

func (wl *windowLoop) Focus() bool {
	select {
	case wl.wfChan <- struct{}{}:
		return true
	default:
		return false
	}
}

func (wl *windowLoop) OpenOrFocus(params WindowParams) bool {
	select {
	case wl.wnChan <- params:
		return true
	case wl.wfChan <- struct{}{}:
		return true
	default:
		return false
	}
}

func (wl *windowLoop) Init() {
	wl.wnChan = make(chan WindowParams)
	wl.wcChan = make(chan struct{})
	wl.wfChan = make(chan struct{})
}

func (ww *windowLoop) Close() {
	select {
	case ww.wcChan <- struct{}{}:
	default:
	}

	ww.close.Do(func() {
		close(ww.wnChan)
	})
}

// Run runs the main loop of this webview.
//
// It *must* be called on the main thread; as such the main() function should call it.
func (ww *windowLoop) Run() {
	windowLogger := zerolog.Ctx(ww.Ctx).With().Str("component", "gui.Window").Logger()

	for p := range ww.wnChan {
		windowLogger.Info().Msg("Opening new window")
		wwDone := make(chan struct{})

		inst, err := ww.backend.create(p)
		if err != nil {
			windowLogger.Error().Err(err).Msg("Unable to create new window")
			continue // wait for the next one
		}

		go func() {
			for {
				select {
				case <-wwDone:
					return
				case <-ww.wfChan:
					windowLogger.Info().Msg("Focusing window")
					ww.backend.requestFocus(inst)
				case <-ww.wcChan:
					windowLogger.Info().Msg("Closing window")
					ww.backend.requestClose(inst)
				}
			}
		}()

		ww.backend.run(inst)
		close(wwDone)
		ww.backend.destroy(inst)

		windowLogger.Info().Msg("Window closed")
	}
}
