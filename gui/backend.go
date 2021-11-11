package gui

import "github.com/webview/webview"

// WindowBackend represents a backend for creating and managing windows.
type WindowBackend interface {
	// create creates a new window
	// create must be called on the main thread.
	create(params WindowParams) (interface{}, error)

	// run runs the main loop of this window.
	// run must be called from the main thread and must be followed by a call to destroy().
	run(interface{})

	// destroy destroys the window.
	// It must be called from the main thread
	destroy(interface{})

	// requestClose requests the window to be closed.
	// may only be called while run has not returned.
	requestClose(interface{})

	// requestFocus requests the window to be focused.
	// may only be called while run has not returned.
	requestFocus(interface{})
}

// WebView is a WindowBackend that uses the WebView library
var WebView WindowBackend = webviewbackend{}

type webviewbackend struct{}

func (webviewbackend) create(params WindowParams) (interface{}, error) {
	w := webview.New(false)
	w.SetTitle(params.Title)
	w.SetSize(800, 600, webview.HintFixed)
	w.Navigate(params.URL)
	return w, nil
}

func (webviewbackend) run(ww interface{}) {
	ww.(webview.WebView).Run()
}

func (webviewbackend) destroy(ww interface{}) {
	ww.(webview.WebView).Destroy()
}

func (webviewbackend) requestClose(ww interface{}) {
	ww.(webview.WebView).Eval("window.close()")
}

func (webviewbackend) requestFocus(ww interface{}) {
	ww.(webview.WebView).Eval("window.focus()")
}
