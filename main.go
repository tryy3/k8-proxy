// This program demonstrates how to use tsnet as a library.
package main

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"tailscale.com/tsnet"
)

func init() {
	initConfig()
}

func main() {
	kubeConfigPath := viper.GetString("kube-config")

	slog.Info("kubeConfigPath", "path", kubeConfigPath)
	authKey, err := getAuthKey()
	if err != nil {
		slog.Error("Error getting auth key", "error", err)
		os.Exit(1)
	}

	srv := new(tsnet.Server)
	srv.AuthKey = authKey

	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		slog.Error("Error building kube config", "error", err)
		os.Exit(1)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		slog.Error("Error creating clientset", "error", err)
		os.Exit(1)
	}

	proxyService, err := NewProxyService(srv, clientset)
	if err != nil {
		slog.Error("Error creating proxy service", "error", err)
		os.Exit(1)
	}
	proxyService.Start()

	select {}
}
