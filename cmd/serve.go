package cmd

import (
	"fmt"

	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server for IDE integration",
	Long: `Start the Model Context Protocol (MCP) server, which exposes suitest
capabilities to AI IDEs like Claude Code, Cursor, and Windsurf.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().Int("port", 3100, "Port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Starting suitest MCP server on port %d\n", port)
	fmt.Printf("Add to your MCP config:\n")
	fmt.Printf(`  {
    "mcpServers": {
      "suitest": {
        "command": "suitest",
        "args": ["serve"]
      }
    }
  }
`)

	server := mcp.NewServer(cfg, port)
	return server.Start()
}
