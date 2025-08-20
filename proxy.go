package main

import (
	"context"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"tailscale.com/tsnet"
)

type ProxyService struct {
	srv        *tsnet.Server
	targetIP   string      // Change to dynamic IP where we continously look at kubernetes
	listenPort []ProxyPort // Maybe multiple ports?
}

type ProxyPort struct {
	RemotePort string
	LocalPort  string
}

func NewProxyService(srv *tsnet.Server) *ProxyService {
	return &ProxyService{
		srv:      srv,
		targetIP: "10.10.0.100",
		listenPort: []ProxyPort{
			{
				RemotePort: "443",
				LocalPort:  "443",
			},
		},
	}
}

func (p *ProxyService) Start() {
	go p.StartTailscaleListener()
}

func (p *ProxyService) StartTailscaleListener() {
	for _, port := range p.listenPort {
		ln, err := p.srv.Listen("tcp", ":"+port.LocalPort)
		if err != nil {
			log.Fatal("Tailscale listener error: ", err)
		}
		defer ln.Close()

		log.Println("Tailscale listener started on port ", port.LocalPort)
		p.handleConnections(ln)
	}
}

func (p *ProxyService) handleConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
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

func (p *ProxyService) proxyConnection(conn net.Conn) {
	defer conn.Close()
	log.Println("New connection from ", conn.RemoteAddr(), conn.LocalAddr())
	remotePort := p.mapLocalPortToRemotePort(conn.LocalAddr().String())

	// TODO: maybe use connection check here, cache latest connection and if failed
	// retrieve new IP from kubernetes
	backendConn, err := p.srv.Dial(context.Background(), "tcp", p.targetIP+":"+remotePort)
	if err != nil {
		log.Fatal("Dial error: ", err)
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
