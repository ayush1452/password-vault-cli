package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

type contextKey string

const (
	configKey contextKey = "config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage vault configuration",
	Long: `Manage vault configuration settings.

You can view, set, or get individual configuration values.
Configuration is stored in ~/.config/vault/config.yaml by default.

Example:
  vault config path                      # Show config file path
  vault config get clipboard_ttl         # Get clipboard timeout
  vault config set clipboard_ttl 60s     # Set clipboard timeout
  vault config get                       # Show all configuration`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value(s)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runConfigGetAll(cmd)
		}
		return runConfigGet(cmd, args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(cmd, args[0], args[1])
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigPath(cmd)
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
}

// NewConfigCommand creates a new config command for testing
func NewConfigCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage vault configuration",
		Long: `Manage vault configuration settings.

You can view, set, or get individual configuration values.
Configuration is stored in ~/.config/vault/config.yaml by default.

Example:
  vault config path                      # Show config file path
  vault config get clipboard_ttl         # Get clipboard timeout
  vault config set clipboard_ttl 60s     # Set clipboard timeout
  vault config get                       # Show all configuration`,
	}

	// Set the config in the command's context
	ctx := context.WithValue(context.Background(), configKey, cfg)
	cmd.SetContext(ctx)

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get configuration value(s)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runConfigGetAll(cmd)
			}
			return runConfigGet(cmd, args[0])
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(cmd, args[0], args[1])
		},
	}

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPath(cmd)
		},
	}

	cmd.AddCommand(getCmd)
	cmd.AddCommand(setCmd)
	cmd.AddCommand(pathCmd)

	return cmd
}

func runConfigGetAll(cmd *cobra.Command) error {
	var out io.Writer = os.Stdout
	if cmd != nil {
		out = cmd.OutOrStdout()
	}

	// Helper function to write formatted output and check for errors
	writeOutput := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(out, format, args...)
		return err
	}

	// Write configuration with error checking
	if err := writeOutput("Configuration file: %s\n\n", cfgFile); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	// Create a slice of write operations to execute
	writeOps := []struct {
		format string
		value  interface{}
	}{
		{"vault_path: %s\n", cfg.VaultPath},
		{"default_profile: %s\n", cfg.DefaultProfile},
		{"auto_lock_ttl: %s\n", cfg.AutoLockTTL},
		{"clipboard_ttl: %s\n", cfg.ClipboardTTL},
		{"session_timeout: %d\n", cfg.Security.SessionTimeout},
		{"output_format: %s\n", cfg.OutputFormat},
		{"show_passwords: %t\n", cfg.ShowPasswords},
		{"confirm_destructive: %t\n", cfg.ConfirmDestructive},
		{"kdf.memory: %d\n", cfg.KDF.Memory},
		{"kdf.iterations: %d\n", cfg.KDF.Iterations},
		{"kdf.parallelism: %d\n", cfg.KDF.Parallelism},
	}

	// Execute all write operations
	for _, op := range writeOps {
		if err := writeOutput(op.format, op.value); err != nil {
			return fmt.Errorf("failed to write configuration: %w", err)
		}
	}

	return nil
}

