package cmd

import (
	"fmt"

	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/spf13/cobra"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Manage saved test targets",
}

var targetCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a saved target",
	Args:  cobra.ExactArgs(1),
	RunE:  createTarget,
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved targets",
	RunE:  listTargets,
}

func init() {
	targetCreateCmd.Flags().String("type", "", "Target type: frontend or backend")
	targetCreateCmd.Flags().String("path", ".", "Project path for this target")
	targetCreateCmd.Flags().String("url", "", "Frontend URL to test")
	targetCreateCmd.Flags().String("curl", "", "Sample curl request for backend testing")
	targetCreateCmd.Flags().String("expectation", "", "Primary positive flow or expected outcome")
	targetCreateCmd.Flags().String("scenario-set", "default", "Default scenario set name")

	targetCmd.AddCommand(targetCreateCmd)
	targetCmd.AddCommand(targetListCmd)
}

func createTarget(cmd *cobra.Command, args []string) error {
	targetPath, err := resolvePath(mustGetString(cmd, "path"))
	if err != nil {
		return err
	}

	target := &catalog.Target{
		Name:        args[0],
		Type:        mustGetString(cmd, "type"),
		Path:        targetPath,
		URL:         mustGetString(cmd, "url"),
		Curl:        mustGetString(cmd, "curl"),
		Expectation: mustGetString(cmd, "expectation"),
		ScenarioSet: mustGetString(cmd, "scenario-set"),
	}
	if err := catalog.SaveTarget(target); err != nil {
		return err
	}

	fmt.Printf("Saved target %q\n", target.Name)
	fmt.Printf("Type: %s\n", target.Type)
	if target.URL != "" {
		fmt.Printf("URL: %s\n", target.URL)
	}
	if target.Expectation != "" {
		fmt.Printf("Expectation: %s\n", target.Expectation)
	}
	return nil
}

func listTargets(cmd *cobra.Command, args []string) error {
	targets, err := catalog.ListTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No saved targets yet. Create one with: suitest target create <name> ...")
		return nil
	}

	for _, target := range targets {
		fmt.Printf("- %s [%s]\n", target.Name, target.Type)
		if target.URL != "" {
			fmt.Printf("  url: %s\n", target.URL)
		}
		fmt.Printf("  path: %s\n", target.Path)
		fmt.Printf("  scenario set: %s\n", target.ScenarioSet)
	}
	return nil
}

func mustGetString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}
