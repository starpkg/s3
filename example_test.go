package s3

import (
	"testing"

	"github.com/1set/starlet"
	"github.com/starpkg/base"
)

// TestStarlarkScripts runs Starlark test scripts from the test directory.
// Scripts with "test-" prefix should succeed, "panic-" prefix should fail.
func TestStarlarkScripts(t *testing.T) {
	// Create a module factory function that returns a fresh module loader for each test
	moduleFactory := func() starlet.ModuleLoader {
		return NewModule().LoadModule()
	}
	extraModules := []string{"go_idiomatic", "http", "json", "file", "path"}

	// Use the helper function from the base package
	base.RunStarlarkTests(t, ModuleName, moduleFactory, extraModules, "")
}

func TestWebModule(t *testing.T) {
	// Test that the module can be created and loaded
	module := NewModule()
	if module == nil {
		t.Fatal("Failed to create web module")
	}

	loader := module.LoadModule()
	if loader == nil {
		t.Fatal("Failed to load web module")
	}
}
