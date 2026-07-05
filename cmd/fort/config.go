package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type config struct {
	VaultDir string `json:"vault_dir"`
	APIURL   string `json:"api_url"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage fortbyte configuration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.VaultDir == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Config file not found. Using default vault directory.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Vault directory: %s\n", cfg.VaultDir)
		}
		if cfg.APIURL != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "API URL: %s\n", cfg.APIURL)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Default vault directory: %s\n", defaultVaultDir())
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		cfg, _ := loadConfig()
		switch key {
		case "vault-dir":
			abs, err := filepath.Abs(value)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}
			cfg.VaultDir = abs
			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Vault directory set to: %s\n", abs)
		case "api-url":
			cfg.APIURL = value
			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "API URL set to: %s\n", value)
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		switch key {
		case "vault-dir":
			if cfg.VaultDir == "" {
				fmt.Fprintln(cmd.OutOrStdout(), defaultVaultDir())
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), cfg.VaultDir)
			}
		case "api-url":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.APIURL)
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		return nil
	},
}

func defaultVaultDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, fortDir)
}

func configPath() string {
	return filepath.Join(defaultVaultDir(), "config.json")
}

func loadConfig() (config, error) {
	var cfg config
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid config file: %w", err)
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	path := configPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd, configGetCmd)
}
