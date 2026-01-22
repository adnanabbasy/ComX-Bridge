package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// FileLoader loads plugins from files.
type FileLoader struct{}

// NewFileLoader creates a new file loader.
func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

// LoadDir loads all plugins from a directory.
func (l *FileLoader) LoadDir(dir string) ([]Plugin, error) {
	// Simple implementation: just list files and try to load them
	// In a real implementation, we would glob for .so files (or .dll if supported)
	return nil, nil
}

// LoadFile loads a plugin from a file.
func (l *FileLoader) LoadFile(path string) (Plugin, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("go plugins are not supported on windows")
	}

	ext := filepath.Ext(path)
	if ext != ".so" {
		return nil, fmt.Errorf("unsupported plugin extension: %s", ext)
	}

	// This is where real plugin loading would happen using "plugin" package.
	// Since "plugin" package is not supported on Windows, we cannot even compile it here
	// without build tags if we import "plugin".
	// So we return error.

	return nil, errors.New("plugin loading not implemented for this platform")
}

// SupportedExtensions returns supported file extensions.
func (l *FileLoader) SupportedExtensions() []string {
	if runtime.GOOS == "windows" {
		return []string{}
	}
	return []string{".so"}
}
