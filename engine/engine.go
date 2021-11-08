package engine

import (
	"sync"
	"sync/atomic"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tkw1536/huelio/logging"
)

// Engine coordinates an engine and index regeneration
type Engine struct {
	noWritableAction uint32 // do we need a wriable lock to do actions?

	l sync.RWMutex // m protects the rest of this struct

	// Connect is a user-defined function to connect to a bridge
	Connect func() (bridge *huego.Bridge, err error)

	bridge *huego.Bridge // bridge is the current bridge

	index    *Index
	indexErr error
}

var engineLogger zerolog.Logger

func init() {
	logging.ComponentLogger("engine.Engine", &engineLogger)
}

func NewEngine(bridge *huego.Bridge) *Engine {
	engine := &Engine{}
	if bridge != nil {
		engine.SetBridge(bridge)
	}
	return engine
}

func (engine *Engine) RefreshIndex() (err error) {
	engineLogger.Info().Msg("refreshing index")
	defer func() {
		if err != nil {
			engineLogger.Error().Err(err).Msg("index refresh failed")
		} else {
			engineLogger.Info().Msg("index refreshed")
		}
	}()

	bridge, err := func() (*huego.Bridge, error) {
		engine.l.RLock()
		defer engine.l.RUnlock()

		if engine.bridge == nil {
			return nil, ErrEngineMissingBridge
		}

		return engine.bridge, nil
	}()
	if err != nil {
		return err
	}

	index, indexErr := NewIndex(bridge)

	engine.l.Lock()
	defer engine.l.Unlock()

	engine.index = index
	engine.indexErr = indexErr

	return engine.indexErr
}

func (engine *Engine) SetBridge(bridge *huego.Bridge) {
	if bridge == nil {
		panic("SetBridge: bridge is nil")
	}

	atomic.StoreUint32(&engine.noWritableAction, 1)

	engine.l.Lock()
	defer engine.l.Unlock()

	engine.bridge = bridge
	engine.index = nil
	engine.indexErr = nil

	go engine.RefreshIndex()
}

var ErrEngineMissingIndex = errors.New("Engine: missing index")
var ErrEngineMissingBridge = errors.New("Engine: missing index")

// Query queries the engine
func (engine *Engine) Query(input string) ([]Action, []MatchScore, []Score, error) {

	engine.l.RLock()
	defer engine.l.RUnlock()

	if engine.bridge == nil {
		actions, matches, scores := engine.linkSpecial(input)
		return actions, matches, scores, nil
	}

	if engine.index == nil {
		return nil, nil, nil, ErrEngineMissingIndex
	}

	if engine.indexErr != nil {
		return nil, nil, nil, engine.indexErr
	}

	actions, matches, scores := engine.index.QueryString(input)
	return actions, matches, scores, nil
}

// Do performs the provided action
func (engine *Engine) Do(action Action) error {
	var writelock bool
	if atomic.LoadUint32(&engine.noWritableAction) == 0 {
		writelock = true

		engine.l.Lock()
		defer engine.l.Unlock()
	} else {
		engine.l.RLock()
		defer engine.l.RUnlock()
	}

	if action.Special != nil {
		return engine.doSpecial(action.Special, writelock)
	}

	return action.Do(engine.bridge)
}

var ErrEngineInvalidSpecial = errors.New("Engine: invalid special action")
var ErrEngineNoConnect = errors.New("Engine: No Connect function")

func (engine *Engine) doSpecial(special *HueSpecial, writeLock bool) error {
	switch special.ID {
	case linkAction.ID:
		return engine.linkInternal(true)
	}
	return ErrEngineInvalidSpecial
}

// Link links the engine
func (engine *Engine) Link() error {
	engine.l.Lock()
	defer engine.l.Unlock()

	return engine.linkInternal(true)
}

func (engine *Engine) linkInternal(writeLock bool) error {
	if !writeLock {
		// this action requires a write lock
		// so don't do it unless you still have one!
		return ErrEngineInvalidSpecial
	}

	if engine.bridge != nil {
		return nil
	}

	if engine.Connect == nil {
		return ErrEngineNoConnect
	}

	bridge, err := engine.Connect()
	if err != nil {
		return err
	}

	atomic.StoreUint32(&engine.noWritableAction, 1)
	engine.bridge = bridge

	go engine.RefreshIndex()

	return nil
}

var linkAction HueSpecial
var linkScores Score
var linkMatchScore MatchScore

func init() {
	linkAction.ID = "link"
	linkAction.Data.Message = "Link Hue Bridge"
}

func (engine *Engine) linkSpecial(input string) ([]Action, []MatchScore, []Score) {
	return []Action{{Special: &linkAction}}, []MatchScore{linkMatchScore}, []Score{linkScores}
}
