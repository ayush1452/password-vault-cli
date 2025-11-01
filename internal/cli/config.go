package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
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
			return runConfigGetAll()
		}
		return runConfigGet(args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(args[0], args[1])
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigPath()
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

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get configuration value(s)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runConfigGetAll()
			}
			return runConfigGet(args[0])
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	}

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPath()
		},
	}

	cmd.AddCommand(getCmd)
	cmd.AddCommand(setCmd)
	cmd.AddCommand(pathCmd)

	return cmd
}

func runConfigGetAll() error {
	fmt.Printf("Configuration file: %s\n\n", cfgFile)
	fmt.Printf("vault_path: %s\n", cfg.VaultPath)
	fmt.Printf("default_profile: %s\n", cfg.DefaultProfile)
	fmt.Printf("auto_lock_ttl: %s\n", cfg.AutoLockTTL)
	fmt.Printf("clipboard_ttl: %s\n", cfg.ClipboardTTL)
	fmt.Printf("session_timeout: %d\n", cfg.Security.SessionTimeout)
	fmt.Printf("output_format: %s\n", cfg.OutputFormat)
	fmt.Printf("show_passwords: %t\n", cfg.ShowPasswords)
	fmt.Printf("confirm_destructive: %t\n", cfg.ConfirmDestructive)
	fmt.Printf("kdf.memory: %d\n", cfg.KDF.Memory)
	fmt.Printf("kdf.iterations: %d\n", cfg.KDF.Iterations)
	fmt.Printf("kdf.parallelism: %d\n", cfg.KDF.Parallelism)
	return nil
}

func runConfigGet(key string) error {
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")

	switch normalized {
	case "vault_path":
		fmt.Println(cfg.VaultPath)
	case "default_profile":
		fmt.Println(cfg.DefaultProfile)
	case "auto_lock_ttl":
		fmt.Println(cfg.AutoLockTTL)
	case "clipboard_ttl":
		fmt.Println(cfg.ClipboardTTL)
	case "session_timeout":
		fmt.Println(cfg.Security.SessionTimeout)
	case "output_format":
		fmt.Println(cfg.OutputFormat)
	case "show_passwords":
		fmt.Println(cfg.ShowPasswords)
	case "confirm_destructive":
		fmt.Println(cfg.ConfirmDestructive)
	case "kdf.memory":
		fmt.Println(cfg.KDF.Memory)
	case "kdf.iterations":
		fmt.Println(cfg.KDF.Iterations)
	case "kdf.parallelism":
		fmt.Println(cfg.KDF.Parallelism)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}

func runConfigSet(key, value string) error {
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

	fmt.Printf("✓ Configuration updated: %s = %s\n", key, value)
	return nil
}

func runConfigPath() error {
	fmt.Println(cfgFile)
	return nil
}