func runConfigGet(cmd *cobra.Command, key string) error {
	var out io.Writer = os.Stdout
	if cmd != nil {
		out = cmd.OutOrStdout()
	}

	// Helper function to write output and check for errors
	writeOutput := func(v interface{}) error {
		_, err := fmt.Fprintln(out, v)
		return err
	}

	// Get config from command context or use global config
	var currentCfg *config.Config
	if cmd != nil && cmd.Root() != nil {
		if c, ok := cmd.Root().Context().Value(configKey).(*config.Config); ok && c != nil {
			currentCfg = c
		}
	}
	if currentCfg == nil {
		currentCfg = cfg
	}

	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")

	switch normalized {
	case "vault_path":
		if err := writeOutput(currentCfg.VaultPath); err != nil {
			return fmt.Errorf("failed to write vault path: %w", err)
		}
	case "default_profile":
		if err := writeOutput(currentCfg.DefaultProfile); err != nil {
			return fmt.Errorf("failed to write default profile: %w", err)
		}
	case "auto_lock_ttl":
		if err := writeOutput(currentCfg.AutoLockTTL); err != nil {
			return fmt.Errorf("failed to write auto lock TTL: %w", err)
		}
	case "clipboard_ttl":
		if err := writeOutput(currentCfg.ClipboardTTL); err != nil {
			return fmt.Errorf("failed to write clipboard TTL: %w", err)
		}
	case "session_timeout":
		if err := writeOutput(currentCfg.Security.SessionTimeout); err != nil {
			return fmt.Errorf("failed to write session timeout: %w", err)
		}
	case "output_format":
		if err := writeOutput(currentCfg.OutputFormat); err != nil {
			return fmt.Errorf("failed to write output format: %w", err)
		}
	case "show_passwords":
		if err := writeOutput(currentCfg.ShowPasswords); err != nil {
			return fmt.Errorf("failed to write show passwords setting: %w", err)
		}
	case "confirm_destructive":
		if err := writeOutput(currentCfg.ConfirmDestructive); err != nil {
			return fmt.Errorf("failed to write confirm destructive setting: %w", err)
		}
	case "kdf.memory":
		if err := writeOutput(currentCfg.KDF.Memory); err != nil {
			return fmt.Errorf("failed to write KDF memory: %w", err)
		}
	case "kdf.iterations":
		if err := writeOutput(currentCfg.KDF.Iterations); err != nil {
			return fmt.Errorf("failed to write KDF iterations: %w", err)
		}
	case "kdf.parallelism":
		if err := writeOutput(currentCfg.KDF.Parallelism); err != nil {
			return fmt.Errorf("failed to write KDF parallelism: %w", err)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	
	return nil
}

func runConfigSet(cmd *cobra.Command, key, value string) error {
	var out io.Writer = os.Stdout
	if cmd != nil {
		out = cmd.OutOrStdout()
	}
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")

	switch normalized {
	case "vault_path":
		cfg.VaultPath = value
	case "default_profile":
		cfg.DefaultProfile = value
	case "auto_lock_ttl":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		cfg.AutoLockTTL = duration
	case "clipboard_ttl":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		cfg.ClipboardTTL = duration
	case "session_timeout":
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid timeout value: %w", err)
		}
		if seconds <= 0 {
			return fmt.Errorf("invalid timeout value: must be positive seconds")
		}
		cfg.Security.SessionTimeout = seconds
	case "output_format":
		if value != "table" && value != "json" && value != "yaml" {
			return fmt.Errorf("invalid output format: %s (valid: table, json, yaml)", value)
		}
		cfg.OutputFormat = value
	case "show_passwords":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		cfg.ShowPasswords = boolVal
	case "confirm_destructive":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		cfg.ConfirmDestructive = boolVal
	case "kdf.memory":
		intVal, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid integer value: %w", err)
		}
		cfg.KDF.Memory = uint32(intVal)
	case "kdf.iterations":
		intVal, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid integer value: %w", err)
		}
		cfg.KDF.Iterations = uint32(intVal)
	case "kdf.parallelism":
		intVal, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid integer value: %w", err)
		}
		cfg.KDF.Parallelism = uint8(intVal)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Save configuration
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintf(out, "✓ Configuration updated: %s = %s\n", key, value)
	return nil
}

func runConfigPath(cmd *cobra.Command) error {
	var out io.Writer = os.Stdout
	if cmd != nil {
		out = cmd.OutOrStdout()
	}
	
	// Write the config file path and check for errors
	if _, err := fmt.Fprintln(out, cfgFile); err != nil {
		return fmt.Errorf("failed to write config file path: %w", err)
	}
	
	return nil
}
