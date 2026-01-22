package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/dop251/goja"
)

// JSEngine implements a JavaScript-based rule engine using goja.
type JSEngine struct {
	mu      sync.Mutex
	vm      *goja.Runtime
	onMsg   goja.Callable
	console *jsConsole
}

// jsConsole provides console.log functionality.
type jsConsole struct {
	logs []string
}

func (c *jsConsole) Log(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logs = append(c.logs, msg)
	// Also print to stdout for debugging
	fmt.Println("[JS]", msg)
}

func (c *jsConsole) Warn(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logs = append(c.logs, "WARN: "+msg)
	fmt.Println("[JS WARN]", msg)
}

func (c *jsConsole) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logs = append(c.logs, "ERROR: "+msg)
	fmt.Println("[JS ERROR]", msg)
}

// NewJSEngine creates a new JavaScript rule engine.
func NewJSEngine(script string) (*JSEngine, error) {
	vm := goja.New()

	// Create console object
	console := &jsConsole{}
	consoleObj := vm.NewObject()
	consoleObj.Set("log", console.Log)
	consoleObj.Set("warn", console.Warn)
	consoleObj.Set("error", console.Error)
	vm.Set("console", consoleObj)

	// Add JSON helpers
	vm.Set("JSON", map[string]interface{}{
		"parse": func(s string) (interface{}, error) {
			var result interface{}
			err := json.Unmarshal([]byte(s), &result)
			return result, err
		},
		"stringify": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
	})

	// Add utility functions
	vm.Set("hexToBytes", func(hex string) []byte {
		// Simple hex decoder
		result := make([]byte, len(hex)/2)
		for i := 0; i < len(hex)/2; i++ {
			fmt.Sscanf(hex[i*2:i*2+2], "%02x", &result[i])
		}
		return result
	})

	vm.Set("bytesToHex", func(b []byte) string {
		result := ""
		for _, v := range b {
			result += fmt.Sprintf("%02x", v)
		}
		return result
	})

	// Run the script
	_, err := vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("script error: %w", err)
	}

	// Get on_message function
	onMsgVal := vm.Get("on_message")
	var onMsg goja.Callable
	if onMsgVal != nil && !goja.IsUndefined(onMsgVal) {
		fn, ok := goja.AssertFunction(onMsgVal)
		if ok {
			onMsg = fn
		}
	}

	return &JSEngine{
		vm:      vm,
		onMsg:   onMsg,
		console: console,
	}, nil
}

// NewJSEngineFromFile creates a JS engine from a file path.
func NewJSEngineFromFile(scriptPath string) (*JSEngine, error) {
	content, err := readFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	return NewJSEngine(string(content))
}

// readFile reads a file
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Execute runs the 'on_message' function in JavaScript.
func (e *JSEngine) Execute(gateway string, data []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.onMsg == nil {
		// No hook defined, pass through
		return data, nil
	}

	// Call on_message(gateway, data)
	result, err := e.onMsg(goja.Undefined(), e.vm.ToValue(gateway), e.vm.ToValue(string(data)))
	if err != nil {
		return nil, fmt.Errorf("js execution error: %w", err)
	}

	// Handle return value
	if goja.IsNull(result) || goja.IsUndefined(result) {
		return nil, nil // Drop message
	}

	// Convert result to string/bytes
	exported := result.Export()
	switch v := exported.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case nil:
		return nil, nil
	default:
		// Try JSON serialization
		b, err := json.Marshal(v)
		if err != nil {
			return data, nil // Pass through on error
		}
		return b, nil
	}
}

// Close closes the JS engine.
func (e *JSEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// goja doesn't need explicit cleanup
	e.vm = nil
	e.onMsg = nil
	return nil
}

// GetLogs returns console logs from the script.
func (e *JSEngine) GetLogs() []string {
	return e.console.logs
}

// RunScript executes arbitrary JavaScript code.
func (e *JSEngine) RunScript(script string) (interface{}, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result, err := e.vm.RunString(script)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

// SetGlobal sets a global variable in the JS context.
func (e *JSEngine) SetGlobal(name string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vm.Set(name, value)
}

// CallFunction calls a named function with arguments.
func (e *JSEngine) CallFunction(name string, args ...interface{}) (interface{}, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	fnVal := e.vm.Get(name)
	if fnVal == nil || goja.IsUndefined(fnVal) {
		return nil, fmt.Errorf("function %s not found", name)
	}

	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", name)
	}

	// Convert args to goja values
	gojaArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		gojaArgs[i] = e.vm.ToValue(arg)
	}

	result, err := fn(goja.Undefined(), gojaArgs...)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}
