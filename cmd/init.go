package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/runners"
	"github.com/mulhamna/suitest/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const suitestBanner = `
   _____       _ __           __
  / ___/__  __(_) /____  ____/ /_
  \__ \/ / / / / __/ _ \/ __  __/
 ___/ / /_/ / / /_/  __/ /_/ /
/____/\__,_/_/\__/\___/\__,_/

AI-powered testing agent
`

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize suitest in a project",
	Long:  `Set up global settings and save an initial reusable test target for this project.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absPath, err := resolvePath(targetPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	detectedMode, err := runners.Detect(absPath)
	if err != nil {
		detectedMode = "auto"
	}

	fmt.Print(suitestBanner)
	fmt.Println("Welcome to suitest.")
	fmt.Println("Let's set up your testing workspace.")
	fmt.Println()

	if err := runInteractiveInit(cfg, absPath, detectedMode); err != nil {
		return err
	}

	return nil
}

func runInteractiveInit(cfg *config.Config, absPath, detectedMode string) error {
	targetNameDefault := filepath.Base(absPath)
	if targetNameDefault == "." || targetNameDefault == string(filepath.Separator) {
		targetNameDefault = "default-target"
	}

	projectTypeDefault := defaultProjectType(detectedMode)
	if cfg.Mode == "browser" {
		projectTypeDefault = "frontend"
	} else if cfg.Mode == "api" {
		projectTypeDefault = "backend"
	}

	storageDriver, err := promptChoice("[1/5] Storage", []string{"json", "sqlite"}, fallback(cfg.Storage.Driver, "json"))
	if err != nil {
		return err
	}
	operatorMode, err := promptChoice("[2/5] Operator", []string{"native", "cli-agent"}, fallback(cfg.Operator.Mode, "native"))
	if err != nil {
		return err
	}
	projectType, err := promptChoice("[3/5] Project type", []string{"frontend", "backend"}, projectTypeDefault)
	if err != nil {
		return err
	}
	targetName, err := promptText("[4/5] Target name", targetNameDefault)
	if err != nil {
		return err
	}

	var entryURL string
	var seedCurl string
	if projectType == "frontend" {
		entryURL, err = promptText("Frontend URL", fallback(cfg.EntryURL, "http://localhost:3000"))
		if err != nil {
			return err
		}
	} else {
		seedCurl, err = promptText("Sample curl command", "")
		if err != nil {
			return err
		}
	}

	expectation, err := promptText("[5/5] Expected success flow", "")
	if err != nil {
		return err
	}

	cfg.Storage.Driver = storageDriver
	cfg.Operator.Mode = operatorMode
	if cfg.Storage.Driver == "sqlite" {
		if err := storage.ValidateSQLiteConfig(); err != nil {
			return err
		}
	}
	if err := config.SaveGlobal(cfg); err != nil {
		return err
	}

	projectMode := "api"
	if projectType == "frontend" {
		projectMode = "browser"
	}
	if err := saveProjectConfig(absPath, projectMode, entryURL); err != nil {
		return err
	}

	target := &catalog.Target{
		Name:        targetName,
		Type:        projectType,
		Path:        absPath,
		URL:         entryURL,
		Curl:        seedCurl,
		Expectation: expectation,
		ScenarioSet: "default",
	}
	if err := catalog.SaveTarget(target); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Setup complete.")
	fmt.Printf("Global config saved to %s\n", filepath.Join(userHomeDir(), ".suitest", "config.yaml"))
	fmt.Printf("Project config saved to %s\n", filepath.Join(absPath, ".suitest.yaml"))
	fmt.Printf("Saved target %q\n", target.Name)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. suitest scenario map %s\n", target.Name)
	fmt.Printf("  2. suitest run %s\n", target.Name)
	fmt.Printf("  3. suitest settings\n")
	return nil
}

func saveProjectConfig(absPath, mode, entryURL string) error {
	projectConfig := map[string]interface{}{
		"mode":     mode,
		"provider": "auto",
		"test_dir": "./tests",
	}
	if mode == "browser" && entryURL != "" {
		projectConfig["entry_url"] = entryURL
	}

	data, err := yaml.Marshal(projectConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(absPath, ".suitest.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func defaultProjectType(mode string) string {
	switch mode {
	case "browser":
		return "frontend"
	case "api":
		return "backend"
	default:
		return "frontend"
	}
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~"
	}
	return home
}
