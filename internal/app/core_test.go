package app

import "testing"

func TestNewCoreWithoutTransport(t *testing.T) {
	core := NewCore(Config{Name: "test-server", Version: "v1"})
	if core.Server == nil || core.Host == nil || core.Logger == nil {
		t.Fatal("core is missing server, host or logger")
	}
	if core.Host.Name() != "test-server" || core.Host.Version() != "v1" {
		t.Fatalf("host = %s %s", core.Host.Name(), core.Host.Version())
	}
}

func TestNewCoreDefaults(t *testing.T) {
	core := NewCore(Config{})
	if core.Host.Name() != "mayamcp" || core.Host.Version() != "dev" {
		t.Fatalf("defaults = %s %s", core.Host.Name(), core.Host.Version())
	}
}
