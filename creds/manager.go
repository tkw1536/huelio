package creds

import (
	"context"
	"sync"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Manager manages credentials for a single Hue Bridge
type Manager struct {
	l sync.Mutex

	Ctx context.Context

	Finder Finder
	Store  Store
}

// Connect connects to a Hue Bridge.
// It is intended to be used by engine.Connect.
func (sm *Manager) Connect() (*huego.Bridge, error) {
	managerLogger := zerolog.Ctx(sm.Ctx).With().Str("component", "creds.Manager").Logger()

	sm.l.Lock()
	defer sm.l.Unlock()

	// if we have credentials in the store, return them!
	managerLogger.Info().Msg("reading credentials from store")
	creds, err := sm.Store.Read()
	if err != nil {
		managerLogger.Error().Err(err).Msg("unable to read stored credentials")
		return nil, errors.Wrap(err, "unable to read stored credentials")
	}
	if creds != nil {
		managerLogger.Info().Msg("using stored credentials")
		return NewBridge(creds)
	}

	// create new credentials
	managerLogger.Info().Msg("finding new credentials")
	credentials, err := sm.Finder.Find()
	if err != nil {
		managerLogger.Error().Err(err).Msg("unable to generate new credentials")
		return nil, errors.Wrap(err, "unable to generate new credentials")
	}

	// make a bridge
	managerLogger.Info().Msg("connecting to bridge")
	bridge, err := NewBridge(credentials)
	if err != nil {
		managerLogger.Error().Err(err).Msg("bridge connection failed")
		return nil, errors.Wrap(err, "bridge connection failed")
	}

	// write the credentials to the store!
	managerLogger.Info().Msg("writing credentials to store")
	if err := sm.Store.Write(credentials); err != nil {
		managerLogger.Error().Err(err).Msg("unable to store credentials")
		return nil, errors.Wrap(err, "unable to store credentials")
	}

	// and return!
	managerLogger.Info().Msg("connection to bridge finished")
	return bridge, nil
}
