package main

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"tailscale.com/client/tailscale/v2"
)

func getAuthKey() (string, error) {
	var err error
	authKey := viper.GetString("tailscale.auth_key")
	if viper.GetString("tailscale.client_id") != "" && viper.GetString("tailscale.client_secret") != "" {
		authKey, err = getOAuthKey()
		if err != nil {
			return "", fmt.Errorf("Error getting OAuth key: %w", err)
		}
	}

	if authKey == "" {
		authKey = viper.GetString("tailscale.auth_key")
	}

	return authKey, nil
}

func getOAuthKey() (string, error) {
	ctx := context.Background()
	oauthConfig := tailscale.OAuthConfig{
		ClientID:     viper.GetString("tailscale.client_id"),
		ClientSecret: viper.GetString("tailscale.client_secret"),
		Scopes:       []string{"auth_keys"},
	}
	tsclient := &tailscale.Client{
		UserAgent: "K8-Proxy",
		Tailnet:   "-",
		HTTP:      oauthConfig.HTTPClient(),
	}

	capabilities := tailscale.KeyCapabilities{}
	capabilities.Devices.Create.Ephemeral = false
	capabilities.Devices.Create.Reusable = false
	capabilities.Devices.Create.Preauthorized = true
	capabilities.Devices.Create.Tags = viper.GetStringSlice("tailscale.tags")

	ckr := tailscale.CreateKeyRequest{
		Capabilities: capabilities,
		Description:  "K8-Proxy",
	}

	authKey, err := tsclient.Keys().Create(ctx, ckr)
	if err != nil {
		return "", fmt.Errorf("Error creating auth key: %w", err)
	}

	return authKey.Key, nil
}
