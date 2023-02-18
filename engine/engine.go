// Package engine provides Engine and Index
package engine

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Engine struct {
	readOnly uint32       // actions should be considered read-onlt
	l        sync.RWMutex // m protects the rest of this struct

	Ctx context.Context

	// Connect is a user-defined function to connect to a bridge
	Connect func() (bridge *huego.Bridge, err error)

	bridge *huego.Bridge // bridge is the current bridge

	index    *Index
	indexErr error
}

// NewEngine creates a new engine with the given context and bridge.
// If bridge is nil, the bridge is not set.
func NewEngine(bridge *huego.Bridge, ctx context.Context) *Engine {
	engine := &Engine{
		Ctx: ctx,
	}
	if bridge != nil {
		engine.SetBridge(bridge)
	}
	return engine
}

// RefreshIndex refreshes the index on this engine
func (engine *Engine) RefreshIndex() (err error) {
	if cerr := engine.Ctx.Err(); cerr != nil {
		return cerr
	}

	logger := zerolog.Ctx(engine.Ctx)

	logger.Info().Msg("refreshing index")
	defer func() {
		if err != nil {
			logger.Error().Err(err).Msg("index refresh failed")
		} else {
			logger.Info().Msg("index refreshed")
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

	index, indexErr := NewIndex(bridge, engine.Ctx)

	engine.l.Lock()
	defer engine.l.Unlock()

	engine.index = &index
	engine.indexErr = indexErr

	return engine.indexErr
}

func (engine *Engine) SetBridge(bridge *huego.Bridge) {
	if bridge == nil {
		panic("SetBridge: bridge is nil")
	}

	atomic.StoreUint32(&engine.readOnly, 1)

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
func (engine *Engine) Query(input string) ([]Action, []BufferScore, []Score, error) {

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
	engine.logDo(action)

	var writelock bool
	if atomic.LoadUint32(&engine.readOnly) == 0 {
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

func (engine *Engine) logDo(action Action) error {
	bytes, err := json.Marshal(action)
	if err != nil {
		return err
	}

	zerolog.Ctx(engine.Ctx).Info().RawJSON("action", bytes).Msg("performing action")
	return nil
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

	atomic.StoreUint32(&engine.readOnly, 1)
	engine.bridge = bridge

	go engine.RefreshIndex()

	return nil
}

var linkAction HueSpecial
var linkScores Score
var linkMatchScore BufferScore

func init() {
	linkAction.ID = "link"
	linkAction.Data.Message = "Link Hue Bridge"
}

func (engine *Engine) linkSpecial(input string) ([]Action, []BufferScore, []Score) {
	return []Action{{Special: &linkAction}}, []BufferScore{linkMatchScore}, []Score{linkScores}
}
