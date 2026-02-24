package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rackspace-spot/spotctl/internal/mcpserver"
	"github.com/spf13/cobra"
)

var (
	mcpTransport string
	mcpAddr      string
)

// mcpCmd runs spotctl as an MCP server.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run spotctl as a Model Context Protocol (MCP) server",
	Long: `Run spotctl as an MCP server over stdio or HTTP.

Examples:
  # Run as a stdio MCP server
  spotctl mcp --transport stdio

  # Run as a streamable HTTP MCP server
  spotctl mcp --transport http --addr localhost:8080
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		switch mcpTransport {
		case "stdio":
			return mcpserver.RunStdio(ctx)
		case "http":
			if mcpAddr == "" {
				return fmt.Errorf("--addr is required when --transport=http")
			}
			return mcpserver.RunHTTP(ctx, mcpAddr)
		default:
			return fmt.Errorf("unsupported transport %q (expected \"stdio\" or \"http\")", mcpTransport)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "MCP transport to use (stdio or http)")
	mcpCmd.Flags().StringVar(&mcpAddr, "addr", "localhost:8080", "Address for HTTP MCP server (host:port)")
}

