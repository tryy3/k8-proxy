package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
)

func initConfig() {
	viper.SetDefault("traefik.service", "traefik")
	viper.SetDefault("traefik.namespace", "kube-system")

	viper.SetDefault("tailscale.client_id", "")
	viper.SetDefault("tailscale.client_secret", "")
	viper.SetDefault("tailscale.tags", []string{"tag:example"})

	viper.SetDefault("proxy.ports", []ProxyPort{
		{
			RemotePort: "443",
			LocalPort:  "443",
		},
		{
			RemotePort: "80",
			LocalPort:  "80",
		},
	})

	if home := homedir.HomeDir(); home != "" {
		viper.SetDefault("kube-config", filepath.Join(home, ".kube", "config"))
	} else {
		viper.SetDefault("kube-config", ".kube")
	}

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/config")
	viper.AddConfigPath(".") // optionally look for config in the working directory// Try to read the config file

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create it with default values
			slog.Info("Config file not found, creating default config...")
			if err := createDefaultConfig(); err != nil {
				slog.Error("Error creating default config", "error", err)
				os.Exit(1)
			}
		} else {
			// Config file was found but another error was produced
			slog.Error("Fatal error reading config file", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("Using config file", "file", viper.ConfigFileUsed())
	}
}

func createDefaultConfig() error {
	// Write the current configuration (with defaults) to a file
	if err := viper.WriteConfigAs("config.json"); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	slog.Info("Created default config file", "file", "config.json")
	return nil
}
