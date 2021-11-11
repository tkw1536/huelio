// Package gui contains bindings for the huelio gui
package gui

import (
	"context"
	"strings"

	hook "github.com/robotn/gohook"
)

// KeyCombiantion represents a keyboard combination
type KeyCombination []string

// NewKeyCombination parses a key combination from string
func NewKeyCombination(value string) KeyCombination {
	keys := strings.Split(strings.ToLower(value), "+")
	code := keys[:0]
	for _, k := range keys {
		if _, ok := hook.Keycode[k]; ok {
			code = append(code, k)
		}
	}
	return KeyCombination(code)
}

// NewComboListener creates a new system-wide lisenter that triggers on keydown events for the provided combination.
// The Listener is scheduled on a new goroutine stops as soon as ctx is cancelled.
//
// Whenever the listener triggers, NewComboListener attempts to send a message to the provided channel.
// When no reader is trying to receive, hook will ignore the trigger.
//
// Only one ComboListener can be active at the same time, globally.
// Therefore it is recommended to only call this function from a main method.
func NewComboListener(combo KeyCombination, ctx context.Context) <-chan struct{} {
	listener := make(chan struct{})

	// register the combination
	hook.Register(hook.KeyDown, combo, func(e hook.Event) {
		select {
		case listener <- struct{}{}:
		default:
		}
	})

	// start listening
	hook.Process(hook.Start())

	// register a cleanup function
	go func() {
		<-ctx.Done()
		close(listener)
		hook.End()
	}()
	return listener
}
