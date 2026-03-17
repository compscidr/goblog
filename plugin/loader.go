package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// LoadDynamicPlugins scans a directory for .go plugin files and loads them
// using the Yaegi Go interpreter. Each file must use `package main` and
// define a function:
//
//	func NewPlugin() plugin.Plugin
//
// The returned plugins are registered with the given registry.
func LoadDynamicPlugins(registry *Registry, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: could not read plugin directory %s: %v", dir, err)
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		p, err := loadPlugin(path)
		if err != nil {
			log.Printf("Warning: failed to load plugin %s: %v", entry.Name(), err)
			continue
		}
		registry.Register(p)
	}
}

func loadPlugin(path string) (Plugin, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}
	// Export the plugin package symbols so dynamic plugins can use them
	if err := i.Use(Symbols); err != nil {
		return nil, err
	}

	_, err = i.Eval(string(src))
	if err != nil {
		return nil, err
	}

	v, err := i.Eval("NewPlugin()")
	if err != nil {
		return nil, err
	}

	p, ok := v.Interface().(Plugin)
	if !ok {
		return nil, fmt.Errorf("%s: NewPlugin() did not return a plugin.Plugin", path)
	}

	return p, nil
}
