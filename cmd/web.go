package cmd

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/mulhamna/suitest/internal/api"
	"github.com/mulhamna/suitest/internal/jobs"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the local web UI and API for suitest",
	Long: `Start suitest in local web mode.

This launches the local HTTP API used by the optional Svelte web UI. In dev
mode, run the frontend separately with 'make dev-frontend'.`,
	RunE: runWeb,
}

func init() {
	webCmd.Flags().Int("port", 4020, "Port to listen on")
	webCmd.Flags().Bool("open", false, "Open the browser after startup")
	webCmd.Flags().Bool("dev", false, "Run API only for local frontend development")
	rootCmd.AddCommand(webCmd)
}

func runWeb(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	openBrowser, _ := cmd.Flags().GetBool("open")
	devMode, _ := cmd.Flags().GetBool("dev")

	jobManager := jobs.NewManager()
	server := api.NewServer(jobManager)
	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	if openBrowser {
		go openURL(url)
	}

	fmt.Printf("Starting suitest web API on %s\n", addr)
	if devMode {
		fmt.Println("Dev mode enabled. Run the Svelte frontend separately with 'make dev-frontend'.")
	}
	fmt.Printf("Health check: %s/api/health\n", url)

	return http.ListenAndServe(addr, server.Handler())
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
