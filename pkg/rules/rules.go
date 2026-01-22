package rules

import (
	"fmt"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// Engine defines the rule engine interface.
type Engine interface {
	// Execute executes the rules on the data and returns the modified data (or nil if dropped).
	Execute(gateway string, data []byte) ([]byte, error)
	// Close closes the engine.
	Close() error
}

// LuaEngine implements a Lua-based rule engine.
type LuaEngine struct {
	mu sync.Mutex
	L  *lua.LState
}

// NewLuaEngine creates a new Lua rule engine.
func NewLuaEngine(scriptPath string) (*LuaEngine, error) {
	L := lua.NewState()

	// Open standard libs
	L.OpenLibs()

	// Load script
	if err := L.DoFile(scriptPath); err != nil {
		L.Close()
		return nil, err
	}

	return &LuaEngine{
		L: L,
	}, nil
}

// Execute runs the 'on_message' function in Lua.
func (e *LuaEngine) Execute(gateway string, data []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	L := e.L

	// Check if function exists
	fn := L.GetGlobal("on_message")
	if fn.Type() != lua.LTFunction {
		// No hook defined, pass through
		return data, nil
	}

	// Push function and arguments
	L.Push(fn)
	L.Push(lua.LString(gateway))
	L.Push(lua.LString(string(data))) // Assuming string data for simple manipulation

	// Call function (2 args, 1 return)
	if err := L.PCall(2, 1, nil); err != nil {
		return nil, fmt.Errorf("lua execution error: %w", err)
	}

	// Get return value
	ret := L.Get(-1) // returned value
	L.Pop(1)         // remove received value

	if ret.Type() == lua.LTNil {
		return nil, nil // Drop message
	}

	if ret.Type() == lua.LTString {
		return []byte(ret.String()), nil
	}

	return data, nil // Default pass through if return type is unexpected
}

// Close closes the Lua state.
func (e *LuaEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.L.Close()
	return nil
}
