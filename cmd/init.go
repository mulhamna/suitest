package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mulhamna/suitest/internal/runners"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize suitest in a project",
	Long:  `Scaffold a .suitest.yaml config file in the project root by inspecting the codebase.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	configPath := filepath.Join(absPath, ".suitest.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf(".suitest.yaml already exists at %s\n", configPath)
		return nil
	}

	// Detect project type
	mode, err := runners.Detect(absPath)
	if err != nil {
		mode = "auto"
	}

	projectConfig := map[string]interface{}{
		"mode":     mode,
		"provider": "auto",
		"test_dir": "./tests",
	}

	if mode == "browser" {
		projectConfig["entry_url"] = "http://localhost:3000"
	}

	data, err := yaml.Marshal(projectConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created .suitest.yaml at %s\n", configPath)
	fmt.Printf("Detected project type: %s\n", mode)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Set your provider in ~/.suitest/config.yaml")
	fmt.Println("     Or set environment variables:")
	fmt.Println("       ANTHROPIC_API_KEY for Claude")
	fmt.Println("       OPENAI_API_KEY for OpenAI")
	fmt.Println("       OPENROUTER_API_KEY for OpenRouter")
	fmt.Println("  2. Run: suitest run .")

	home, _ := os.UserHomeDir()
	globalConfigPath := filepath.Join(home, ".suitest", "config.yaml")
	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		fmt.Println()
		fmt.Printf("No global config found. Creating template at %s\n", globalConfigPath)
		if err := os.MkdirAll(filepath.Dir(globalConfigPath), 0755); err == nil {
			template := `# suitest global configuration
default_provider: auto

providers:
  claude:
    api_key: "${ANTHROPIC_API_KEY}"
    model: claude-sonnet-4-20250514

  openai:
    api_key: "${OPENAI_API_KEY}"
    model: gpt-4o

  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: mistral/mistral-7b-instruct

  ollama:
    base_url: http://localhost:11434
    model: llama3

agent:
  max_retries: 3
  concurrency: 4
  auto_fix: false
`
			os.WriteFile(globalConfigPath, []byte(template), 0644)
		}
	}

	return nil
}
