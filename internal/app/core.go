// Package app builds the MayaMCP core without choosing a transport.
package app

import (
	"io"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/mcpserver"
)

// Config is the minimal input needed to build the core.
type Config struct {
	Name    string
	Version string
	Logger  *slog.Logger
	// LogWriter, when set, receives structured logs instead of stderr.
	LogWriter io.Writer
}

// Core is the reusable runtime; a front end can serve Core.Server over stdio
// or Streamable HTTP.
type Core struct {
	Server *mcp.Server
	Host   mcpserver.Host
	Logger *slog.Logger
}

type staticHost struct {
	name    string
	version string
}

func (h staticHost) Name() string    { return h.name }
func (h staticHost) Version() string { return h.version }

// NewCore prepares the MCP server and its domain host without starting a
// listener or wiring any front end.
func NewCore(cfg Config) *Core {
	logger := cfg.Logger
	if cfg.LogWriter != nil {
		logger = slog.New(slog.NewTextHandler(cfg.LogWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
	} else if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	name := cfg.Name
	if name == "" {
		name = "mayamcp"
	}
	version := cfg.Version
	if version == "" {
		version = "dev"
	}
	host := staticHost{name: name, version: version}
	server := mcpserver.New(host, version, mcpserver.Options{
		Name:   name,
		Logger: logger,
	})
	return &Core{Server: server, Host: host, Logger: logger}
}
