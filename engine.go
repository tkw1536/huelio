package huelio

import (
	"sync"
	"sync/atomic"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
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

func NewEngine(bridge *huego.Bridge) *Engine {
	engine := &Engine{}
	if bridge != nil {
		engine.SetBridge(bridge)
	}
	return engine
}

func (engine *Engine) RefreshIndex() error {
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
func (engine *Engine) Query(input string) ([]QueryAction, error) {

	engine.l.RLock()
	defer engine.l.RUnlock()

	if engine.bridge == nil {
		return engine.linkSpecial(input), nil
	}

	if engine.index == nil {
		return nil, ErrEngineMissingIndex
	}

	if engine.indexErr != nil {
		return nil, engine.indexErr
	}

	return engine.index.QueryString(input), nil
}

// Do performs the provided action
func (engine *Engine) Do(action QueryAction) error {
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

func init() {
	linkAction.ID = "link"
	linkAction.Data.Message = "Link Hue Bridge"
}

func (engine *Engine) linkSpecial(input string) []QueryAction {
	return []QueryAction{{Special: &linkAction}}
}
