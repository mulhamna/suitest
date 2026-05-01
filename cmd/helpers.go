package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var stdinReader = bufio.NewReader(os.Stdin)

func applyRunOverrides(cmd *cobra.Command, cfg *config.Config) error {
	if mode, _ := cmd.Flags().GetString("mode"); mode != "auto" {
		cfg.Mode = mode
	}
	if fix, _ := cmd.Flags().GetBool("fix"); fix {
		cfg.Agent.AutoFix = true
	}
	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		cfg.DryRun = true
	}
	if output, _ := cmd.Flags().GetString("output"); output != "terminal" {
		cfg.Output = output
	}
	if maxRetries, _ := cmd.Flags().GetInt("max-retries"); maxRetries != 3 {
		cfg.Agent.MaxRetries = maxRetries
	}
	if concurrency, _ := cmd.Flags().GetInt("concurrency"); concurrency != 4 {
		cfg.Agent.Concurrency = concurrency
	}
	if p := viper.GetString("provider"); p != "" && p != "auto" {
		cfg.DefaultProvider = p
	}
	if m := viper.GetString("model"); m != "" {
		if cfg.Providers[cfg.DefaultProvider] == nil {
			cfg.Providers[cfg.DefaultProvider] = &config.ProviderConfig{}
		}
		cfg.Providers[cfg.DefaultProvider].Model = m
	}
	if bu := viper.GetString("base_url"); bu != "" {
		if cfg.Providers[cfg.DefaultProvider] == nil {
			cfg.Providers[cfg.DefaultProvider] = &config.ProviderConfig{}
		}
		cfg.Providers[cfg.DefaultProvider].BaseURL = bu
	}
	return nil
}

func initProvider(cfg *config.Config) (providers.Provider, error) {
	provider, err := providers.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize provider: %w", err)
	}
	return provider, nil
}

func resolveRunTarget(name string) (*catalog.Target, bool) {
	target, err := catalog.LoadTarget(name)
	if err != nil {
		return nil, false
	}
	return target, true
}

func resolvePath(value string) (string, error) {
	absPath, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", absPath)
	}
	return absPath, nil
}

func confirmSavedRun(ctx context.Context, target *catalog.Target, set *catalog.ScenarioSet, assumeYes bool) (bool, error) {
	if assumeYes {
		return true, nil
	}

	fmt.Println()
	fmt.Printf("Target:       %s\n", target.Name)
	fmt.Printf("Type:         %s\n", target.Type)
	fmt.Printf("Path:         %s\n", target.Path)
	if target.URL != "" {
		fmt.Printf("URL:          %s\n", target.URL)
	}
	if target.Expectation != "" {
		fmt.Printf("Expectation:  %s\n", target.Expectation)
	}
	if set != nil {
		fmt.Printf("Scenario set: %s\n", set.Name)
		if set.Approved {
			fmt.Printf("Approval:     approved")
			if set.ApprovedAt != "" {
				fmt.Printf(" (%s)", set.ApprovedAt)
			}
			fmt.Println()
		} else {
			fmt.Println("Approval:     draft")
		}
		fmt.Printf("Scenarios:    %d\n", len(set.Plans))
		for index, plan := range set.Plans {
			fmt.Printf("  %d. %s\n", index+1, plan.Name)
		}
	}
	fmt.Print("\nProceed? [Y/n] ")

	answer, err := stdinReader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" || answer == "y" || answer == "yes" {
		return true, nil
	}
	return false, nil
}

func promptText(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	answer, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultValue, nil
	}
	return answer, nil
}

func promptChoice(label string, options []string, defaultValue string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options available for %s", label)
	}
	valid := make(map[string]struct{}, len(options))
	for _, option := range options {
		valid[strings.ToLower(option)] = struct{}{}
	}
	for {
		fmt.Printf("%s [%s]\n", label, strings.Join(options, "/"))
		answer, err := promptText("> ", defaultValue)
		if err != nil {
			return "", err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		if _, ok := valid[answer]; ok {
			return answer, nil
		}
		fmt.Printf("Please choose one of: %s\n", strings.Join(options, ", "))
	}
}

func promptConfirm(label string, defaultYes bool) (bool, error) {
	suffix := "[y/N]"
	defaultValue := "n"
	if defaultYes {
		suffix = "[Y/n]"
		defaultValue = "y"
	}
	for {
		answer, err := promptText(fmt.Sprintf("%s %s", label, suffix), defaultValue)
		if err != nil {
			return false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Println("Please answer yes or no.")
	}
}
