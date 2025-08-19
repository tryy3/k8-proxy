package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"tailscale.com/client/tailscale/v2"
)

type ConfigData struct {
	AuthKey      string
	ClientID     string
	ClientSecret string
}

var Config *ConfigData

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	Config = &ConfigData{
		AuthKey:      os.Getenv("TAILSCALE_AUTH_KEY"),
		ClientID:     os.Getenv("TAILSCALE_CLIENT_ID"),
		ClientSecret: os.Getenv("TAILSCALE_CLIENT_SECRET"),
	}
}

func getAuthKey() string {
	authKey := Config.AuthKey
	if Config.ClientID != "" && Config.ClientSecret != "" {
		authKey = getOAuthKey()
	}

	if authKey == "" {
		authKey = Config.AuthKey
	}

	return authKey
}

func getOAuthKey() string {
	ctx := context.Background()
	oauthConfig := tailscale.OAuthConfig{
		ClientID:     Config.ClientID,
		ClientSecret: Config.ClientSecret,
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
	capabilities.Devices.Create.Tags = []string{"tag:example"}

	ckr := tailscale.CreateKeyRequest{
		Capabilities: capabilities,
		Description:  "K8-Proxy",
	}

	authKey, err := tsclient.Keys().Create(ctx, ckr)
	if err != nil {
		log.Fatal("Error creating auth key: ", err)
	}

	return authKey.Key
}
