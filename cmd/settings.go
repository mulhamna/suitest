package cmd

import (
	"fmt"

	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/storage"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "View or update global suitest settings",
	RunE:  showSettings,
}

var settingsSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a global setting",
	Args:  cobra.ExactArgs(2),
	RunE:  setSetting,
}

func init() {
	settingsCmd.AddCommand(settingsSetCmd)
}

func showSettings(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("storage.driver: %s\n", cfg.Storage.Driver)
	if path, err := storage.DBPath(); err == nil {
		fmt.Printf("storage.path: %s\n", path)
	}
	if cfg.Storage.Driver == "sqlite" {
		if err := storage.ValidateSQLiteConfig(); err != nil {
			fmt.Printf("storage.status: error (%v)\n", err)
		} else {
			fmt.Println("storage.status: ready")
		}
	}
	fmt.Printf("operator.mode: %s\n", cfg.Operator.Mode)
	fmt.Printf("default_provider: %s\n", cfg.DefaultProvider)
	return nil
}

func setSetting(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	key := args[0]
	value := args[1]
	switch key {
	case "storage.driver":
		if value != "json" && value != "sqlite" {
			return fmt.Errorf("storage.driver must be json or sqlite")
		}
		cfg.Storage.Driver = value
		if value == "sqlite" {
			if err := storage.ValidateSQLiteConfig(); err != nil {
				return err
			}
		}
	case "storage.path":
		cfg.Storage.Path = value
		if cfg.Storage.Driver == "sqlite" {
			if err := storage.ValidateSQLitePath(value); err != nil {
				return err
			}
		}
	case "operator.mode":
		if value != "native" && value != "cli-agent" {
			return fmt.Errorf("operator.mode must be native or cli-agent")
		}
		cfg.Operator.Mode = value
	case "default_provider":
		cfg.DefaultProvider = value
	default:
		return fmt.Errorf("unsupported setting: %s", key)
	}

	if err := config.SaveGlobal(cfg); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", key)
	return nil
}
