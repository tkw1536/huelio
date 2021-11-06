package creds

import (
	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio/logging"
)

// Finder finds credentials on a Hue Bridge
type Finder struct {
	NewName string

	Hostname string
	Username string
}

var finderLogger zerolog.Logger

func init() {
	logging.ComponentLogger("creds.Finder", &finderLogger)
}

func (pf Finder) Find() (*Credentials, error) {
	if pf.Hostname != "" && pf.Username != "" {
		finderLogger.Info().Str("hostname", pf.Hostname).Msg("using provided credentials")

		return &Credentials{
			Hostname: pf.Hostname,
			Username: pf.Username,
		}, nil
	}

	var bridge *huego.Bridge
	if pf.Hostname == "" {
		finderLogger.Info().Msg("looking for bridges")
		var err error
		bridge, err = huego.Discover()
		if err != nil {
			finderLogger.Error().Err(err).Msg("unable to discover bridge to bridge")
			return nil, errors.Wrap(err, "unable to discover bridge")
		}
	} else {
		finderLogger.Info().Str("hostname", pf.Hostname).Msg("using provided bridge")
		bridge = huego.New(pf.Hostname, "")
	}

	finderLogger.Info().Str("hostname", pf.Hostname).Str("username", pf.NewName).Msg("creating new user for bridge")
	user, err := bridge.CreateUser(pf.NewName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create new user")
	}

	finderLogger.Info().Str("username", user).Msg("using new created user")

	return &Credentials{
		Hostname: bridge.Host,
		Username: user,
	}, nil
}
