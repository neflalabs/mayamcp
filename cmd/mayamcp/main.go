// Command mayamcp serves the MCP core over stdio (default) or Streamable HTTP.
// Authentication and TLS are left to the embedding application or proxy:
// the template deliberately keeps the transport layer thin.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/app"
)

func main() {
	transport := flag.String("transport", "stdio", "transport: stdio (default) or http")
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address (http transport only)")
	name := flag.String("name", "mayamcp", "server identity name")
	version := flag.String("version", "dev", "server version")
	flag.Parse()

	core := app.NewCore(app.Config{Name: *name, Version: *version})

	switch *transport {
	case "stdio":
		// stdout is reserved for the MCP protocol; the logger writes to stderr.
		if err := core.Server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "http":
		handler := mcp.NewStreamableHTTPHandler(
			func(*http.Request) *mcp.Server { return core.Server },
			&mcp.StreamableHTTPOptions{Logger: core.Logger, Stateless: true},
		)
		core.Logger.Info("serving MCP over HTTP", "addr", *addr)
		if err := http.ListenAndServe(*addr, handler); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown transport %q (want stdio or http)\n", *transport)
		os.Exit(2)
	}
}
