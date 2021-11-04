package huelio

import (
	"sync"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
)

// Engine coordinates an engine and index regeneration
type Engine struct {
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
		return engine.connectSpecial(input), nil
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
	engine.l.Lock()
	defer engine.l.Unlock()

	if action.Special != nil {
		return engine.doSpecial(action.Special)
	}

	return action.Do(engine.bridge)
}

var ErrEngineInvalidSpecial = errors.New("Engine: invalid special action")
var ErrEngineNoConnect = errors.New("Engine: No Connect function")

func (engine *Engine) doSpecial(special *HueSpecial) error {
	if special.ID != connectAction.ID {
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
	engine.bridge = bridge

	go engine.RefreshIndex()

	return nil
}

var connectAction HueSpecial

func init() {
	connectAction.ID = "connect"
	connectAction.Data.Message = "Connect to Hue Bridge"
}

func (engine *Engine) connectSpecial(input string) []QueryAction {
	return []QueryAction{{Special: &connectAction}}
}
