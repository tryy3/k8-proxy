package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"tailscale.com/tsnet"
)

type ProxyService struct {
	srv        *tsnet.Server
	targetIP   string      // Change to dynamic IP where we continously look at kubernetes
	listenPort []ProxyPort // Maybe multiple ports?
	clientset  *kubernetes.Clientset
}

type ProxyPort struct {
	RemotePort string
	LocalPort  string
}

func NewProxyService(srv *tsnet.Server, clientset *kubernetes.Clientset) *ProxyService {
	return &ProxyService{
		srv:       srv,
		targetIP:  "10.10.0.100",
		clientset: clientset,
		listenPort: []ProxyPort{
			{
				RemotePort: "443",
				LocalPort:  "443",
			},
		},
	}
}

func (p *ProxyService) Start() {
	go func() {
		for {
			pods, err := p.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}
			fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

			// Examples for error handling:
			// - Use helper functions like e.g. errors.IsNotFound()
			// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
			namespace := "default"
			pod := "dnsutils"
			_, err = p.clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				fmt.Printf("Error getting pod %s in namespace %s: %v\n",
					pod, namespace, statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
			}

			time.Sleep(10 * time.Second)
		}
	}()

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
