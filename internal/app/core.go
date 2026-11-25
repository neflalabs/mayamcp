// Package app builds the MayaMCP core without choosing a transport.
package app

import (
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/internal/mcpserver"
)

// Config is the minimal input needed to build the core.
type Config struct {
	Name    string
	Version string
	// Title is the human-readable server title. Empty uses the mcpserver default.
	Title string
	// Instructions replaces the default usage hint sent to clients on initialize.
	Instructions string
	// TargetExtractor supplies domain-specific activity-log targets. Empty uses
	// the generic extractor in mcpserver.
	TargetExtractor mcpserver.TargetExtractor
	// Logger is used as-is when LogWriter is nil. LogLevel and LogJSON only
	// affect loggers built by NewCore.
	Logger *slog.Logger
	// LogWriter, when set, receives structured logs instead of stderr and
	// overrides Logger. When neither is set, logs go to stderr.
	LogWriter io.Writer
	// LogLevel overrides the default info level for loggers built here.
	LogLevel slog.Leveler
	// LogJSON switches loggers built here from text to JSON handler.
	LogJSON bool
	// Register replaces the starter tools when set.
	Register func(*mcp.Server)
	// RegisterResources adds consumer resources when set.
	RegisterResources func(*mcp.Server)
	// Middleware is added before the activity middleware.
	Middleware []mcp.Middleware
	// DisableActivityMiddleware lets a consumer install its own activity
	// middleware while keeping identity, logging and hooks from the core.
	DisableActivityMiddleware bool
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
	logger := mcpserver.NewLogger(mcpserver.LogOptions{
		Writer: cfg.LogWriter,
		Logger: cfg.Logger,
		Level:  cfg.LogLevel,
		JSON:   cfg.LogJSON,
	})
	name := cfg.Name
	if name == "" {
		name = mcpserver.DefaultName
	}
	version := cfg.Version
	if version == "" {
		version = mcpserver.DefaultVersion
	}
	host := staticHost{name: name, version: version}
	opts := mcpserver.Options{
		Name:                      name,
		Title:                     cfg.Title,
		Instructions:              cfg.Instructions,
		Version:                   version,
		TargetExtractor:           cfg.TargetExtractor,
		Logger:                    logger,
		Register:                  cfg.Register,
		RegisterResources:         cfg.RegisterResources,
		Middleware:                cfg.Middleware,
		DisableActivityMiddleware: cfg.DisableActivityMiddleware,
	}
	server := mcpserver.New(host, version, opts)
	return &Core{Server: server, Host: host, Logger: logger}
}
