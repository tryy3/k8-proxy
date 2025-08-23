package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"tailscale.com/tsnet"
)

type ProxyService struct {
	srv              *tsnet.Server
	traefikIP        string // Change to dynamic IP where we continously look at kubernetes
	traefikNamespace string
	listenPort       []ProxyPort // Maybe multiple ports?
	clientset        *kubernetes.Clientset
}

type ProxyPort struct {
	RemotePort string
	LocalPort  string
}

func NewProxyService(srv *tsnet.Server, clientset *kubernetes.Clientset) (*ProxyService, error) {
	var configProxyPort []ProxyPort
	err := viper.UnmarshalKey("proxy.ports", &configProxyPort)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling proxy ports: %w", err)
	}

	return &ProxyService{
		srv:              srv,
		clientset:        clientset,
		traefikNamespace: viper.GetString("traefik.namespace"),
		listenPort:       configProxyPort,
	}, nil
}

func (p *ProxyService) Start() {
	go p.StartTailscaleListener()
}

func (p *ProxyService) StartTailscaleListener() {
	for _, port := range p.listenPort {
		ln, err := p.srv.Listen("tcp", ":"+port.LocalPort)
		if err != nil {
			slog.Error("Tailscale listener error: ", "error", err)
		}
		defer ln.Close()

		slog.Info("Tailscale listener started on port ", "port", port.LocalPort)
		p.handleConnections(ln)
	}
}

func (p *ProxyService) handleConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("Accept error: ", "error", err)
		}

		go p.proxyConnection(conn)
	}
}

func (p *ProxyService) extractPort(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func (p *ProxyService) mapLocalPortToRemotePort(addr string) string {
	localPort := p.extractPort(addr)
	for _, port := range p.listenPort {
		if port.LocalPort == localPort {
			return port.RemotePort
		}
	}
	return ""
}

func (p *ProxyService) getTraefikServiceIP() (string, error) {
	if p.traefikIP != "" {
		return p.traefikIP, nil
	}

	svc, err := p.clientset.CoreV1().Services(p.traefikNamespace).Get(context.Background(), viper.GetString("traefik.service"), metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("Error getting traefik service: %w", err)
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return "", fmt.Errorf("No external IP found for traefik service")
	}

	p.traefikIP = svc.Status.LoadBalancer.Ingress[0].IP
	return p.traefikIP, nil
}

func (p *ProxyService) getBackendConnection(targetPort string) (net.Conn, error) {
	// First check if we have a cached connection
	// If not, attempt to retrieve from kubernetes
	// Then try to connect to server, if failed do 1 more
	// reason being if cached then kubernetes might have switched over
	// if it's first time, then it's fine to try 1 more time
	svcIP, err := p.getTraefikServiceIP()
	if err != nil {
		return nil, fmt.Errorf("Error getting traefik service IP: %w", err)
	}

	// Try to connect to the service
	slog.Info("Dialing traefik service ", "service", svcIP+":"+targetPort)
	conn, err := p.srv.Dial(context.Background(), "tcp", svcIP+":"+targetPort)
	if err != nil {
		// Try one more time
		svcIP, err = p.getTraefikServiceIP()
		slog.Info("Second attempt to get traefik service IP: ", "service", svcIP)
		if err != nil {
			return nil, fmt.Errorf("Error getting traefik service IP: %w", err)
		}

		conn, err = p.srv.Dial(context.Background(), "tcp", svcIP+":"+targetPort)
		if err != nil {
			return nil, fmt.Errorf("Error connecting to traefik service: %w", err)
		}
	}
	return conn, nil
}

func (p *ProxyService) proxyConnection(conn net.Conn) {
	defer conn.Close()
	slog.Info("New connection", "remote", conn.RemoteAddr(), "local", conn.LocalAddr())
	remotePort := p.mapLocalPortToRemotePort(conn.LocalAddr().String())

	backendConn, err := p.getBackendConnection(remotePort)
	if err != nil {
		slog.Error("Error getting backend connection: ", "error", err)
	}
	defer backendConn.Close()

	// Bidirectional proxy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(backendConn, conn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, backendConn)
	}()

	wg.Wait()
}
