package geeorm

import (
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func OpenDB(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine("sqlite", filepath.Join(t.TempDir(), "gee.db"))
	if err != nil {
		t.Fatal("failed to connect", err)
	}
	return engine
}

func TestNewEngine(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
}
