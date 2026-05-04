package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/spf13/cobra"
)

var scenarioCmd = &cobra.Command{
	Use:   "scenario",
	Short: "Manage saved scenario sets",
}

var scenarioMapCmd = &cobra.Command{
	Use:   "map [target]",
	Short: "Generate and save scenarios for a target",
	Args:  cobra.ExactArgs(1),
	RunE:  mapScenarioSet,
}

var scenarioListCmd = &cobra.Command{
	Use:   "list [target]",
	Short: "List saved scenarios for a target",
	Args:  cobra.ExactArgs(1),
	RunE:  listScenarioSets,
}

var scenarioApproveCmd = &cobra.Command{
	Use:   "approve [target] [set]",
	Short: "Approve a saved scenario set for execution",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  approveScenarioSet,
}

func init() {
	scenarioMapCmd.Flags().String("name", "default", "Scenario set name")
	scenarioMapCmd.Flags().Bool("replace-default", true, "Update the target's default scenario set")
	scenarioMapCmd.Flags().Bool("yes", false, "Approve the generated scenario set without prompting")
	scenarioMapCmd.Flags().Bool("draft", false, "Save the generated scenario set without approving it")

	scenarioCmd.AddCommand(scenarioMapCmd)
	scenarioCmd.AddCommand(scenarioListCmd)
	scenarioCmd.AddCommand(scenarioApproveCmd)
}

func mapScenarioSet(cmd *cobra.Command, args []string) error {
	target, err := catalog.LoadTarget(args[0])
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := applyRunOverrides(cmd, cfg); err != nil {
		return err
	}

	provider, err := initProvider(cfg)
	if err != nil {
		return err
	}

	discovery, err := agent.DiscoverProject(target.Path)
	if err != nil {
		return err
	}
	discovery.TargetName = target.Name
	discovery.TargetType = target.Type
	discovery.EntryURL = target.URL
	discovery.SeedCurl = target.Curl
	discovery.Expectation = target.Expectation
	if discovery.EntryURL != "" {
		discovery.Summary = fmt.Sprintf("%s | url: %s", discovery.Summary, discovery.EntryURL)
	}
	if discovery.Expectation != "" {
		discovery.Summary = fmt.Sprintf("%s | expected flow: %s", discovery.Summary, discovery.Expectation)
	}

	mode := resolveTargetMode(target)
	planner := agent.NewPlanner(provider)
	plans, err := planner.Plan(context.Background(), discovery, mode)
	if err != nil {
		return err
	}

	setName, _ := cmd.Flags().GetString("name")
	set := &catalog.ScenarioSet{
		Name:        setName,
		TargetName:  target.Name,
		Mode:        mode,
		Summary:     discovery.Summary,
		Expectation: target.Expectation,
		Plans:       plans,
	}
	fmt.Printf("Generated %d scenarios for target %q\n", len(plans), target.Name)
	for index, plan := range plans {
		fmt.Printf("  %d. %s\n", index+1, plan.Name)
		fmt.Printf("     %s\n", plan.Description)
	}

	draftOnly, _ := cmd.Flags().GetBool("draft")
	autoApprove, _ := cmd.Flags().GetBool("yes")
	if !draftOnly {
		approved := autoApprove
		if !autoApprove {
			approved, err = promptConfirm("Approve this scenario set for runs now?", true)
			if err != nil {
				return err
			}
		}
		if approved {
			set.Approved = true
			set.ApprovedAt = time.Now().Format(time.RFC3339)
		}
	}

	if err := catalog.SaveScenarioSet(set); err != nil {
		return err
	}

	replaceDefault, _ := cmd.Flags().GetBool("replace-default")
	if replaceDefault {
		target.ScenarioSet = set.Name
		if err := catalog.SaveTarget(target); err != nil {
			return err
		}
	}

	status := "draft"
	if set.Approved {
		status = "approved"
	}
	fmt.Printf("Saved scenario set %q for target %q (%s)\n", set.Name, target.Name, status)
	return nil
}

func listScenarioSets(cmd *cobra.Command, args []string) error {
	target, err := catalog.LoadTarget(args[0])
	if err != nil {
		return err
	}

	sets, err := catalog.ListScenarioSets(target.Name)
	if err != nil {
		return err
	}
	if len(sets) == 0 {
		fmt.Println("No saved scenarios yet. Generate them with: suitest scenario map <target>")
		return nil
	}

	for _, set := range sets {
		status := "draft"
		if set.Approved {
			status = "approved"
		}
		fmt.Printf("- %s (%d scenarios, %s)\n", set.Name, len(set.Plans), status)
		if set.Expectation != "" {
			fmt.Printf("  expectation: %s\n", set.Expectation)
		}
	}
	return nil
}

func approveScenarioSet(cmd *cobra.Command, args []string) error {
	target, err := catalog.LoadTarget(args[0])
	if err != nil {
		return err
	}

	setName := target.ScenarioSet
	if len(args) > 1 {
		setName = args[1]
	}
	if setName == "" {
		setName = "default"
	}

	set, err := catalog.LoadScenarioSet(target.Name, setName)
	if err != nil {
		return err
	}
	set.Approved = true
	set.ApprovedAt = time.Now().Format(time.RFC3339)
	if err := catalog.SaveScenarioSet(set); err != nil {
		return err
	}
	fmt.Printf("Approved scenario set %q for target %q\n", set.Name, target.Name)
	return nil
}

func resolveTargetMode(target *catalog.Target) string {
	if target.Type == "frontend" {
		return "browser"
	}
	return "api"
}
