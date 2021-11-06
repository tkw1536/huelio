// Package logging provides internal logging factilities for huelio
package logging

import (
	"sync"

	"github.com/rs/zerolog"
)

var globalMutex sync.Mutex
var globalLogger *zerolog.Logger
var localLoggers = make(map[string]*zerolog.Logger)

// Init configures the global logger.
// It should be set from the global init package.
func Init(global *zerolog.Logger) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalLogger != nil {
		panic("logger.Init: Already called")
	}

	// store the global logger
	globalLogger = global
	for name, logger := range localLoggers {
		writeLogger(name, logger)
	}

	// clear the global map, it is no longer needed
	globalLogger = nil
}

func writeLogger(name string, logger *zerolog.Logger) {
	*logger = globalLogger.With().Str("component", name).Logger()
}

// ComponentLogger registers logger to be a logger for the given component
func ComponentLogger(component string, logger *zerolog.Logger) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	// global logging not yet started, so store it!
	if globalLogger == nil {
		localLoggers[component] = logger
		return
	}

	writeLogger(component, logger)
}
